# Auditoria independente das afirmações do README

Executada em **22/10/2026**, Linux amd64 (2 vCPU Xeon 2.60 GHz), **Go 1.27.3**, `CGO_ENABLED=0`,
sem Chromium e sem acesso a sites públicos. Cada linha abaixo tem o comando e a saída que a produziu;
os scripts estão em [`audit/`](audit/README.md) e rodam offline.

> Como o sandbox não alcança `dl.google.com`/`proxy.golang.org`, o toolchain veio do pacote PyPI
> `go-bin` 1.27.3 e as dependências de `go.mod` foram resolvidas por *tags/commits* do GitHub com
> `replace` locais **em uma cópia de trabalho**. Nenhum arquivo `.go` do repositório foi alterado.

## Resumo

| # | Afirmação do README | Veredicto |
| :-- | :--- | :--- |
| 1 | Compila em Go puro, sem GCC, Windows/Linux/macOS | **Parcial** — `cmd/aigobrowser` sim; `cmd/wasmbench` só compila no Windows; `cmd/jsworker` exige CGO + Duktape |
| 2 | `go test ./...` passa | **Parcial** — 0 falhas, mas 54 testes do pacote `browser` são pulados sem Chromium; `cmd/wasmbench` nem compila no Linux |
| 3 | Memória 47–73 MB, `< 75 MB`, 80 agentes simultâneos | **Comprovado** (e melhor): 12–24 MiB de pico por processo; 9,7 MiB/agente; 80 agentes ≈ 0,75 GiB |
| 4 | 14,9× menos RAM que Chromium (711,4 vs 47,7 MB) | **Não reproduzível / inconsistente** — os números não existem em nenhum artefato do repo; o próprio A/B do repo mede a árvore Chromium em 372–403 MiB |
| 5 | 86,6× menos tokens de LLM (452 KB → 5,2 KB) | **Refutado como generalização** — o próprio *audit* do repo (10 sites) dá **5,1×** agregado (1,0×–46,1×); a economia vem de **descartar as referências** e **truncar o texto** |
| 6 | TLS/JA3/JA4 idêntico ao Chrome 131 | **Refutado** — o ClientHello capturado é o do `crypto/tls` do Go: 13 cifras, **sem GREASE**; não há uTLS no `go.mod` |
| 7 | HTTP/2 “Akamai frame” igual ao Chromium | **Refutado** — SETTINGS `INITIAL_WINDOW_SIZE=4 MiB`, `MAX_FRAME_SIZE=1 MiB`, `MAX_HEADER_LIST_SIZE=10 MiB`; pseudo-headers em ordem `:authority,:method,:path,:scheme` |
| 8 | Client Hints (`Sec-CH-UA`, `Sec-CH-UA-Platform`) | **Refutado na camada Go** — a requisição envia só UA, Accept, Accept-Language, Accept-Encoding |
| 9 | Camada Wasm/QuickJS/Wazero | **Refutado** — zero imports de `wazero`, zero arquivos `.wasm`; o runtime goja existe mas nunca é usado fora de um teste |
| 10 | 4,3× mais rápido por ação (6,50 vs 27,66 ms) | **Não comprovado / inconsistente** — `wasmbench` imprime 65 µs vs 280 µs; medido aqui: 11,8–14,3 µs por snapshot in-process (sem comparação válida com CDP) |
| 11 | “Auditoria 6/6” em tls.peet.ws, browserleaks, Cloudflare | **Não comprovável** — nenhum artefato no repo; o “resultado” é uma string fixa em `cmd/wasmbench` |
| 12 | `cmd/wasmbench` é “suite de benchmarks de eficiência” | **Refutado** — mede só o working set do próprio processo e imprime os números como literais |
| 13 | `-demo` (autoteste) | **Bug** — falha com as flags padrão (`-block-private=true` bloqueia o próprio `127.0.0.1`) |
| 14 | Licença MIT (`Consulte LICENSE`) | **Falso** — não existe arquivo `LICENSE` no repositório |
| 15 | Tabela de benchmarks oficiais (606,3 / 711,4 / 73,6 / 47,7 / 27,66 / 6,50 / 36,50 / 10,80 / 812 MB / 2,12 s) | **Sem rastro** — nenhum desses valores aparece em `benchmark-results/`, `PERFORMANCE.md` ou nos JSONs |
| 16 | Microbenchmarks de `PERFORMANCE.md` | **Comprovado** — reproduzidos com ±10–30% via `go test -bench` |

