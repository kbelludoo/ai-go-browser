package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/resolver"
	"ai-go-browser/pkg/search"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a standard JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition represents an MCP tool schema.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Server provides an MCP (Model Context Protocol) server over standard I/O.
type Server struct {
	mu       sync.Mutex
	browser  *browser.Browser
	resolver *resolver.SelfHealingResolver
}

// NewServer creates a new MCP server wrapping the AI Go Browser engine.
func NewServer() *Server {
	b := browser.New([]string{"*"}, true)
	return &Server{
		browser:  b,
		resolver: resolver.NewSelfHealingResolver(),
	}
}

// Tools returns the list of available MCP tools.
func (s *Server) Tools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "browser_navigate",
			Description: "Navigate to any URL using the ultra-lightweight AI Go Browser (<10MB RAM).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The destination URL to navigate to.",
					},
					"render_js": map[string]any{
						"type":        "boolean",
						"description": "Whether to render dynamic JavaScript (optional, default false).",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "browser_snapshot",
			Description: "Capture a token-optimized Markdown snapshot with deterministic #ref action markers (88-92% fewer tokens than raw HTML).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"compact": map[string]any{
						"type":        "boolean",
						"description": "If true, returns only essential text and primary interactive targets.",
					},
				},
			},
		},
		{
			Name:        "browser_click",
			Description: "Click an interactive element by reference ID (#ref:N) with automatic self-healing fallback.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref": map[string]any{
						"type":        "string",
						"description": "The reference marker (e.g., '1', '#ref:1') or selector.",
					},
					"fallback_text": map[string]any{
						"type":        "string",
						"description": "Expected text of the element in case the DOM shifted (self-healing).",
					},
				},
				"required": []string{"ref"},
			},
		},
		{
			Name:        "browser_type",
			Description: "Type text into an input or textarea field by reference ID (#ref:N).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref": map[string]any{
						"type":        "string",
						"description": "The reference marker (e.g., '2', '#ref:2').",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "The text to insert into the input field.",
					},
				},
				"required": []string{"ref", "text"},
			},
		},
		{
			Name:        "browser_status",
			Description: "Get the current page URL, title, HTTP status, and live RAM memory consumption (RSS in MiB).",
			InputSchema: map[string]any{
				"type": "object",
			},
		},
		{
			Name:        "browser_search",
			Description: "Search the web in real-time across Google, DuckDuckGo and Wikipedia with live web crawling and AI synthesis.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query or question to investigate.",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Serve handles incoming JSON-RPC lines from r and writes responses to w.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(w, nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(w, req)
	}
	return scanner.Err()
}

