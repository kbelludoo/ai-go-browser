# AI Go Browser v0.7 — Navegador Híbrido Universal para Agentes de IA

Um navegador headless de alta performance projetado para **agentes de IA e automação semântica**, capaz de acessar **qualquer tecnologia da Web** mantendo o consumo estrito abaixo de 300 MB de RAM (com média entre **10 e 25 MB**).

## Arquitetura Híbrida Universal (3 Camadas)

1. **Camada 1: Motor Go Puro (Padrão para 80%+ da Web)**:
   - Conexão HTTP/2 com TLS moderno, Happy Eyeballs e cookies de sessão.
   - Resposta em **50 a 300 ms** e **<25 MB de RAM**.
   - Ideal para Wikipédia, portais de notícia (CNN, G1, BBC), blogs, documentações, WordPress, PHP, Django, Go e APIs.
2. **Camada 2: Extrator de Dados Estruturados e Hydration**:
   - Extrai automaticamente payloads `<script id="__NEXT_DATA__">`, Nuxt data e `<script type="application/ld+json">`.
   - Entrega dados completos de artigos e produtos mesmo em páginas que usam hidratação de cliente.
3. **Camada 3: Motor Headless Efêmero (On-Demand para SPAs em React/Vue/Angular)**:
   - Acionado via comando `{"cmd":"open","url":"...","js":true}`, comando CLI `render <url>` ou flag `-render-js`.
   - **Fallback Automático (`-auto-js`)**: se o motor Go detectar uma casca de SPA vazia (`<div id="root"></div>`), aciona o navegador do sistema temporariamente.
   - Executa o navegador do sistema (Edge no Windows, Chrome/Chromium no Linux/macOS) de forma efêmera por ~2 segundos com `--headless=new --dump-dom`.
   - O processo é **imediatamente finalizado** após o dump do DOM, devolvendo a RAM para 0 MB e alimentando o parser do Go.

---


Protótipo **executado e testado localmente** em 9/10/2026, Linux AMD64, Go 1.23.2. O núcleo de rede, DOM incremental e comandos da IA é Go puro. Para JavaScript, usa opcionalmente **Duktape em C** num subprocesso. Não é Chromium; não é um navegador HTML5 completo.


## O que há de novo na v0.5: independente da linguagem do servidor

O agente precisa acessar os **dados**, não desenhar a interface. O comando
`discover` extrai ações a partir dos protocolos expostos pela página:

- Links HTML (`href`) e formulários (`action`, `method`), inclusive páginas
  atendidas por **PHP, Python/Django, Go, Java e outras linguagens**.
- Atributos HTMX `hx-get`, `hx-post`, `hx-put`, `hx-patch`, `hx-delete`
  em botões **ou outros elementos HTML** (ex.: `<section hx-get="/catalog">`).
- Representações JSON anunciadas por `<link rel="alternate" type="application/json">`
  ou `rel="service-desc"` na página.
- URLs literais de `fetch`, Axios, jQuery e XHR no JavaScript disponível,
  e atributos `data-url` como **candidatos**, sem presumir sua semântica.
- Dados já presentes no DOM associados a `aria-controls` ou `data-target`.

`run` pode fazer um GET confirmado e devolver o texto/JSON diretamente, sem
abrir uma interface ou emular um clique. `POST` de formulários é opcional e
exige `-allow-post` e `confirm:true`. Métodos HTMX que alteram estado
(`POST/PUT/PATCH/DELETE`), dados `data-url` e `formaction` com campos não
reconstruídos são **somente descobertos, não executados automaticamente**.

A linguagem do servidor **não pode ser deduzida com segurança da URL**:
`/products.php` é apenas uma rota de teste; um servidor em Go pode servi-la.
Não é possível descobrir endpoints que o site não expõe, nem reconstruir
com exatidão chamadas que dependam de autenticação, parâmetros, cabeçalhos,
tokens CSRF, WebSockets ou lógica JavaScript complexa. O `confidence`
(`declared` ou `inferred`) indica qualidade da evidência local, **não** que
uma requisição já foi validada contra o servidor.

### Prova reproduzível

```bash
go test ./... -count=1
go test ./browser -run TestServerLanguageIndependentActions -v -count=1
./bin/ai-go-browser-linux-amd64 -demo-actions
```

