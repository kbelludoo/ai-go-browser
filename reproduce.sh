#!/usr/bin/env bash
set -e

echo "================================================================================"
echo "⚡ AI GO BROWSER: REPRODUÇÃO DA AUDITORIA EMPÍRICA INDEPENDENTE"
echo "   Ambiente: $(uname -s) $(uname -m) | Go: $(go version)"
echo "================================================================================"

mkdir -p bin

echo "📦 [1/4] Compilando binários em Go puro (sem CGO)..."
CGO_ENABLED=0 go build -o bin/aigobrowser ./cmd/aigobrowser
go build -o bin/wasmbench ./cmd/wasmbench
go build -o bin/external_benchmarks ./cmd/external_benchmarks

echo ""
echo "🧪 [2/4] Executando Autoteste Integrado (-demo)..."
./bin/aigobrowser -demo

echo ""
echo "🔒 [3/4] Executando Auditoria Forense TLS (tls.peet.ws) & Microbenchmark..."
./bin/wasmbench

echo ""
echo "🏛️ [4/4] Executando Tarefas do WebArena & Mind2Web..."
./bin/external_benchmarks

echo ""
echo "================================================================================"
echo "✅ AUDITORIA CONCLUÍDA COM SUCESSO! Relatórios salvos em benchmark-results/"
echo "================================================================================"
