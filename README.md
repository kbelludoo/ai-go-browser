# AI Go Browser — Navegador Híbrido de Alta Performance para Agentes de IA

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](#)
[![Memory](https://img.shields.io/badge/RAM-%3C%2075%20MB-green)](#)
[![TLS](https://img.shields.io/badge/Fingerprint-Chrome%20131%20JA4-success)](#)

O **AI Go Browser** é um navegador headless de arquitetura híbrida projetado especificamente para **Agentes Autônomos de Inteligência Artificial**, raspagem semântica e automação resiliente. 

Diferente de frameworks tradicionais (como Playwright, Puppeteer e Selenium) que consomem mais de **700 MB a 1 GB de RAM por aba** e são facilmente detectados por sistemas anti-bot, o AI Go Browser combina um motor ultraleve em **Go Puro**, execução segura em **WebAssembly (Wazero)** e controle nativo via **CDP (Chrome DevTools Protocol)** com disfarce criptográfico completo.

---

## ⚡ Por que é o Melhor Navegador para Agentes de IA?

### 1. Consumo de Memória 14x Menor (< 75 MB)
* **Playwright / Chromium:** 600 MB a 1.000 MB de RAM por instância.
* **AI Go Browser:** **47 MB a 73 MB** em regime de trabalho.
* Permite rodar **mais de 80 agentes simultâneos** em servidores modestos (como instâncias gratuitas da Oracle Cloud).

### 2. Redução Drástica de Tokens de LLM (86.6x mais barato)
Ao navegar na web, modelos de linguagem (GPT-4o, Claude 3.5, Gemini 1.5/2.0) cobram por token de contexto.
* Em vez de injetar 500 KB de HTML com CSS, SVGs e rastreadores, a observação semântica (`view:"read"`) extrai apenas referências acionáveis (`b1`, `b2`, `headings`, formulários e dados estruturados).
* Reduz payloads de 452 KB para **5.2 KB de JSON limpo** (**86.6x menos tokens** cobrados na API da LLM).

### 3. Disfarce Perfeito de Navegador Real (Auditoria 6/6 Aprovada)
Auditado e comprovado ao vivo em múltiplos servidores forenses públicos (`tls.peet.ws`, `tls.browserleaks.com`, `cloudflare.com` e `httpbin.org`):
* **TLS Fingerprint:** Assinatura criptográfica JA3/JA4 idêntica ao Google Chrome 131 oficial no Windows.
* **HTTP/2 Akamai Frame:** Negociação com as mesmas configurações de janela e pseudo-headers (`:method`, `:authority`, `:scheme`, `:path`) do Chromium.
* **Client Hints:** Cabeçalhos modernos de segurança (`Sec-CH-UA`, `Sec-CH-UA-Platform: "Windows"`).
* **DOM Stealth:** `navigator.webdriver = false`, objetos `window.chrome` nativos e plugins ativos.

---

## 🏛️ Arquitetura Híbrida em 3 Camadas

```
                           [ Agente de IA / LLM ]
                                     │
                      ┌──────────────┴──────────────┐
                      ▼                             ▼
           [ Camada 1: Go HTTP/2 ]       [ Camada 2: Wasm Engine ]
           • TLS Chrome 131 (JA4)        • QuickJS em WebAssembly
           • Parser HTML5 x/net          • Wazero JIT (Zero CGO)
           • Extração JSON / Next.js     • Scripts & Desafios Anti-Bot
                      │                             │
                      └──────────────┬──────────────┘
                                     │ (Fallback dinâmico)
                                     ▼
                        [ Camada 3: Ephemeral CDP ]
                        • Sessão Chromium / Brave / Edge
                        • TrustedHTML bypass (Input.insertText)
                        • Interação com Google Gemini & SPAs
```

---

## 📊 Benchmarks Oficiais

Testes físicos realizados em ambiente Windows AMD64 com Intel/AMD CPU:

| Métrica / Cenário | Chromium / Edge (CDP) | AI Go Browser | Vantagem Comprovada |
| :--- | :---: | :---: | :---: |
| **Memória Inicial (Cold Start)** | 606.3 MB | **73.6 MB** | 🏆 **8.2x MENOS MEMÓRIA** |
| **Memória Após Carga Pesada** | 711.4 MB | **47.7 MB** | 🏆 **14.9x MENOS MEMÓRIA** |
| **Latência por Ação (Round-Trip)** | 27.66 ms (0.28 ms/op) | **6.50 ms** (0.06 ms/op) | 🏆 **4.3x MAIS RÁPIDO** |
| **Proof-of-Work / Anti-Bot Hashing** | 36.50 ms | **10.80 ms** | 🏆 **3.4x MAIS RÁPIDO** |
| **Extração Hacker News** | 2.12 s (812 MB RAM) | **0.61 s (127 MB RAM)** | 🏆 **3.5x mais rápido** |
| **Extração Go.dev Oficial** | 1.88 s (907 MB RAM) | **0.68 s (129 MB RAM)** | 🏆 **2.7x mais rápido** |
| **Extração Wikipédia Brasil** | 1.99 s (1.092 MB RAM) | **2.31 s (145 MB RAM)** | 🏆 **7.5x menos RAM** |

---

## 🚀 Instalação e Compilação

### Pré-requisitos
* Go 1.26 ou superior instalado.
* Sem dependências de GCC ou MinGW (compilação 100% Go puro).

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

### 1. Navegação Semântica com o CLI
```powershell
# Navegar em modo compacto de baixo consumo
.\bin\aigobrowser.exe -compact=true -max-text=3000 https://go.dev

# Executar com renderização JavaScript híbrida
.\bin\aigobrowser.exe -auto-js=true https://news.ycombinator.com
```

### 2. Automação do Google Gemini (`cmd/geminicontroller`)
Controla sessões autenticadas do Gemini sem precisar de senhas soltas nem enfrentar bloqueios de TrustedHTML:

```powershell
# Enviar um prompt automático para o Gemini aberto no Brave/Chrome
go run ./cmd/geminicontroller -port=9222 -prompt="Explique a teoria da relatividade em 2 frases."
```

### 3. Suite de Benchmarks de Eficiência (`cmd/wasmbench`)
```powershell
go run ./cmd/wasmbench
```

---

## 📁 Estrutura do Repositório

* **`browser/`**: Núcleo do navegador (DOM, parsing HTML5, CDP, navegação, cookies, stealth layer e extração).
* **`cmd/aigobrowser/`**: CLI principal para automação e interação com agentes de IA.
* **`cmd/geminicontroller/`**: Controlador automatizado para interação com Google Gemini via CDP.
* **`cmd/wasmbench/`**: Ferramenta de auditoria de memória, latência e fingerprint.
* **`cmd/comparebrowser/`**, **`cmd/compareplaywright/`**: Ferramentas comparativas contra Chromium e Playwright.
* **`benchmark-results/`**: Relatórios científicos completos das rodadas de medição.

---

## 📜 Licença

Distribuído sob a licença MIT. Consulte `LICENSE` para mais informações.
