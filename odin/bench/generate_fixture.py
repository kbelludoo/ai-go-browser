#!/usr/bin/env python3
"""Deterministic medium HTML fixture to compare static parsers, with no network."""
import html
from pathlib import Path

def main():
    chunks=['<!doctype html><html lang="pt-BR"><head><title>Benchmark Odin Go</title><meta name="description" content="Catálogo de teste estático"></head><body><main><h1>Catálogo</h1>']
    for i in range(350):
        label=f"Produto {i:04d}"
        desc=f"Descrição detalhada do produto {i}: desempenho, documentação, estabilidade e verificação independente."
        chunks.append(f'<section class="row"><h2>{label}</h2><p>{html.escape(desc)}</p><a href="/products/{i}">Abrir {label}</a></section>')
    chunks.append('<form action="/search" method="get"><input name="q"><button>Buscar</button></form></main></body></html>')
    output=Path("odin/bench/fixture_generated.html")
    output.write_text("\n".join(chunks),encoding="utf-8")
    print(f"generated {output}: {output.stat().st_size} bytes")

if __name__=="__main__":
    main()