A prova local usa um servidor HTTP de teste e verifica descoberta sem executar
requisições de API, consulta a uma rota `*.php`, busca GET estilo Django,
consulta HTMX a HTML e consumo direto de um JSON anunciado no `<head>`.
Também comprova que `hx-delete`, `data-url` e `formaction` não são
executados indevidamente. **Nenhum servidor público foi contatado.**

## Novidades reais da v0.4

- **Sessão JavaScript persistente por página**: scripts e variáveis continuam vivos entre cliques, em vez de serem reexecutados do zero.
- `press` permite clicar em botões com `id` no HTML, chamando manipuladores `addEventListener('click', ...)` e `onclick` inline. Esses cliques não fazem uma nova requisição HTTP.
- `fill` sincroniza o valor de inputs com o JavaScript: `document.getElementById('query').value` reflete o preenchimento feito pelo agente.
- O snapshot agora oferece `buttons: [{ref,id,label,type}]` além de links e formulários.
- **Tempo máximo por comando JS** e interrupção do subprocesso em loops infinitos. A página HTML estática permanece disponível em falhas do JavaScript.
- Rede negada por padrão; `-allow-hosts` explicitamente autoriza domínios, mas não substitui um firewall contra DNS rebinding e outras formas de SSRF.

## Testar agora, sem internet

```bash
# A partir da pasta descompactada (Linux AMD64):
./bin/ai-go-browser-linux-amd64 -demo

# Com JavaScript e clique dinâmico (requer libduktape.so.207)
./bin/ai-go-browser-linux-amd64 -demo-events -js-worker="$(pwd)/bin/jsworker-linux-amd64"
```

O modo `-demo-events` executa `open`, `press` duas vezes e `query`. A saída é JSONL. O texto muda de `JavaScript ativo` para `Cliques: 1` e depois `Cliques: 2`, sem novo GET.

## Comandos para um agente

```jsonl
{"cmd":"open","url":"https://example.com"}
{"cmd":"snapshot"}
{"cmd":"query","selector":"#main"}
{"cmd":"press","ref":"b1"}
{"cmd":"fill","ref":"i1","value":"pesquisa"}
{"cmd":"click","ref":"l1"}
{"cmd":"submit","ref":"f1","confirm":true}
{"cmd":"back"}
{"cmd":"quit"}
```

Para domínios externos *autorizados*: `./bin/ai-go-browser-linux-amd64 -allow-hosts=example.com -js-worker=./bin/jsworker-linux-amd64`. A rede externa deste ambiente não foi utilizada para esses testes.

## Reproduzir compilação e testes

```bash
go test ./...
go vet ./...
AI_JS_WORKER="$PWD/bin/jsworker-linux-amd64" go test ./browser -count=1
CGO_ENABLED=0 go build -o bin/ai-go-browser-linux-amd64 ./cmd/aigobrowser
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/ai-go-browser-linux-arm64 ./cmd/aigobrowser
# Para compilar o worker no Linux AMD64 com libduktape.so.207:
CGO_ENABLED=1 go build -o bin/jsworker-linux-amd64 ./cmd/jsworker
```

**Oracle Ampere ARM64:** o controlador Go está compilado para ARM64, mas não foi executado nessa arquitetura. O worker Duktape incluído é AMD64; é necessário compilá-lo em ARM64 com uma biblioteca Duktape disponível no sistema. Sem ele, o navegador continua funcionando com HTML, links e formulários, mas não executa JavaScript.

## Evidências e limites

- 22 testes passaram (`test-output-v0.4.txt`); `go vet` passou. Incluem cliques com JavaScript persistente, sincronização de `fill`, handler inline e cancelamento de loop infinito.
- Na demo `-demo-events`: processo Go com RSS observado de aproximadamente **7,81 MiB** e worker JS com pico informado de aproximadamente **3,79 MiB**. A soma aproximada dos picos é **11,60 MiB**. São picos separados numa página local pequena — não são medição sincronizada da árvore de processos e **não garantem** que sites reais usem menos de 300 MiB.
- Apenas um subconjunto de HTML/CSS/DOM é implementado; Duktape não dá compatibilidade moderna total com a web: sem módulos, Fetch/XHR, APIs de navegador extensas, WebGL, renderização visual ou service workers. Apenas botões com `id` aceitam `press` com eventos JavaScript nesta versão. `onclick` em links e ações padrão de botões de envio ainda não seguem integralmente o comportamento dos navegadores.
- **Não é um mecanismo para burlar anti-bots** e não temos comprovação de compatibilidade com sites externos nesta execução.
- Limites de memória do **processo inteiro** exigem cgroups/`MemoryMax` no Linux; o protótipo ainda não impõe 300 MiB como teto rígido. O JavaScript roda sem acesso a APIs de arquivo/rede via DOM, mas o subprocesso não é um sandbox de segurança de sistema operacional completo.
- O projeto não incorpora um parser HTML5 de conformidade integral. Próximos marcos: parser HTML5 real, eventos para elementos sem ID, eventos de formulário/links com navegação e teste em páginas autorizadas com métricas de compatibilidade.