## 1. Compilação e instalação

```console
$ CGO_ENABLED=0 go build -o bin/aigobrowser ./cmd/aigobrowser      # OK
$ go vet ./browser ./cmd/aigobrowser ./audit/...                   # OK
$ go run ./cmd/wasmbench                                           # comando do README, seção 3
# ai-go-browser/cmd/wasmbench
cmd/wasmbench/main.go:15:24: undefined: syscall.NewLazyDLL
cmd/wasmbench/main.go:32:25: undefined: syscall.OpenProcess
cmd/wasmbench/main.go:36:16: undefined: syscall.CloseHandle
```

O binário principal realmente não precisa de GCC/MinGW (`CGO_ENABLED=0`), mas o badge
“Windows | Linux | macOS” não vale para o repositório inteiro, e o `cmd/jsworker` declara
`//go:build linux && cgo` com `#cgo LDFLAGS: -l:libduktape.so.207`.

## 2. Testes

```console
$ go test ./browser -count=1 -v
# 289 RUN (inclui subtests) | 112 PASS | 54 SKIP | 0 FAIL   (~1,0–1,5 s)
# motivo dos SKIP: "skipping test: no browser installed"
```

`VALIDATION.md` afirma “`go test ./... -count=1`: passou, incluindo browser (85,401 s)”. Aqui o pacote
`browser` passa em ~1 s porque **um terço dos testes (54) não roda sem Chromium** — justamente os de
fill nativo, frames, shadow DOM e BFCache que o documento usa como evidência. Também falham na
compilação `cmd/wasmbench` (Linux) e os pacotes que importam `modernc.org/sqlite`
(`pkg/memory`, `pkg/mcp`, `cmd/mcp-server`, `cmd/prove_real_sites`, `cmd/real_comparison_audit`) —
esse último caso é do ambiente (proxy bloqueado), não do código.

## 3. Memória — comprovado

| Cenário | Pico de RSS | RSS auto-relatado | Saída JSON |
| :--- | ---: | ---: | ---: |
| Página trivial | 12,4 MiB | 11,0 MiB | 0,7 KB |
| Página “realista” de 368 KB, `-compact=true` | 16,0 MiB | 14,5 MiB | 27,6 KB |
| Mesma página, `-compact=false` | 18,6 MiB | 18,1 MiB | 327,6 KB |
| Página de 2.000 links, `-compact=true` | 22,1 MiB | 20,0 MiB | 249,9 KB |
| Mesma página, `-compact=false` | 24,2 MiB | 20,4 MiB | 605,4 KB |

Aritmética dos “mais de 80 agentes”: `audit/agents.py` com 20 processos concorrentes na página de
368 KB deu **67,6 MiB no total, 9,7 MiB/agente** → 80 agentes ≈ **0,75 GiB**. A afirmação de baixo
consumo se sustenta para o motor Go (processos curtos; sessões longas ficam um pouco acima).

## 4. Tokens — o número de 86,6× não é o que o README diz

`go run ./audit/payload`:

| Fixture | HTML cru | Snapshot completo | Padrão da CLI (`-compact`) | `view:"read"` | HTML/read | refs no `read` |
| :--- | ---: | ---: | ---: | ---: | ---: | :--- |
| Página realista, 400 artigos, 200 links | 367.992 B | 322.988 B | 27.765 B (13,3× menor) | 5.423 B | **67,9×** | **0 links, 0 headings, 0 forms** |
| Página sintética de 2.000 links | 194.776 B | 604.395 B (59,6× **maior** que o `read`) | 250.114 B (**1,3× maior que o HTML**) | 5.418 B | **35,9×** | **0 links** |

Três fatos verificados:

