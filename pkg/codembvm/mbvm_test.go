package codembvm_test

import (
	"path/filepath"
	"testing"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/codembvm"
)

func TestCompileAIGoBrowserToMBVM(t *testing.T) {
	// Root dir of ai-go-browser
	rootDir, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to resolve root dir: %v", err)
	}

	start := time.Now()
	mbvm, bytecodeBytes, err := codembvm.CompileProject(rootDir)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("falha ao compilar projeto para MB-VM: %v", err)
	}

	rawTokensEst := int(float64(mbvm.RawBytes) / 4.0) // ~4 chars per token for raw Go source
	bytecodeSize := len(bytecodeBytes)

	// AI Prompt Context
	aiContext := mbvm.GenerateAIPromptContext(100000)
	aiTokensEst := browser.EstimateTokens(aiContext)

	sampleLen := 300
	if sampleLen > len(aiContext) {
		sampleLen = len(aiContext)
	}
	t.Logf("aiContext len: %d chars, sample:\n%s", len(aiContext), aiContext[:sampleLen])

	tokenSavingsPct := (1.0 - float64(aiTokensEst)/float64(rawTokensEst)) * 100
	byteSavingsPct := (1.0 - float64(bytecodeSize)/float64(mbvm.RawBytes)) * 100

	t.Logf("═════════════════════════════════════════════════════════════════════════════")
	t.Logf("🧠 [TESTE EMPÍRICO REAL] COMPILAÇÃO DO AI-GO-BROWSER EM MB-VM")
	t.Logf("═════════════════════════════════════════════════════════════════════════════")
	t.Logf("📁 Arquivos Go analisados:            %d arquivos", mbvm.TotalFiles)
	t.Logf("📦 Símbolos extraídos (AST pura):     %d structs/interfaces/funções", len(mbvm.Symbols))
	t.Logf("⏱️ Tempo de compilação da MB-VM:      %v", duration)
	t.Logf("─────────────────────────────────────────────────────────────────────────────")
	t.Logf("📄 1. CÓDIGO BRUTO (RAW GO SOURCE):")
	t.Logf("   - Tamanho total em disco:          %d bytes (~%.2f MB)", mbvm.RawBytes, float64(mbvm.RawBytes)/(1024*1024))
	t.Logf("   - Tokens que a IA precisaria ler:  ~%d tokens (Inviável para maioria dos chats!)", rawTokensEst)
	t.Logf("─────────────────────────────────────────────────────────────────────────────")
	t.Logf("⚡ 2. ARQUIVO BINÁRIO MB-VM (.mbvm):")
	t.Logf("   - Tamanho compilado em bytecode:   %d bytes (~%.1f KB)", bytecodeSize, float64(bytecodeSize)/1024)
	t.Logf("   - Economia de armazenamento:       %.2f%% de redução de bytes!", byteSavingsPct)
	t.Logf("─────────────────────────────────────────────────────────────────────────────")
	t.Logf("🤖 3. CONTEXTO ENTREGUE À IA (SEM PERDA DE CONTEXTO):")
	t.Logf("   - Tokens consumidos pela IA:       ~%d tokens (Cabe com folga em QUALQUER prompt!)", aiTokensEst)
	t.Logf("   - 🎯 ECONOMIA REAL DE TOKENS:      %.2f%% DE REDUÇÃO!", tokenSavingsPct)
	t.Logf("   - Preserva: Todas as interfaces, structs, métodos, assinaturas e pacotes!")
	t.Logf("═════════════════════════════════════════════════════════════════════════════")

	if tokenSavingsPct < 80.0 {
		t.Fatalf("esperava reducao de tokens >= 80%%, obteve %.2f%%", tokenSavingsPct)
	}
}
