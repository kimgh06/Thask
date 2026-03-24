package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
)

const ContextIsV1 = "v1"

// V1ResponseWrapper intercepts handler responses on /api/v1/ routes.
// - Sets c.Set("v1", true) so handlers can detect v1 context.
// - For error responses (non-2xx), transforms {"error":"msg"} to structured v1 format.
// - For success responses (2xx), passes through directly (zero copy, no buffering).
func V1ResponseWrapper() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextIsV1, true)
			c.Response().Header().Set("X-API-Version", "v1")

			// Replace writer with a smart writer that buffers only on error status
			origWriter := c.Response().Writer
			sw := &smartWriter{
				orig:   origWriter,
				header: origWriter.Header(),
			}
			c.Response().Writer = sw

			err := next(c)

			// If the smart writer buffered an error response, transform it
			if sw.buffered && sw.buf != nil {
				body := sw.buf.Bytes()
				status := sw.statusCode

				var origResp dto.Response
				if jsonErr := json.Unmarshal(body, &origResp); jsonErr == nil && origResp.Error != "" {
					v1Err := dto.V1Err(status, origResp.Error)
					transformed, _ := json.Marshal(v1Err)
					origWriter.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					origWriter.WriteHeader(status)
					origWriter.Write(transformed)
					return err
				}

				// Not a dto.Err response, pass through as-is
				origWriter.WriteHeader(status)
				origWriter.Write(body)
			}

			return err
		}
	}
}

// smartWriter passes through 2xx responses directly to the original writer.
// For non-2xx responses, it buffers the body for transformation.
type smartWriter struct {
	orig       http.ResponseWriter
	header     http.Header
	buf        *bytes.Buffer
	buffered   bool
	statusCode int
	decided    bool
}

func (w *smartWriter) Header() http.Header {
	return w.header
}

func (w *smartWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.decided = true
	if statusCode >= 200 && statusCode < 300 {
		// Success — pass through directly
		w.buffered = false
		w.orig.WriteHeader(statusCode)
	} else {
		// Error — buffer for transformation
		w.buffered = true
		w.buf = new(bytes.Buffer)
	}
}

func (w *smartWriter) Write(b []byte) (int, error) {
	if !w.decided {
		// WriteHeader not called yet — default to 200 (success, pass through)
		w.WriteHeader(http.StatusOK)
	}
	if w.buffered {
		return w.buf.Write(b)
	}
	return w.orig.Write(b)
}

// Flush implements http.Flusher for streaming support (e.g., SSE).
func (w *smartWriter) Flush() {
	if f, ok := w.orig.(http.Flusher); ok {
		f.Flush()
	}
}
