package middleware

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

// Idempotency middleware deduplicates POST/PATCH/DELETE requests on v1 routes
// when the Idempotency-Key header is present.
// If the key was seen before (within 24h TTL), returns the cached response.
// If not, proceeds with the handler and caches the response.
// Idempotency is opt-in: requests without the header proceed normally.
func Idempotency(repo *repository.IdempotencyRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only apply to mutating methods
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			idempKey := c.Request().Header.Get("Idempotency-Key")
			if idempKey == "" {
				return next(c) // no idempotency key, proceed normally
			}
			if len(idempKey) > 256 {
				return c.JSON(http.StatusBadRequest, dto.Err("Idempotency-Key must be 256 characters or fewer"))
			}

			apiKeyID, _ := c.Get(ContextAPIKeyID).(string)
			if apiKeyID == "" {
				return next(c) // cookie auth, idempotency only for API keys
			}

			ctx := c.Request().Context()
			path := c.Request().URL.Path

			// Atomic claim — prevents race conditions between concurrent identical requests
			entry, claimed, err := repo.TryClaim(ctx, idempKey, apiKeyID, method, path)
			if err != nil {
				slog.Warn("idempotency claim failed", "key", idempKey, "error", err)
				return next(c) // fallback to non-idempotent
			}

			if !claimed && entry != nil {
				// Replay cached response
				if entry.Method != method || entry.Path != path {
					return c.JSON(http.StatusUnprocessableEntity, dto.Err("Idempotency key already used for a different request"))
				}
				c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				c.Response().Header().Set("X-Idempotency-Replayed", "true")
				c.Response().WriteHeader(entry.StatusCode)
				c.Response().Write(entry.Response)
				return nil
			}

			// Claimed — capture the response
			origWriter := c.Response().Writer
			buf := new(bytes.Buffer)
			capWriter := &captureWriter{
				ResponseWriter: origWriter,
				buf:            buf,
			}
			c.Response().Writer = capWriter

			handlerErr := next(c)

			// Update the claimed key with the actual response (cap at 64KB)
			status := c.Response().Status
			body := buf.Bytes()
			const maxCacheSize = 65536
			if len(body) > 0 && len(body) <= maxCacheSize {
				if storeErr := repo.UpdateResponse(ctx, idempKey, apiKeyID, status, body); storeErr != nil {
					slog.Warn("failed to update idempotency response", "key", idempKey, "error", storeErr)
				}
			}

			return handlerErr
		}
	}
}

// captureWriter writes to both the original writer and a buffer.
type captureWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.buf.Write(b) // capture
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// Flush implements http.Flusher for streaming support.
func (w *captureWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
