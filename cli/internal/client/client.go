package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	// ClientHeader is sent as X-Thask-Client on every outbound request so the
	// backend can record provenance (cli vs mcp vs web) in audit_log.
	// Format: "thask-cli/<ver>" or "thask-mcp/<ver> model=... session=...".
	ClientHeader string
}

type APIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body any) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	if c.ClientHeader != "" {
		req.Header.Set("X-Thask-Client", c.ClientHeader)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := apiResp.Error
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		if resp.StatusCode >= 500 {
			return nil, &ServerError{Message: msg, Code: resp.StatusCode}
		}
		return nil, &ClientError{Message: msg, Code: resp.StatusCode}
	}

	return apiResp.Data, nil
}

func (c *Client) Get(path string) (json.RawMessage, error) {
	return c.do("GET", path, nil)
}

func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.do("POST", path, body)
}

func (c *Client) Put(path string, body any) (json.RawMessage, error) {
	return c.do("PUT", path, body)
}

func (c *Client) Patch(path string, body any) (json.RawMessage, error) {
	return c.do("PATCH", path, body)
}

func (c *Client) Delete(path string) (json.RawMessage, error) {
	return c.do("DELETE", path, nil)
}

func (c *Client) GetRaw(path, accept string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.ClientHeader != "" {
		req.Header.Set("X-Thask-Client", c.ClientHeader)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.Error != "" {
			msg = apiResp.Error
		} else {
			var v1Resp struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(respBody, &v1Resp); err == nil && v1Resp.Error.Message != "" {
				msg = v1Resp.Error.Message
			}
		}
		if resp.StatusCode >= 500 {
			return nil, "", &ServerError{Message: msg, Code: resp.StatusCode}
		}
		return nil, "", &ClientError{Message: msg, Code: resp.StatusCode}
	}

	return respBody, resp.Header.Get("Content-Type"), nil
}

// Error types for exit code differentiation
type ClientError struct {
	Message string
	Code    int
}

func (e *ClientError) Error() string { return e.Message }

type ServerError struct {
	Message string
	Code    int
}

func (e *ServerError) Error() string { return e.Message }

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch err.(type) {
	case *ServerError:
		return 2
	default:
		return 1
	}
}
