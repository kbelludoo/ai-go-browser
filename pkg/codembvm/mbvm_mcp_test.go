package codembvm_test

import (
	"os"
	"path/filepath"
	"testing"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/codembvm"
)

func TestCompileMCPSchemasToMBVM(t *testing.T) {
	mcpRoot := `C:\Users\kbell\.gemini\antigravity\mcp`
	if _, err := os.Stat(mcpRoot); os.IsNotExist(err) {
		t.Skip("mcpRoot not found on this environment")
	}

	entries, err := os.ReadDir(mcpRoot)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	var totalRawBytes, totalBytecodeBytes, totalRawTokens, totalMbvmTokens int

	t.Logf("═════════════════════════════════════════════════════════════════════════════")
	t.Logf("⚡ [BENCHMARK EMPÍRICO REAL] COMPILAÇÃO DE TODOS OS MCPS EM MB-VM")
	t.Logf("═════════════════════════════════════════════════════════════════════════════")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mcpDir := filepath.Join(mcpRoot, e.Name())
		bin, proj, rawBytes, err := codembvm.CompileMCPSchemas(mcpDir)
		if err != nil || rawBytes == 0 {
			continue
		}

		rawTokensEst := int(float64(rawBytes) / 3.5)
		mbvmTokensEst := browser.EstimateTokens(proj)
		bytecodeBytes := len(bin)

		totalRawBytes += rawBytes
		totalBytecodeBytes += bytecodeBytes
		totalRawTokens += rawTokensEst
		totalMbvmTokens += mbvmTokensEst

		tokenSavingsPct := (1.0 - float64(mbvmTokensEst)/float64(rawTokensEst)) * 100
		byteSavingsPct := (1.0 - float64(bytecodeBytes)/float64(rawBytes)) * 100

		t.Logf("📦 MCP Server: %-15s | JSON: %6d B (~%5d tok) | MB-VM: %5d B (~%4d tok) | -%.1f%% tok | -%.1f%% bytes",
			e.Name(), rawBytes, rawTokensEst, bytecodeBytes, mbvmTokensEst, tokenSavingsPct, byteSavingsPct)
	}

	totalTokenSavings := (1.0 - float64(totalMbvmTokens)/float64(totalRawTokens)) * 100
	totalByteSavings := (1.0 - float64(totalBytecodeBytes)/float64(totalRawBytes)) * 100

	t.Logf("─────────────────────────────────────────────────────────────────────────────")
	t.Logf("TOTAL CONSOLIDADO EM TODOS OS MCPS:")
	t.Logf("  - JSON Bruto:        %d bytes (~%d tokens)", totalRawBytes, totalRawTokens)
	t.Logf("  - MB-VM Compilado:   %d bytes (~%d tokens)", totalBytecodeBytes, totalMbvmTokens)
	t.Logf("  - Economia de Armazenamento: %.2f%% de redução de bytes", totalByteSavings)
	t.Logf("  - Economia de Contexto IA:   %.2f%% DE REDUÇÃO REAL DE TOKENS!", totalTokenSavings)
	t.Logf("═════════════════════════════════════════════════════════════════════════════")

	if totalTokenSavings < 70.0 {
		t.Fatalf("esperava reducao total de tokens >= 70%%, obteve %.2f%%", totalTokenSavings)
	}
}
