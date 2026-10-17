package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPServer(t *testing.T) {
	server := NewServer()

	// 1. Test initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	var out bytes.Buffer
	if err := server.Serve(strings.NewReader(initReq), &out); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", resp.Error)
	}

	// 2. Test tools/list
	toolsReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	out.Reset()
	if err := server.Serve(strings.NewReader(toolsReq), &out); err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}

	var toolsResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Tools []ToolDefinition `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &toolsResp); err != nil {
		t.Fatalf("invalid tools response: %v", err)
	}
	if len(toolsResp.Result.Tools) < 4 {
		t.Fatalf("expected at least 4 tools, got %d", len(toolsResp.Result.Tools))
	}
}