func (s *Server) handleRequest(w io.Writer, req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.writeResponse(w, req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "ai-go-browser",
				"version": "1.0.0",
			},
		})

	case "notifications/initialized":
		// No-op for notification

	case "ping":
		s.writeResponse(w, req.ID, map[string]any{})

	case "tools/list":
		s.writeResponse(w, req.ID, map[string]any{
			"tools": s.Tools(),
		})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(w, req.ID, -32602, "Invalid params")
			return
		}

		resultText, err := s.callTool(params.Name, params.Arguments)
		if err != nil {
			s.writeResponse(w, req.ID, map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
				},
				"isError": true,
			})
			return
		}

		s.writeResponse(w, req.ID, map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": resultText},
			},
		})

	default:
		s.writeError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) callTool(name string, argsRaw json.RawMessage) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch name {
	case "browser_navigate":
		var args struct {
			URL      string `json:"url"`
			RenderJS bool   `json:"render_js"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "", err
		}
		res := s.browser.Execute(browser.Command{
			Cmd:      "open",
			URL:      args.URL,
			RenderJS: args.RenderJS,
		})
		if !res.OK {
			return "", fmt.Errorf("%s", res.Error)
		}
		title := ""
		if res.Page != nil {
			title = res.Page.Title
		}
		return fmt.Sprintf("Navigated to %s (Title: %q, Status: %d, RSS: %.2f MiB)", args.URL, title, res.Page.Status, res.Memory.RSSMiB), nil

	case "browser_snapshot":
		res := s.browser.Execute(browser.Command{
			Cmd: "snapshot",
		})
		if !res.OK {
			return "", fmt.Errorf("%s", res.Error)
		}
		if res.Page == nil {
			return "No active page", nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\nURL: %s (Status: %d)\n\n", res.Page.Title, res.Page.URL, res.Page.Status))
		sb.WriteString(res.Page.Markdown)
		if len(res.Page.Buttons) > 0 || len(res.Page.Inputs) > 0 || len(res.Page.Links) > 0 {
			sb.WriteString("\n\n### Interactive Elements:\n")
			for _, b := range res.Page.Buttons {
				sb.WriteString(fmt.Sprintf("- [%s] Button: %s\n", b.Ref, b.Label))
			}
			for _, f := range res.Page.Inputs {
				sb.WriteString(fmt.Sprintf("- [%s] Input (%s): %s\n", f.Ref, f.Type, f.Name))
			}
			for _, l := range res.Page.Links {
				sb.WriteString(fmt.Sprintf("- [%s] Link: %s -> %s\n", l.Ref, l.Label, l.URL))
			}
		}
		return sb.String(), nil

	case "browser_click":
		var args struct {
			Ref          string `json:"ref"`
			FallbackText string `json:"fallback_text"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "", err
		}

		// Try standard click
		res := s.browser.Execute(browser.Command{
			Cmd: "click",
			Ref: args.Ref,
		})
		if res.OK {
			return fmt.Sprintf("Clicked %s successfully", args.Ref), nil
		}

		// Self-healing fallback if element text was provided
		if args.FallbackText != "" && s.browser.Current != nil {
			var allEls []browser.Element
			for _, b := range s.browser.Current.Buttons {
				allEls = append(allEls, browser.Element{
					Ref:   b.Ref,
					Tag:   "button",
					Label: b.Label,
					Text:  b.Label,
				})
			}
			for _, l := range s.browser.Current.Links {
				allEls = append(allEls, browser.Element{
					Ref:   l.Ref,
					Tag:   "a",
					Label: l.Label,
					Text:  l.Label,
					Href:  l.URL,
				})
			}
			match := s.resolver.Resolve(allEls, resolver.MatchTarget{
				Ref:  args.Ref,
				Text: args.FallbackText,
			})
			if match != nil && match.Element.Ref != "" {
				retryRes := s.browser.Execute(browser.Command{
					Cmd: "click",
					Ref: match.Element.Ref,
				})
				if retryRes.OK {
					return fmt.Sprintf("Self-healed! Clicked %s (matched %q with %.0f%% confidence)", match.Element.Ref, match.Element.Text, match.Confidence*100), nil
				}
			}
		}

		return "", fmt.Errorf("click failed: %s", res.Error)

	case "browser_type":
		var args struct {
			Ref  string `json:"ref"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "", err
		}
		res := s.browser.Execute(browser.Command{
			Cmd:   "fill",
			Ref:   args.Ref,
			Value: args.Text,
		})
		if !res.OK {
			return "", fmt.Errorf("%s", res.Error)
		}
		return fmt.Sprintf("Typed %q into %s", args.Text, args.Ref), nil

	case "browser_status":
		p := s.browser.Current
		if p == nil {
			return "Browser is idle (no page loaded)", nil
		}
		mem := browser.ReadProcessMemory(os.Getpid())
		return fmt.Sprintf("Current URL: %s\nTitle: %q\nStatus: %d\nMemory RSS: %.2f MiB", p.URL, p.Title, p.Status, mem.RSSMiB), nil

	case "browser_search":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return "", err
		}
		if args.Query == "" {
			return "", fmt.Errorf("param 'query' is required")
		}
		eng := search.NewEngine()
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		resp, err := eng.Search(ctx, args.Query)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		sb.WriteString(resp.AISummary)
		sb.WriteString("\n\n### Fontes Encontradas:\n")
		for i, r := range resp.Results {
			sb.WriteString(fmt.Sprintf("%d. [%s](%s): %s\n", i+1, r.Title, r.URL, r.Snippet))
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) writeResponse(w io.Writer, id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s\n", bytes)
}

func (s *Server) writeError(w io.Writer, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s\n", bytes)
}