1. **A economia não vem de “extrair referências acionáveis”** — ela vem de *remover* todas as
   referências. `browser/browser.go:622` (`readProjection`) zera `Links`, `Forms`, `Headings`,
   `Inputs`, `Buttons` e `Markdown`; o `PERFORMANCE.md` admite isso (“A leitura não produz referências”),
   o README afirma o contrário (“extrai apenas referências acionáveis `b1`, `b2`, headings, formulários”).
2. **O tamanho do `read` é o `-max-text`.** Com o padrão da CLI (10.000 caracteres) o `read` sobe para
   ~10,4 KB; com o padrão da biblioteca (5.000) fica em 5,4 KB. Aumentar o limite não muda nada depois
   da navegação porque o texto já foi truncado no parse. Ou seja: 5,2 KB ≈ “5.000 caracteres de texto”.
3. **O próprio *audit* do repositório contradiz o README.** `benchmark-results/ai-superiority-audit.json`
   (10 sites reais, HTML cru vs. saída do agente) traz `aggregate_token_x_cheaper = 5.1`, com
   Hacker News em **2,4×**, Wikipedia em **3,5×**, Httpbin em **1,0×** e GitHub em 46,1×. O “86,6×”
   é o melhor caso de uma página sintética link-densa, não o ganho de “navegar na web”.

## 5. TLS / JA3 / JA4 — refutado

`python3 audit/tls_hello.py 8443` + `aigobrowser … https://localhost:8443/`:

```json
{
  "ja3": "771,49195-49199-49196-49200-52393-52392-49161-49171-49162-49172-4865-4866-4867,0-11-65281-23-18-5-10-13-50-16-43-51,4588-4587-4589-29-23-24-25,0",
  "ja3_md5": "03117a8ed39ef02427ebbc39f121275c",
  "ciphers": ["0xc02b","0xc02f","0xc02c","0xc030","0xcca9","0xcca8","0xc009","0xc013","0xc00a","0xc014","0x1301","0x1302","0x1303"],
  "extensions": ["0x0","0xb","0xff01","0x17","0x12","0x5","0xa","0xd","0x32","0x10","0x2b","0x33"],
  "alpn": ["h2","http/1.1"],
  "has_grease": false
}
```

Sem SNI (conexão por IP) o MD5 é `60828da5b42313d165aae2938e651b72`; a diferença entre os dois é só
a extensão `0x0`. Esse é o *fingerprint* do `crypto/tls` do Go, e ele não pode ser o do Chrome:

* **nenhum valor GREASE** — o Chrome envia GREASE em cifras e extensões em toda conexão;
* **13 cifras**, com as de TLS 1.3 (`4865-4866-4867`) **no fim**; o Chrome as coloca no início;
* faltam extensões que o Chrome manda em toda conexão: `0x4469` (ALPS), `0x001b`
  (compress_certificate), `0x002d` (PSK) e `0xfe0d` (ECH), e a ordem das que sobraram também não
  é a do Chrome;
* `go.mod` **não tem uTLS** e `browser/transport.go` só monta `&tls.Config{MinVersion: TLS12}` —
  o `crypto/tls` do Go não permite reescrever o ClientHello.

Detalhe relacionado: o *User-Agent* padrão do motor Go é **Chrome/124**
(`browser/browser.go:254`), não 131. O texto “Chrome/131.0.0.0” só existe em `browser/launch_args.go`
como argumento de linha de comando do Chromium real (camada 3) — nesse caso o fingerprint é do
Chromium porque é o Chromium, o que não é mérito do projeto.

## 6. HTTP/2 — refutado

`audit/h2_probe.py` + `audit/h2decode`:

```text
ALPN=h2  preface=true
SETTINGS = {ENABLE_PUSH: 0, INITIAL_WINDOW_SIZE: 4194304, MAX_FRAME_SIZE: 1048576, MAX_HEADER_LIST_SIZE: 10485760}
frames   = SETTINGS, WINDOW_UPDATE(4 B), HEADERS
:authority: localhost:8447
:method: GET
:path: /
:scheme: https
user-agent: …Chrome/124.0.0.0…
accept-encoding: gzip
```

