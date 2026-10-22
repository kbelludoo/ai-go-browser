package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPMemoryToolsIntegration(t *testing.T) {
	s := NewServer()
	defer func() {
		if s.memory != nil {
			_ = s.memory.Close()
		}
	}()

	// 1. Verify Tools list contains memory tools
	tools := s.Tools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	expectedTools := []string{
		"browser_navigate",
		"browser_snapshot",
		"browser_click",
		"browser_type",
		"browser_status",
		"browser_search",
		"browser_memory_search",
		"browser_memory_learn",
		"browser_memory_handoff_export",
		"browser_memory_handoff_resume",
	}

	for _, exp := range expectedTools {
		if !toolMap[exp] {
			t.Errorf("missing tool definition: %s", exp)
		}
	}

	// 2. Test browser_memory_learn
	learnReq := map[string]string{
		"domain": "testsite.com",
		"type":   "selector",
		"key":    "login_button",
		"value":  "#btn-login",
	}
	learnBytes, _ := json.Marshal(learnReq)
	learnRes, err := s.callTool("browser_memory_learn", learnBytes)
	if err != nil {
		t.Fatalf("browser_memory_learn failed: %v", err)
	}
	if !strings.Contains(learnRes, "Learned selector") {
		t.Errorf("unexpected learn response: %s", learnRes)
	}

	// 3. Test memory lookup after learn
	sel, ok := s.memory.GetLearnedSelector("testsite.com", "login_button")
	if !ok || sel != "#btn-login" {
		t.Errorf("expected learned selector #btn-login, got %q", sel)
	}

	// 4. Test browser_memory_search
	_ = s.memory.RecordVisit("https://testsite.com/docs", "Test Documentation", "High performance Go microservices architecture guide", 200)

	searchReq := map[string]interface{}{
		"query": "microservices architecture",
		"limit": 5,
	}
	searchBytes, _ := json.Marshal(searchReq)
	searchRes, err := s.callTool("browser_memory_search", searchBytes)
	if err != nil {
		t.Fatalf("browser_memory_search failed: %v", err)
	}
	if !strings.Contains(searchRes, "Test Documentation") {
		t.Errorf("expected search to find 'Test Documentation', got: %s", searchRes)
	}

	// 5. Test browser_memory_handoff_export & resume
	exportReq := map[string]string{
		"goal": "Verify Go microservices benchmark",
	}
	exportBytes, _ := json.Marshal(exportReq)
	exportRes, err := s.callTool("browser_memory_handoff_export", exportBytes)
	if err != nil {
		t.Fatalf("browser_memory_handoff_export failed: %v", err)
	}
	if !strings.Contains(exportRes, "Created Navigation Handoff Packet") {
		t.Errorf("unexpected export response: %s", exportRes)
	}
}
