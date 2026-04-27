// SPDX-License-Identifier: MIT

//go:build e2e

package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	orkmcp "github.com/vijay431/orkestra/internal/mcp"
	"github.com/vijay431/orkestra/internal/testutil"
	"github.com/vijay431/orkestra/internal/ticket"
)

func startTestServer(t *testing.T, projectID string) string {
	t.Helper()
	return startTestServerWithToken(t, projectID, "")
}

func startTestServerWithToken(t *testing.T, projectID, token string) string {
	t.Helper()
	db := testutil.NewTestDB(t)
	log := testutil.NopLogger()
	svc := ticket.NewService(db, projectID, log)

	port := testutil.FreePort(t)
	cfg := orkmcp.Config{
		ProjectID: projectID,
		Port:      fmt.Sprintf("%d", port),
		BindAddr:  "127.0.0.1",
		MCPToken:  token,
	}

	srv := orkmcp.NewServer(cfg, svc, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Start(ctx) }()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server did not become healthy within 5s")
	return ""
}

// mcpClient is a minimal MCP-over-SSE client for testing.
type mcpClient struct {
	base    string
	token   string
	msgURL  string
	mu      sync.Mutex
	pending map[int64]chan json.RawMessage
	idSeq   atomic.Int64
}

func newMCPClient(t *testing.T, base, token string) *mcpClient {
	t.Helper()
	c := &mcpClient{
		base:    base,
		token:   token,
		pending: make(map[int64]chan json.RawMessage),
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/sse", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	endpointReady := make(chan struct{})
	go c.readSSE(resp.Body, endpointReady)

	select {
	case <-endpointReady:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SSE endpoint event")
	}

	// MCP initialization handshake
	initID := c.nextID()
	initCh := c.registerPending(initID)
	c.postJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      initID,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "e2e-test", "version": "1.0"},
		},
	})
	select {
	case <-initCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initialize response")
	}

	// Notify server that client is ready
	c.postJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})

	return c
}

func (c *mcpClient) readSSE(body io.Reader, endpointReady chan struct{}) {
	scanner := bufio.NewScanner(body)
	var eventType, data string
	once := sync.Once{}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			eventType = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		case line == "":
			c.handleEvent(eventType, data, func() {
				once.Do(func() { close(endpointReady) })
			})
			eventType, data = "", ""
		}
	}
}

func (c *mcpClient) handleEvent(eventType, data string, notifyEndpoint func()) {
	switch eventType {
	case "endpoint":
		if data != "" {
			if strings.HasPrefix(data, "/") {
				c.msgURL = c.base + data
			} else {
				c.msgURL = data
			}
			notifyEndpoint()
		}
	case "message":
		var rpc struct {
			ID     *int64          `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &rpc); err != nil || rpc.ID == nil {
			return
		}
		c.mu.Lock()
		ch, ok := c.pending[*rpc.ID]
		if ok {
			delete(c.pending, *rpc.ID)
		}
		c.mu.Unlock()
		if !ok {
			return
		}
		if rpc.Result != nil {
			ch <- rpc.Result
		} else {
			ch <- rpc.Error
		}
	}
}

func (c *mcpClient) nextID() int64 {
	return c.idSeq.Add(1)
}

func (c *mcpClient) registerPending(id int64) chan json.RawMessage {
	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	return ch
}

func (c *mcpClient) postJSON(t *testing.T, body any) {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.msgURL, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", c.msgURL, err)
	}
	resp.Body.Close()
}

// CallTool sends an MCP tool call and returns the TOON text from the response.
func (c *mcpClient) CallTool(t *testing.T, name string, args map[string]any) string {
	t.Helper()
	id := c.nextID()
	ch := c.registerPending(id)
	c.postJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	})
	select {
	case raw := <-ch:
		var result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("parse tool result for %q: %v (raw: %s)", name, err, raw)
		}
		if len(result.Content) == 0 {
			return ""
		}
		return result.Content[0].Text
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout waiting for tool %q response", name)
		return ""
	}
}

// extractTOONField extracts the value of a field (e.g. "ua:") from a TOON string.
func extractTOONField(toon, prefix string) string {
	idx := strings.Index(toon, prefix)
	if idx < 0 {
		return ""
	}
	rest := toon[idx+len(prefix):]
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
