# AI Go Browser — Navegador Nativo de Alta Performance para Agentes de IA

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Audit CI](https://github.com/kbelludoo/ai-go-browser/actions/workflows/empirical-audit.yml/badge.svg)](https://github.com/kbelludoo/ai-go-browser/actions/workflows/empirical-audit.yml)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](#)
[![Memory](https://img.shields.io/badge/RAM-12--24%20MiB-success)](#)
[![TLS](https://img.shields.io/badge/uTLS-Chrome%20GREASE%20JA4-success)](#)

O **AI Go Browser** é um navegador semântico e headless projetado especificamente para **Agentes Autônomos de Inteligência Artificial**, raspagem semântica e automação resiliente.

Diferente de frameworks tradicionais (como Playwright, Puppeteer e Selenium) que consomem entre **700 MB e 2.2 GB de RAM por sessão** e explodem janelas de contexto de LLMs com dezenas de milhares de tokens HTML brutos, o AI Go Browser utiliza um motor nativo em **Go puro**, transporte com **uTLS (camuflagem TLS com GREASE real)**, parser semântico com referências acionáveis (`#ref`), dedução de formulários/botões pré-clique e memória local integrada com SQLite FTS5.

---

## ⚡ Diferenciais de Engenharia para Agentes de IA

### 1. Consumo Físico de Memória Ultrabaixo (12 a 24 MiB)
* **Playwright / Chromium Headless:** 700 MB a 2.242 MB de RAM (árvore de processos Chromium).
* **AI Go Browser (Motor Nativo):** **12 MiB a 24 MiB** de RSS por processo.
* **Escala:** Medido independentemente em ~9,7 MiB por agente isolado, permitindo rodar **mais de 80 agentes simultâneos** em servidores modestos (como VPS de 1 GB de RAM) sem acionar o Linux OOM Killer.

### 2. Redução Semântica de Tokens para LLMs (~89% de Economia)
Modelos de linguagem (GPT-4o, Claude 3.5, Gemini 1.5/2.0) cobram por token e sofrem com degradação de atenção ao processar HTML massivo:
* Em vez de injetar páginas inteiras de HTML cru com estilos, scripts e SVGs, o parser semântico gera Markdown compacto estruturado com referências acionáveis indexadas (`l1`, `b1`, `f1`, `i1`).
* **Comprovado em 200 sites reais da Internet:** Reduziu **20.783.600 tokens brutos para 2.216.100 tokens** — uma economia média de **89.34%** de contexto.

### 3. Inteligência Pré-Clique nos Botões (Dedução de Ações)
* Navegadores tradicionais tratam botões como elementos visuais opacos.
* O AI Go Browser inspeciona a árvore de formulários associados e atributos de dados, revelando para a IA o **endpoint de destino (URL), método HTTP (`GET`/`POST`) e parâmetros/CSRF ocultos** antes de qualquer clique ser executado.

### 4. Zero-JS Hydration (SPA & SSR)
* Páginas modernas construídas com Next.js, Nuxt ou com metadados estruturados expõem seu estado em tags `<script id="__NEXT_DATA__">` e Schema.org (`JSON-LD`).
* O motor extrai o estado estruturado da aplicação em microssegundos sem precisar inicializar motores pesados de renderização.

### 5. Camuflagem de Rede com uTLS e Client Hints
* Integração de transporte com **`uTLS`** (`github.com/refraction-networking/utls`), gerando um ClientHello TLS idêntico ao Chrome com suporte a **cifras GREASE** e extensões TLS 1.3 na ordem oficial.
* Envio automático de cabeçalhos de integridade modernos: `Sec-CH-UA`, `Sec-CH-UA-Platform`, `Sec-Fetch-Dest`, `Sec-Fetch-Mode`.

---

## 🏛️ Arquitetura do Sistema

```
                       [ Agente de IA / LLM ]
                                  │
                 ┌────────────────┴────────────────┐
                 ▼                                 ▼
      [ Motor Nativo Go Puro ]          [ Camada de Transporte ]
      • Parser HTML5 x/net              • uTLS com GREASE & JA4
      • Extração de Refs (#ref)         • Proteção Anti-SSRF
      • Zero-JS Hydration               • Client Hints Chrome 131
      • Dedução Pré-Clique de Ações     • Governor de Memória (sync.Pool)
                 │                                 │
                 └────────────────┬────────────────┘
                                  │
                 ┌────────────────┴────────────────┐
                 ▼                                 ▼
      [ JavaScript In-Process ]         [ Memória Persistente ]
      • Goja ECMAScript VM in-process   • SQLite FTS5 + BM25
      • Suporte a cookies/sessões       • O(1) recall de páginas
      • Zero dependência de binários    • Busca semântica local
```

---

## 📊 Resultados Empíricos Medidos

### Benchmark Comparativo Direto (Executado no mesmo ambiente):
Dados físicos medidos em tempo real comparando `ai_go` contra `chromium_cdp` ([`benchmark-results/live_direct_compare.md`](benchmark-results/live_direct_compare.md)):

| Métrica | Chromium Headless (msedge.exe via CDP) | AI Go Browser (Motor Nativo) | Impacto Real |
| :--- | :---: | :---: | :---: |
| **Startup do Motor** | 426.51 ms | **0.00 ms** | Inicialização in-process instantânea |
| **Tempo Total de Execução** | 5.414 ms | **2.393 ms** | **Mais de 2x mais rápido** |
| **Pico de Memória RAM (RSS)** | **738.69 MiB** | **24.42 MiB** | **30x menos memória física** |

### Benchmark em Larga Escala (200 Sites Reais):
Varredura concorrente em 200 sites públicos globais ([`benchmark-results/real-sites-200.md`](benchmark-results/real-sites-200.md)):
- **Taxa de Sucesso:** 86.0% (172/200)
- **Latência Média de Parse Go:** 37.96 ms
- **Tokens HTML Brutos:** 20.783.600 tokens
- **Tokens Agente (#ref Markdown):** 2.216.100 tokens (Economia de 18.567.500 tokens / **89.34%**)
- **Botões com Ação Pré-Deduzida:** 216 botões
- **Entidades Schema.org / JSON-LD:** 107 entidades extraídas sem executar JS

---

## 🚀 Instalação e Compilação

### Pré-requisitos
* Go 1.24 ou superior instalado.
* Compilação 100% Go puro (sem GCC/CGO obrigatório no binário principal).

```bash
# Clonar o repositório
git clone https://github.com/kbelludoo/ai-go-browser.git
cd ai-go-browser

# Baixar dependências
go mod tidy

# Compilar o navegador principal
go build -o bin/aigobrowser.exe ./cmd/aigobrowser
```

---

## 🛠️ Modos de Uso

### 1. Autoteste Integrado (`-demo`)
Executa um fluxo completo local (navegação, clique em links, preenchimento de inputs e submissão de formulário) com medição de memória e tempo:
```powershell
.\bin\aigobrowser.exe -demo
```

### 2. Navegação Semântica via CLI
```powershell
# Navegar em modo compacto de baixo consumo
.\bin\aigobrowser.exe -compact=true -max-text=3000 https://go.dev

# Saída direta em formato Markdown com referências para a IA
.\bin\aigobrowser.exe -format=markdown https://github.com
```

### 3. Validador de Fingerprint e Eficiência (`cmd/wasmbench`)
Mede o parsing in-process e audita ao vivo o handshake TLS contra servidores forenses:
```powershell
go run ./cmd/wasmbench
```

### 4. Executar o Benchmark de 200 Sites
```powershell
go run ./cmd/benchmark_200_sites
```

### 5. Reprodução Completa da Auditoria em 1 Comando
Para que qualquer pesquisador, auditor ou desenvolvedor possa reproduzir localmente todas as medições empíricas sem configurações adicionais:

```bash
# No Linux ou macOS:
bash reproduce.sh

# No Windows (PowerShell):
.\reproduce.ps1
```

O script compila os binários em Go puro, executa o autoteste `-demo`, audita a assinatura TLS contra o `tls.peet.ws` e executa as tarefas do WebArena, salvando os resultados em `benchmark-results/`.

---

## 📄 Licença

Distribuído sob a licença [MIT](LICENSE). Consulte o arquivo `LICENSE` para obter mais detalhes.