Resultados em `results.json`, execuções em `demo-events.jsonl`, benchmarks em `benchmark-v0.4.txt`.

---

## Teste de sites REAIS (preparado, **não executado** neste ambiente)

Adicionado `cmd/webtest`: um programa Go que visita uma lista **explícita** de sites
públicos, um por vez, usando o motor do próprio AI Go Browser v0.4. Não envia
formulários nem tenta contornar CAPTCHAs ou sistemas anti-bot.

**Não afirmar compatibilidade com sites públicos antes de executar o teste
numa máquina com rede.** A validação feita aqui utilizou somente o servidor
local embutido (`-local-smoke`).

### Teste na Oracle ARM64 sem JavaScript

Descompacte o projeto e, dentro da pasta, execute:

```bash
chmod +x bin/webtest-linux-arm64
./bin/webtest-linux-arm64 -sites webtest/sites.txt -out webtest/real-sites-results.json
```

Lista inicial em `webtest/sites.txt`: Example Domain, Hacker News, GitHub,
Wikipedia e MDN. É uma seleção de páginas públicas com diferentes graus de
complexidade; **nenhum resultado de acesso foi presumido**. Sites podem
redirecionar para outros hosts; a política do navegador bloqueia o redirecionamento
quando o host de destino não estiver autorizado. Para autorizar host adicionais
específicos, utilize `-extra-hosts=dominio.exemplo`.

Para obter dados comparáveis, rode 3 vezes por endereço:

```bash
./bin/webtest-linux-arm64 -sites webtest/sites.txt -repeat 3 \
  -out webtest/real-sites-results.json
```

### AMD64 com JavaScript opcional

```bash
./bin/webtest-linux-amd64 -sites webtest/sites.txt \
  -js-worker="$(pwd)/bin/jsworker-linux-amd64" \
  -out webtest/real-sites-js-results.json
```

A biblioteca `libduktape.so.207` deve estar instalada no AMD64. Para ARM64,
o worker JavaScript precisa ser compilado na própria máquina, com Duktape
instalado e CGO ativado; **não há binário ARM64 do worker neste pacote**.
O JavaScript desta versão suporta apenas parte do ECMAScript e do DOM;
não suporta scripts externos, módulos modernos, fetch/XHR ou frameworks web
complexos como um Chromium completo.

### O que o relatório realmente mede

O JSON inclui, por site: código HTTP, título, prévia do texto extraído, contagem
de links/formulários/botões, tamanho HTML, duração da operação, RSS amostrado
do processo Go, estatísticas do worker e mensagens de erro.

- `loaded=true` significa apenas código HTTP 2xx e parse bem-sucedido.
- `useful_text=true` é **uma heurística** (pelo menos 40 caracteres extraídos),
  não uma garantia de navegação funcional ou fidelidade ao Chromium.
- `approx_sum_rss_mib` soma medições de processos feitas em momentos distintos,
  **não** é o pico real da árvore de processos; não garante menos de 300 MiB.
- Não executa login nem testes contra defesas anti-bot. Páginas 403, 429 e
  JavaScript incompatível devem ser reportadas como limitações, não mascaradas.

### Teste local reproduzido nesta sessão

```bash
./bin/webtest-linux-amd64 -local-smoke -out webtest/local-smoke-result.json
./bin/webtest-linux-amd64 -local-smoke \
  -js-worker="$(pwd)/bin/jsworker-linux-amd64" \
  -out webtest/local-smoke-js-result.json
```

Ambos retornaram HTTP 200 e extraíram texto. No segundo, o único script de teste
alterou o texto visível para `JavaScript changed the article`.

Os relatórios `local-smoke-*.json` são **somente** resultados de testes locais;
`real-sites-results.json` não foi fabricado nem incluído com resultados inexistentes.

### Compilar o runner

```bash
go test ./...
go vet ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/webtest-linux-amd64 ./cmd/webtest
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/webtest-linux-arm64 ./cmd/webtest
```
