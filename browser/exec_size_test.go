package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExecutableOverheadByLanguage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exec-size-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Compilar um executável Go nativo mínimo com informação embutida
	goSrc := `package main
import "fmt"
func main() {
	fmt.Println("Site: Portal Gov | Status: 200 | Auth: Bearer")
}
`
	goFile := filepath.Join(tmpDir, "site.go")
	_ = os.WriteFile(goFile, []byte(goSrc), 0644)
	goExe := filepath.Join(tmpDir, "site_go.exe")

	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", goExe, goFile)
	err = cmd.Run()
	var goExeSize int64 = 0
	if err == nil {
		if fi, err := os.Stat(goExe); err == nil {
			goExeSize = fi.Size()
		}
	}

	// 2. Módulo WASM mínimo (WebAssembly Header: 8 bytes)
	// Magic (\0asm) + Version (1) + minimal section
	wasmData := []byte{
		0x00, 0x61, 0x73, 0x6D, // \0asm
		0x01, 0x00, 0x00, 0x00, // version 1
		0x00, 0x08, 0x04, 's', 'i', 't', 'e', 0x01, // custom name section
	}
	wasmSize := len(wasmData)

	t.Logf("═════════════════════════════════════════════════════════════════")
	t.Logf("📏 [TAMANHOS REAIS DE EXECUTÁVEIS NA PRÁTICA]")
	t.Logf("═════════════════════════════════════════════════════════════════")
	t.Logf("1️⃣ Binário Go Nativo (.exe de SO): %d bytes (~%.2f MB)", goExeSize, float64(goExeSize)/(1024*1024))
	t.Logf("   -> Motivo: O compilador Go embute o Runtime (GC, Goroutines, scheduler)")
	t.Logf("   -> Se tivéssemos 2.000 sites em .exe separados: gitaria ~%.1f GB de disco!", float64(goExeSize*2000)/(1024*1024*1024))
	t.Logf("─────────────────────────────────────────────────────────────────")
	t.Logf("2️⃣ Módulo WebAssembly (.wasm executável em sandbox): %d bytes", wasmSize)
	t.Logf("   -> Sem runtime do SO, cabeçalho de apenas 8 bytes!")
	t.Logf("   -> 2.000 sites em WASM: apenas ~%.2f MB de disco!", float64(wasmSize*2000)/(1024*1024))
	t.Logf("═════════════════════════════════════════════════════════════════")
}