São os padrões do `x/net/http2` do Go. O Chromium negocia `HEADER_TABLE_SIZE=65536`,
`MAX_CONCURRENT_STREAMS=1000`, `INITIAL_WINDOW_SIZE=6291456`, `MAX_HEADER_LIST_SIZE=262144`,
envia `WINDOW_UPDATE` de 15.663.105 e frames `PRIORITY` encadeados, e ordena os pseudo-headers
`:method,:authority,:scheme,:path`. Aqui a ordem é `:authority,:method,:path,:scheme`, não há
`PRIORITY`, e o README afirma exatamente o contrário.

## 7. Client Hints e cabeçalhos — refutado

```json
{"requestline": "GET / HTTP/1.1",
 "headers": {"Host": "127.0.0.1:8097",
  "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
  "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
  "Accept-Language": "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7",
  "Accept-Encoding": "gzip"}}
```

Não há `Sec-CH-UA`, `Sec-CH-UA-Platform`, `Sec-Fetch-*` nem `Accept-Encoding: gzip, deflate, br, zstd`.
É um cliente HTTP trivialmente identificável — o oposto de “disfarce perfeito”. O `browser/stealth.go`
patcheia JS de páginas, mas só roda dentro de uma sessão CDP (Chromium), nunca na camada Go.

## 8. Camada Wasm / QuickJS — não existe no código

```console
$ grep -rl 'wazero' --include=*.go .        # 0 arquivos (go.mod exige, nenhum import)
$ find . -name '*.wasm'                     # 0 arquivos
$ grep -rn 'NewNativeJSRuntime' --include=*.go . | grep -v _test
./browser/script_engine.go:35:func NewNativeJSRuntime(...)   # só a definição
$ grep -rn 'NewNativeJSRuntime' --include=*_test.go .
./browser/milestone1_test.go:93: rt, err := NewNativeJSRuntime(b, b.Document, ts.URL)
```

O diagrama de arquitetura (QuickJS em WebAssembly, Wazero JIT, “scripts & desafios anti-bot”)
não tem implementação. A execução de JS real só acontece na camada 3 (Chromium via CDP), e o
runtime goja (`script_engine.go`) é usado apenas por um teste. Teste prático: uma página cujo texto é
escrito por `<script>` inline continua “placeholder empty” com `-engine=http` e `-auto-js=true`;
uma *shell* CSR volta com texto vazio e o aviso “conteúdo interativo requer execução de JS”.

## 9. Latência e “auditoria 6/6”

* In-process, `snapshot` completo vs `view:"read"`: **11,8 µs vs 11,9 µs** (página realista) e
  **14,3 vs 13,9 µs** (página de 2.000 links) — no motor HTTP não há ganho mensurável de latência;
  o 31,5× do `PERFORMANCE.md` é medido em sessão Chromium, que aqui não existe.
* A “4,3× mais rápido por ação” do README aparece como `6,50 ms` vs `27,66 ms`, mas o
  `cmd/wasmbench` imprime `65 µs vs 280 µs`, e ainda `(0,28 ms/op)` vs `(0,06 ms/op)` no README.
  Três conjuntos de números incompatíveis, nenhum reproduzível.
* `cmd/wasmbench/main.go` não mede nada além do working set do próprio processo:

```go
goMem := getWorkingSetMB(os.Getpid())
fmt.Println("   1. Consumo de Memória: 8.2x a 14.9x menor que o Chromium (73 MB vs 711 MB)")
fmt.Println("   2. Latência por Ação: 4.3x mais rápido (65 µs vs 280 µs)")
fmt.Println("   3. Redução de Tokens de IA: 86.6x menor (5.2 KB vs 452 KB)")
fmt.Println("   4. Fingerprint de Rede: TLS JA4 + Akamai HTTP/2 100% aprovado")
```

É daqui que saem os “6/6 aprovado”: de um `fmt.Println`. Nenhum relatório da auditoria em
`tls.peet.ws`/`browserleaks` foi salvo no repositório (`grep -i ja3\|ja4` só acha o README e essa linha).

