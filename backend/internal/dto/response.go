package dto

type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

func OK(data any) *Response {
	return &Response{Data: data}
}

func Err(message string) *Response {
	return &Response{Error: message}
}
