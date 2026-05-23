package mcp

import "testing"

func TestHandleRequestSkipsInitializedNotification(t *testing.T) {
	resp, ok := handleRequest(nil, "", "", Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})

	if ok {
		t.Fatalf("expected initialized notification to produce no response, got %+v", resp)
	}
}

func TestHandleRequestSkipsUnknownNotification(t *testing.T) {
	resp, ok := handleRequest(nil, "", "", Request{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
	})

	if ok {
		t.Fatalf("expected unknown notification to produce no response, got %+v", resp)
	}
}

func TestHandleRequestReturnsErrorForUnknownRequest(t *testing.T) {
	resp, ok := handleRequest(nil, "", "", Request{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "unknown/method",
	})

	if !ok {
		t.Fatal("expected unknown request with id to produce an error response")
	}
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected method-not-found error, got %+v", resp)
	}
}
