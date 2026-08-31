package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/drivers/somnia"
	"github.com/nathfavour/beaverish/pkg/engine"
)

func TestMCPServer_InitializeAndListTools(t *testing.T) {
	cfg := config.DefaultConfig()
	adapter, err := somnia.NewSomniaAdapter(cfg, nil, nil)
	if err != nil {
		t.Fatalf("Failed to init adapter: %v", err)
	}

	eng := engine.NewEngine(cfg, adapter)
	handler := NewHandler(cfg, eng)

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_markets","arguments":{}}}` + "\n"

	inBuf := bytes.NewBufferString(input)
	outBuf := &bytes.Buffer{}

	server := NewServer(inBuf, outBuf, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server error: %v", err)
	}

	outStr := outBuf.String()
	lines := strings.Split(strings.TrimSpace(outStr), "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 JSON-RPC responses, got %d. Output: %s", len(lines), outStr)
	}

	// Verify initialize response
	var initResp Response
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("Failed to unmarshal init response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("Initialize returned error: %v", initResp.Error)
	}

	// Verify tools/list
	var listResp Response
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("Failed to unmarshal tools/list response: %v", err)
	}

	// Verify tools/call get_markets
	var callResp Response
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("Failed to unmarshal tools/call response: %v", err)
	}
}