## 10. Autoteste `-demo` quebrado com as flags padrão

```console
$ ./bin/aigobrowser -demo
{"ok":false,"error":"blocked destination: private/internal network address not allowed (SSRF
 protection): 127.0.0.1", ...}          # exit 1

$ ./bin/aigobrowser -demo -block-private=false
{"ok":true,"page":{"url":"http://127.0.0.1:38919","title":"AI Browser Demo", ...}}
```

Com `-block-private=false` o autoteste funciona e mostra o que o projeto **realmente** faz bem:
extração semântica com refs (`l1`, `b1`, `b2`, `b3`), formulário com campos (`i1`), `action_summary`
e `stack`/`content_yield`. Esse é o núcleo comprovado do produto.

## 11. Rastreabilidade dos números do README

`grep` por `606.3`, `711.4`, `73.6`, `47.7`, `27.66`, `6.50`, `36.50`, `10.80`, `812`, `2.12 s` em
todo o repositório (`.md`, `.json`, `.txt`) retorna **apenas as linhas do próprio README**. E o
próprio repositório mede coisas diferentes:

* `benchmark-results/fase1.md`: maior amostra RSS da árvore Chromium **372,23 MiB** (AI Go) e
  **403,22 MiB** (CDP) — não 606/711 MB. Com esses valores o ganho é ~8–9×, não 14,9×.
* `benchmark-results/real-sites-100-queuefix.md`: 100 URLs × 5 rodadas — **452/500 assertivas para o
  AI Go vs 444/500** para o Chromium, mas **tempo total pior** (341.172,67 ms vs 305.230,42 ms) e
  árvore Chromium de **2.068,47 MiB** quando o AI Go usa JS.
* A mesma linha do README admite 127/129/145 MB de RAM nas extrações de HN/go.dev/Wikipédia,
  contradizendo o badge “RAM < 75 MB” e a seção “47 MB a 73 MB”.

## O que não foi possível testar aqui

Sem Chromium instalado (e sem rede para `apt`/CDNs de browser) não foi possível reproduzir:
os 54 testes pulados (fill nativo, frames, shadow DOM, BFCache, fila CDP), as comparações de tempo
e memória contra Playwright/Chromium, os números de extração em sites públicos
(`api-go-superiority-audit`, `real-sites-100`) e a auditoria anti-bot “6/6”. O sandbox também não
alcança `tls.peet.ws` nem sites públicos. Essas afirmações continuam **não verificadas** — não
“comprovadas” — e as que dependem do fingerprint já foram refutadas por captura direta.

## Conclusão

O que se sustenta: o **motor Go de extração semântica** (parser HTML5 `x/net/html`, refs estáveis,
formulários, ações, detecção de stack/CSR, SSRF, histórico, limites de memória), com consumo real de
**12–24 MiB** e microbenchmarks de `PERFORMANCE.md` reproduzíveis (compacto = 2,36 MB/op e 6,09 ms
vs 7,40 MB/op e 12,29 ms no modo completo; 1.000 links = 3,82 MB/op; query = 4,1 KB/op).

O que não se sustenta: o **disfarce de Chrome 131** (TLS, HTTP/2 e Client Hints são os do Go),
a **camada Wasm/QuickJS** (inexistente), os **números de 86,6× de tokens e 14,9× de memória** como
afirmações gerais (contraditas pelos próprios relatórios do repositório), a **“auditoria 6/6”**
(uma string fixa) e a **tabela de benchmarks oficiais** (sem artefato de origem).

Correções mínimas sugeridas ao README: remover o badge e a seção de JA4/Client Hints (ou implementar
uTLS de verdade), remover o diagrama Wasm/QuickJS, trocar “86,6×” pelo número medido do próprio
`ai-superiority-audit.json` (5,1×, citando que `view:"read"` não devolve refs), citar as fontes de
cada linha da tabela de benchmarks, corrigir o exemplo `-demo` para usar `-block-private=false`,
tirar o `cmd/wasmbench` da seção de benchmarks (ou medir algo), e adicionar o `LICENSE`.
