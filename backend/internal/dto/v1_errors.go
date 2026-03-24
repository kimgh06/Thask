package dto

// V1 structured error codes
const (
	ErrCodeAuthRequired = "AUTHENTICATION_REQUIRED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeRateLimited  = "RATE_LIMITED"
	ErrCodeBodyTooLarge = "BODY_TOO_LARGE"
	ErrCodeInternal     = "INTERNAL_ERROR"
)

// V1ErrorResponse is the structured error envelope for /api/v1/ routes.
type V1ErrorResponse struct {
	Error V1Error `json:"error"`
}

// V1Error is the structured error body.
type V1Error struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details []V1ErrorDetail `json:"details,omitempty"`
}

// V1ErrorDetail provides field-level validation error details.
type V1ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// V1Err creates a V1ErrorResponse from an HTTP status code and message.
func V1Err(statusCode int, message string, details ...V1ErrorDetail) *V1ErrorResponse {
	code := statusToCode(statusCode)
	return &V1ErrorResponse{
		Error: V1Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

func statusToCode(status int) string {
	switch {
	case status == 400:
		return ErrCodeValidation
	case status == 401:
		return ErrCodeAuthRequired
	case status == 403:
		return ErrCodeForbidden
	case status == 404:
		return ErrCodeNotFound
	case status == 409:
		return ErrCodeConflict
	case status == 413:
		return ErrCodeBodyTooLarge
	case status == 429:
		return ErrCodeRateLimited
	case status >= 500:
		return ErrCodeInternal
	default:
		return ErrCodeInternal
	}
}
