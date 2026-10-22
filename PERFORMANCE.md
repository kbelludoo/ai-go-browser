# Desempenho e uso de memória

Medições locais em 10/10/2026, Windows AMD64, Go 1.27.0, Intel Xeon E5-2697 v3. Os microbenchmarks da etapa inicial usam três execuções com `-benchmem`; os comparativos de tarefas usam três workers novos por motor. As tabelas de tarefas abaixo registram o checkpoint final de fill nativo. Tempos variam com a carga da máquina.

## Microbenchmarks da etapa inicial

| Cenário | Antes | Depois | Interpretação |
| --- | ---: | ---: | --- |
| Texto DOM, HTML Unicode de 475 KB, limite de 256 caracteres | 28,17 MB alocados/op; ~19,2 ms/op | 9,69 MB/op; ~5,61 ms/op | ~66% menos bytes alocados |
| Parser simples, mesmo HTML e limite | 18,96 MB/op; ~13,6 ms/op | 0,476 MB/op; ~0,206 ms/op | ~97% menos bytes alocados |
| Query DOM, nó Unicode de 475 KB, limite de 700 caracteres | 10,22 MB/op; ~8,35 ms/op | 3.936 B/op; ~11,85 µs/op | A consulta para ao obter o prefixo |
| DOM com 1.000 links | 4,35 MB/op; ~8,05 ms/op | 3,72 MB/op; ~7,34 ms/op | ~14% menos bytes alocados |
| Navegação HTTP em memória, HTML com script de 1 MiB | 11,76 MB alocados/op; ~8,20 ms/op | 9,65 MB/op; ~8,19 ms/op | ~18% menos bytes alocados |
| Métricas de memória por comando, Windows | ~38,0 µs/op; 96 B/op | ~9,15 µs/op; 184 B/op | `runtime/metrics` + amostra física via API Windows; o custo inclui `OpenProcess` |
| Navegação de artigo com 250 seções, modos completo vs compacto atuais | 7,44 MB/op; ~9,9 ms/op | ~2,38 MB/op; ~5,9 ms/op | ~68% menos bytes alocados ao evitar gerar Markdown automático |

`MB/op` nesta tabela usa bytes decimais. `MiB` nas flags e na saída JSON usa 1.048.576 bytes.

Esses microbenchmarks medem tempo e soma de alocações por operação. Não medem pico de RAM, árvore de processos ou compatibilidade com sites públicos. O benchmark DOM de texto inclui a produção do Markdown completo; o parser simples não produz Markdown. A última linha compara duas configurações da implementação atual, não duas revisões de código.

## Mudanças

A coleta incremental de texto evita slices com todos os fragmentos, concatenação de todo o conteúdo e conversões integrais para `[]rune`. Continua percorrendo os elementos necessários para expor links, headings, botões e formulários, mesmo depois de truncar o texto. `innerText` interrompe a leitura de uma subárvore ao obter o prefixo necessário para labels e consultas.

O navegador guarda HTML de origem apenas com uma sessão JavaScript ativa. Sem worker, ou se ele falhar, a navegação estática e a descoberta de endpoints usam o DOM. O buffer da resposta é reutilizado quando a sessão precisa dele, sem fazer outra cópia integral. Payloads estruturados pequenos têm cópia própria, para evitar que uma string curta mantenha o HTML inteiro em memória.

O histórico mantém as últimas 100 URLs por padrão. URLs removidas são apagadas do backing array, e uma tentativa de `back` que falha preserva o histórico para outra tentativa. `Close()` remove as referências à página e fecha conexões HTTP ociosas; a recuperação efetiva de memória depende do garbage collector e do sistema operacional.

`RenderHeadless` captura no máximo 5 MiB de DOM e cancela o processo ao exceder esse limite. Há testes com um subprocesso local para limites, overflow, timeout, saída vazia e falha do renderer. Esse limite de buffer não é um limite de RAM para o processo externo.

A navegação JavaScript usa uma sessão Chromium persistente, com eventos, estado de formulário, cookies e histórico nativos. `wait` usa um seletor visível com timeout. `release` encerra os processos de renderização e preserva a última observação. Testes locais verificam interação sem IDs, refs estáveis após inserções, hidratação assíncrona, recuperação de timeout, redirecionamentos, status HTTP, query CSS e guards de POST.

O snapshot Chromium inclui a árvore composta de shadow roots abertos, respeitando slots e associações nativas de formulário. A consulta e a espera percorrem roots com limites de tamanho; editores `contenteditable` recebem texto via CDP `Input.insertText`, com eventos nativos. A descoberta de frames mantém a seleção atual e limita metadados a 100 frames e 256 KiB. Esses testes ampliam a cobertura local, sem comprovar compatibilidade universal.

O preenchimento dos controles de texto (`text`, `email`, `number`, `password`, `search`, `tel`, `url`) e textarea usa a edição nativa do Chromium, com verificação de foco/seleção antes do envio. Select e controles especiais conservam setters e eventos sintéticos. Formatação, limites e cancelamento preservam o resultado nativo; formatter/maxlength podem produzir erro de divergência, e number com `badInput` produz erro. A resposta conserva uma observação atual do resultado quando a leitura é possível; erros nativos e erros de atualização não são descartados. Valores renderizados de editores ficam coerentes entre query e snapshot, preservando parágrafos e linhas vazias, inclusive quando diferem da string solicitada. A soma das anotações desses valores aceita até 5 MiB UTF-8, verificados antes de copiar atributos; isso evita multiplicar texto grande em textboxes aninhados. O limite do DOM completo continua separado.

Frames podem ser selecionados e receber ações, inclusive em outra origem ou em um processo separado. O controlador reutiliza mundos isolados pelo ID único do Chromium, invalida referências após navegação e mantém histórico da página principal. Registros de frames e contextos têm limites, inclusive 200 identidades conhecidas para restaurações do BFCache. A fila CDP é limitada a 256 eventos de até 1 MiB; exceder esses limites gera erro e fecha o transporte, em vez de perder eventos e deixar operações pendentes. Respostas DOM conservam o limite próprio de observação.

A conversão de charset reconhece codificações legadas sem reinterpretar todo UTF-8 malformado como Windows-1252. UTF-8 válido reutiliza o buffer; respostas convertidas também precisam caber no orçamento. Recursos JSON com BOM são classificados após a decodificação para manter a redação de campos sensíveis.

A fila de eventos do CDP (`browser/cdp.go`) passou de 256 slots para a constante `cdpEventQueueCapacity` (8192). O consumidor vivo executa round-trips de CDP — a interceptação de `Fetch.requestPaused` e a configuração de frames em `frameEvent` — enquanto uma página carregada emite milhares de eventos de `Network` e `Page`; com 256 slots, uma rajada dessas enchia a fila e o `readLoop` fechava a conexão inteira. O estouro continua fechando a conexão: descartar eventos em silêncio deixaria navegações sem observação e renderizadores esperando depuração, o que é pior do que uma falha explícita. `openLive` agora verifica se o laço de eventos da sessão terminou (`liveSession.ended`) e, em caso positivo, encerra a sessão morta e inicia um Chromium novo antes de navegar, então uma falha de transporte custa uma navegação em vez de derrubar todas as tarefas seguintes; timeouts de navegação continuam preservando a sessão. `TestLiveOpenRestartsFailedConnection` cobre a recuperação, e os testes de fila preenchem a capacidade real declarada pela constante.

O lançamento headless cria somente a aba controlada por CDP, sem deixar uma aba inicial adicional. Perfis headless temporários usam flags conservadoras de serviços de fundo presentes no Playwright; perfis persistentes mantêm extensões e sincronização. O teste de inicialização verifica exatamente um target de página e um PID válido. Os relatórios Playwright anteriores a essa mudança foram preservados como `playwright-comparison-initial-defaults.*`.

A CLI ativa `-compact=true`: evita construir e reter Markdown a cada navegação e limita as listas observadas a 200 links e 100 headings. O DOM permanece completo e os números omitidos aparecem no JSON. `markdown` gera a representação completa somente para essa resposta; `-compact=false` restaura a saída completa automática. A biblioteca conserva o comportamento completo por padrão.

Formulários com constraints nativas, multipart ou scripts podem migrar do HTTP para Chromium com `-auto-js=true`. A migração exige outra leitura da página, transfere controles editáveis e preserva tokens ocultos novos. O custo dessa transição pertence à operação que a acionou.

## Otimizações de 11/10/2026

Duas etapas, medidas separadamente, ambas em Windows AMD64, Go 1.27.0, Edge Chromium 155.0.4283.45, cinco repetições e os mesmos fixtures `local-v1`. Relatórios em [`benchmark-results/fase1.md`](benchmark-results/fase1.md), [`benchmark-results/fase2.md`](benchmark-results/fase2.md) e [`benchmark-results/baseline_fase1_revert.md`](benchmark-results/baseline_fase1_revert.md), com os JSONs ao lado.

O baseline não é um relatório antigo: é uma rodada própria de hoje, com o código-fonte atual e **apenas as mudanças da etapa 1 revertidas**. Os relatórios históricos `local-comparison.*` e `_ao_vivo_local.*` registraram `Document bytes` de 589 para `spa_click_fill`, enquanto o código atual produz 697 em ambos os lados do A/B — ou seja, aqueles binários anteriores não tinham o código atual e não servem de linha de base para esta rodada.

### Etapa 1: launcher e setup do Chromium

O launcher headless passou a aplicar `--disable-gpu` e um conjunto de switches conservadores presentes no Playwright em perfis temporários, e a porta do DevTools agora é aguardada com polling de 5 ms em vez de 100 ms. Os round-trips de setup (`Page.enable`, `Runtime.enable`, `Page.addScriptToEvaluateOnNewDocument`) passaram a ser enviados em paralelo sobre a mesma conexão, serializados pelo write gate da sessão.

| Métrica local, 5 rodadas, mesmo código exceto a etapa 1 | Antes | Depois | Variação |
| --- | ---: | ---: | ---: |
| Corpus AI Go com startup | 1.931,11 ms | 1.557,04 ms | −19,4% |
| Corpus Chromium/CDP com startup | 2.216,55 ms | 1.782,54 ms | −19,6% |
| Maior amostra RSS Chromium do AI Go | 823,59 MiB | 372,23 MiB | −54,8% |
| Maior amostra RSS Chromium do CDP | 856,76 MiB | 403,22 MiB | −52,9% |
| Startup do Chromium/CDP | 534,46 ms | 358,06 ms | −33,0% |
| Tarefa `spa_click_fill` do AI Go | 1.039,60 ms | 615,04 ms | −40,8% |

Os dois lados declaram `Document bytes` de 697 e `Compact JSON bytes` de 633 para `spa_click_fill` nas duas rodadas, então a comparação é como-por-como. As 35 assertions passam nos dois lados.

O baseline `chromium_cdp` usa o mesmo `LaunchStealth`, então o ganho do launcher aparece dos dois lados e a margem sobre o CDP **diminuiu**: 285,44 ms antes, 225,50 ms depois. Quem quiser vantagem própria do AI Go deve olhar as tarefas que não passam pelo launcher. O `async_dom` ficou estável em torno de 900 ms porque é dominado pelo `setTimeout` de 800 ms do próprio fixture, não pelo navegador.

### Etapa 2: observação `view:"read"`

`{"cmd":"snapshot","view":"read"}` devolve apenas `url`, título, texto visível, status e metadados do motor, sem transferir nem re-parsear o DOM. Em uma sessão Chromium o script de leitura roda na própria página e devolve só o texto; em modo HTTP a leitura é uma projeção do documento já parseado. A normalização de espaços reproduz a de uma observação completa, então uma leitura e uma observação completa têm a mesma forma de texto.

A leitura **não produz referências**: `Links`, `Forms`, `Headings`, `Inputs`, `Buttons` e `Markdown` saem vazios ou ausentes, e a página com referências é preservada em `Document`/`Current` até o próximo refresh completo. Por isso uma leitura não invalida as refs que o agente já usa. Os exemplos estão no `help` da biblioteca.

| Observação em página de 2.000 links, mediana de 3 | Observação completa | `view:"read"` | Fator |
| --- | ---: | ---: | ---: |
| Latência por leitura | 102,2 ms | 3,24 ms | 31,5× menor |
| Payload JSON | 452.428 B | 5.222 B | 86,6× menor |
| Bytes alocados por leitura | 12,99 MB | 33,9 KB | 383× menor |
| Alocações por leitura | 140.282 | 56 | ~2.505× menor |

Produzido por `BenchmarkLiveObservation` em `browser/live_read_bench_test.go`:

```powershell
go test ./browser -run '^$' -bench BenchmarkLiveObservation -benchtime 30x -count=3 -benchmem
```

O payload menor é o custo de tokens por leitura: os tokens de contexto do LLM são proporcionais ao JSON devolvido, e a leitura reduz a umas 87 partes em 100 para o mesmo texto. Isso não mede tokens de modelo real, apenas o JSON que chegaria ao prompt.

**Limites honestos desta medição:** o ganho é proporcional ao tamanho do DOM. Nos fixtures pequenos do corpus a diferença é de poucos milissegundos por leitura, por isso o harness do comparativo **não** foi alterado para usar `view:"read"` — ele continua tirando a observação completa dos dois motores, preservando a comparabilidade com todos os relatórios anteriores. A página de 2.000 links é um cenário sintético que aproxima um site real, não um site real. `innerText` é derivado do layout, então elementos ocultos por folha de estilo saem da leitura; uma observação completa deriva o texto da árvore DOM e pode diferir em páginas com CSS de ocultação, `aria-hidden`, SVG ou roots de sombra. As sete coberturas em `browser/live_read_test.go` verificam igualdade de texto em página simples, ausência de referências, mutação tardia com refs ainda válidas, frame selecionado, projeção HTTP, payload menor e fallback de view desconhecida.

`go test ./... -count=1` e `go vet ./...` passaram após as duas etapas. `compareplaywright` e `comparemodern` **não foram re-mediados nesta rodada**: esta máquina não tem Node nem npm. Seus números continuam válidos para o checkpoint anterior e não comparam a esta revisão.

## Teste em sites reais (100 URLs)

Cem URLs públicas distintas, lista em [`benchmark-results/real-sites-100.txt`](benchmark-results/real-sites-100.txt), origem e critérios em [`benchmark-results/real-sites-candidates.txt`](benchmark-results/real-sites-candidates.txt) (127 candidatas; caíram 5 por URL morta — DNS, 404 ou 500 — e entraram 5 reposições). Cada motor recebe as mesmas 100 URLs, uma sessão por motor e por rodada, ordem alternada entre rodadas, timeout de 20 s por caso, modo `-public` de checagem somente leitura.

### Rodada atual: 5 repetições, após a correção da fila (17/10/2026)

Relatório em [`benchmark-results/real-sites-100-queuefix.md`](benchmark-results/real-sites-100-queuefix.md) e [JSON](benchmark-results/real-sites-100-queuefix.json). Os dois motores usam o mesmo executável Edge/Chromium.

| Métrica, 5 rodadas × 100 URLs | AI Go | Chromium/CDP |
| --- | ---: | ---: |
| Tarefas que passaram | 452/500 (89–92 por rodada) | 444/500 (88–90 por rodada) |
| Rodadas vencidas em assertivas | 5 de 5 | 0 de 5 |
| Média do tempo completo do corpus | 341.172,67 ms | 305.230,42 ms |
| Maior amostra RSS Chromium | 2.068,47 MiB | 2.242,88 MiB |
| Maior amostra RSS do harness Go | 94,68 MiB | 18,73 MiB |
| Maior heap Go do worker | 58,23 MiB | 5,06 MiB |
| Mediana de JSON compacto | 22.169 B | 19.981 B |

**Nenhuma falha por overflow de fila em 500 tarefas.** O bug que derrubou `public_89` (apnews.com) e envenenou as 11 tarefas seguintes na rodada anterior não se repetiu; `public_89` falhou em 1 de 5 rodadas com HTTP 403, e as demais URLs 90–100 não herdaram a falha.

**Latência pareada.** O total do corpus favorece quem falha cedo, então a comparação justa é nos pares em que **os dois** passaram na mesma rodada:

| Nos 436 pares em que ambos passaram | AI Go | Chromium/CDP |
| --- | ---: | ---: |
| Soma do tempo de tarefa | 817.242 ms | 933.482 ms |
| Média | 1.874,4 ms | 2.141,0 ms |
| Mediana | 1.333,2 ms | 1.886,5 ms |
| P90 | 3.646,6 ms | 3.753,6 ms |

AI Go foi mais rápido em 244 dos 436 pares (56,0%), com 12,5% menos tempo agregado e mediana 29,3% menor.

**Tempo total do corpus — Chromium vence, por falha rápida.** A média do corpus completo (341,2 s contra 305,2 s) favorece o Chromium porque ele recebe HTTP 403 em cerca de 230 ms nos sites de desafio anti-bot, enquanto o AI Go espera o timeout de 20 s antes de falhar. São 6 sites × 5 rodadas × ~20 s de esperas do AI Go. Isso é uma desvantagem real de comportamento em sites que bloqueiam cliente automatizado.

**Compatibilidade — AI Go vence agora.** 452/500 contra 444/500, com mais assertivas em todas as rodadas. As 48 falhas do AI Go se distribuem assim:

- **6 sites de desafio anti-bot, 5/5 rodadas cada** (`stackoverflow.com`, `superuser.com`, `serverfault.com`, `askubuntu.com`, `w3.org`, `opensource.org`). O Chromium também falhou nesses (HTTP 403 em ~230 ms); o AI Go estoura o timeout de 20 s em vez de receber o 403.
- **4 falhas de `gnu.org`** com timeout de rede — o Chromium falhou 5/5 no mesmo site.
- **5 falhas de `openlibrary.org`** e **4 de `nasa.gov`** — os dois motores falham nos mesmos sites (timeout de navegação em ambos).
- **1 falha cada** em `python.org`, `php.net` e `ruby-lang.org` (timeout transitório de DNS/TLS; o Chromium falhou 5/5 em `ruby-lang.org`) e 1 `ERR_CONNECTION_RESET` em `postgresql.org`.
- **1 falha com HTTP 403 em `apnews.com`**, o site que derrubava a sessão antiga; o desafio anti-bot varia entre rodadas.

Do lado do Chromium, as 56 falhas incluem `archive.org` (5/5 com timeout; o AI Go passou), `ruby-lang.org` (5/5), `askubuntu.com` (2/5 com 403), `ubuntu.com`, `bbc.co.uk` e `en.wikisource.org` (rate limit 429) com 1–2 falhas cada, além dos mesmos sites de desafio.

**Memória.** O AI Go amostrou 7,8% menos RSS na árvore do Chromium (2.068,47 contra 2.242,88 MiB), mas continua 5,0× mais pesado no processo Go do harness (94,68 contra 18,73 MiB) porque parseia e serializa o DOM em Go, enquanto o baseline só repassa a página. São contadores diferentes e não devem ser somados como pico único.

**Limites desta medição.** São 5 rodadas na mesma máquina e na mesma rede, com variância diária de carga e de bloqueio anti-bot; a dispersão entre dias não foi medida. `Document bytes` e `Compact JSON bytes` em página real chegam a megabytes e dezenas de kilobytes: o ganho percentual do corpus local não se transfere automaticamente para a web. A lista foi filtrada por alcance com requisição HTTP antes do teste, o que a enviesa para sites que não bloqueiam cliente automatizado. Nenhuma conta, formulário ou ação foi executada; o modo `-public` só lê. Nenhum token de modelo foi medido. Os 500 pares de tarefas não são independentes entre si (mesmas URLs em rodadas repetidas), então os números descrevem esta lista e esta semana, não a web em geral.

### Histórico: rodada anterior, 1 repetição, antes da correção (10/10/2026)

Relatório em [`benchmark-results/real-sites-100.md`](benchmark-results/real-sites-100.md) e [JSON](benchmark-results/real-sites-100.json). Uma repetição por motor, sem alternância de ordem entre repetições — os intervalos são amplos e valem como ordem de grandeza, não como diferença estatisticamente confirmada.

| Métrica, 100 URLs, 1 repetição | AI Go | Chromium/CDP |
| --- | ---: | ---: |
| Tarefas que passaram | 81/100 | 87/100 |
| Tempo total do corpus | 257.640,73 ms | 315.054,81 ms |
| Maior amostra RSS Chromium | 2.441,42 MiB | 3.031,38 MiB |
| Maior amostra RSS do harness Go | 94,26 MiB | 16,68 MiB |
| Maior heap Go do worker | 58,05 MiB | 3,35 MiB |
| Mediana de JSON compacto | 24.849 B | 20.120 B |

Nos 75 sites em que **os dois** passaram: soma 114.172 ms contra 158.495 ms (AI Go 28,0% menor), média 1.522 ms contra 2.113 ms, mediana 1.170 ms contra 2.047 ms, P90 3.487 ms contra 4.967 ms; AI Go mais rápido em 45 dos 75 (60,0%).

As 19 falhas do AI Go daquela rodada se dividiam em: **12 causadas por um único bug de sessão** — `public_89` a `public_100` falharam com `cdp event queue overflow (capacity 256)` e, a partir daí, a conexão CDP morria de vez (`task_ms` = 0 nas 11 seguintes); 5 timeouts nos mesmos sites de proteção contra bot que o Chromium também não passou; e 2 falhas de servidor/rede (`gutenberg.org` com HTTP 504 nos dois motores e `openlibrary.org` com timeout). Dos 13 erros do Chromium, 6 eram HTTP 403 da mesma leva, 2 falha de DNS transitória, 3 timeout e 1 504. Esse bug de fila foi corrigido em 17/10/2026 (ver a seção de mudanças acima) e a rodada atual mede o código corrigido.

## Reproduzir

Na pasta do projeto:

```powershell
go test ./... -count=1
go vet ./...
go test ./browser -run '^$' -bench 'Benchmark(DOM1000Links|TextLimit|DOMQueryTextLimit|NavigateLargeHTML|NavigateSnapshotModes|MemoryStats|AnalyzeTechStackLargeHTML|ExtractStructuredDataLargeHTML)$' -benchmem -count=3
go build -o bin/aigobrowser.exe ./cmd/aigobrowser
.\bin\aigobrowser.exe -auto-js=false -memory-limit-mib=64 -max-history=50 -max-text=3000 -demo
.\bin\aigobrowser.exe -auto-js=false -memory-limit-mib=64 -demo-actions
go build -o bin/comparebrowser.exe ./cmd/comparebrowser
$u = Get-Content benchmark-results/real-sites-100.txt
$a = @('-public'); foreach ($x in $u) { $a += @('-url', $x) }
$a += @('-runs','5','-timeout','20s','-json','benchmark-results/real-sites-100-queuefix.json','-markdown','benchmark-results/real-sites-100-queuefix.md')
.\bin\comparebrowser.exe @a
```

Os microbenchmarks de navegação usam um transporte HTTP local em memória; as demos usam um servidor em `127.0.0.1`. Os testes Chromium usam um servidor local e o navegador instalado. O teste de sites reais usa a rede externa, leva em torno de 10 minutos com `-runs 1` e cerca de uma hora com `-runs 5` nesta máquina, e sai com código 1 quando alguma tarefa falha — o relatório é gravado antes do código de saída.

## Comparativo de tarefas e páginas públicas

A referência principal solicitada é [Playwright com Chromium](cmd/compareplaywright/README.md). O comando reutiliza o corpus local e o executável Chromium da comparação CDP, registrando as versões reais de Node e Playwright. O tempo completo do worker inclui setup e encerramento nos dois lados; Go, Node e Chromium aparecem separados nas amostras de memória. O comparativo CDP abaixo permanece como evidência complementar, com outro harness.

[Validação final de fill nativo contra Playwright](benchmark-results/playwright-native-fill-validation.md), com [amostras JSON](benchmark-results/playwright-native-fill-validation.json): três rodadas, os mesmos sete fixtures de `local-v1`, Edge Chromium 155.0.4283.45, Playwright 1.62.1 e Node v24.19.0. O checkpoint produziu:

| Métrica local | AI Go | Playwright/Chromium |
| --- | ---: | ---: |
| Assertions | 21/21 | 21/21 |
| Tempo completo médio do worker | 2.954,29 ms | 3.696,33 ms |
| Tempo médio do corpus com setup do motor | 2.210,39 ms | 3.175,68 ms |
| JSON compacto mediano | 313 B | 236 B |
| Maior amostra da soma RSS Chromium | 494,43 MiB | 524,24 MiB |
| Maior amostra RSS do worker Go | 30,95 MiB | 18,07 MiB |
| Maior amostra de heap Go | 16,77 MiB | 5,22 MiB |
| Maior amostra RSS Node | não usado | 118,01 MiB |
| Maior amostra de heap V8 Node | não usado | 55,74 MiB |

O worker completo levou 20,1% menos tempo neste corpus; a maior soma RSS Chromium amostrada foi 5,7% menor. Playwright produziu JSON menor, com outro schema. As amostras Go, Node e Chromium são separadas e seus máximos não devem ser somados como pico físico simultâneo. O ganho observado não prova superioridade geral.

A [validação após frames](benchmark-results/playwright-frame-validation.md) registra o checkpoint anterior ao fill nativo. O [comparativo após ajuste do launcher](benchmark-results/playwright-comparison.md) e a [validação da recuperação](benchmark-results/playwright-recovery-validation.md) preservam rodadas anteriores ao suporte interativo a frames. Na mudança do launcher, a maior soma amostrada de working sets Chromium do AI Go caiu de 860,54 MiB na configuração inicial para 490,39 MiB ao remover a aba extra e aplicar as flags de fundo em conjunto (~43%). Essa comparação entre rodadas não isola o efeito de cada flag nem mede pico contínuo. Os tempos absolutos variaram entre rodadas; não use medições de momentos diferentes como comparação controlada de revisões.

```powershell
go build -o bin/compareplaywright.exe ./cmd/compareplaywright
.\bin\compareplaywright.exe -runs 3 -json benchmark-results/playwright-native-fill-validation.json -markdown benchmark-results/playwright-native-fill-validation.md
```

[Corpus moderno](cmd/comparemodern/README.md), com [relatório final de fill nativo](benchmark-results/modern-native-fill-validation.md) e [amostras JSON](benchmark-results/modern-native-fill-validation.json): cinco tarefas adicionais, todas com Chromium, em três rodadas. Os fixtures `local-modern-v1` continuam iguais: shadow DOM/slots, editor nativo, validação e multipart, frame de mesma origem e frame de outra origem possuem assertions de eventos/valores; o servidor verifica as submissões esperadas e seu conteúdo. O oracle AI Go agora também exige o `Page.FrameRef` do filho e confirma ao fim que a página principal conserva ID/URL e que o filho original continua selecionado.

| Métrica moderna | AI Go | Playwright/Chromium |
| --- | ---: | ---: |
| Assertions | 15/15 | 15/15 |
| Tempo completo médio do worker | 3.134,30 ms | 3.161,67 ms |
| Tempo médio do corpus com setup do motor | 2.381,03 ms | 2.686,55 ms |
| JSON compacto mediano | 652 B | 385 B |
| Maior amostra RSS Chromium | 582,58 MiB | 577,96 MiB |
| Maior amostra RSS do worker Go | 16,93 MiB | 11,28 MiB |
| Maior amostra de heap Go | 2,61 MiB | 1,24 MiB |
| Maior amostra RSS Node | não usado | 122,71 MiB |
| Maior amostra de heap V8 Node | não usado | 57,60 MiB |

Os tempos completos ficaram próximos (AI Go ~0,9% menor); a diferença não estabelece vantagem prática neste corpus, e a soma RSS Chromium do AI Go foi ligeiramente maior. O fill de texto agora é nativo, mas os fixtures modernos preservam suas assertions: exigem input confiável em contenteditable e cliques confiáveis, sem testar `isTrusted` nos inputs comuns. A confiança desses eventos, parágrafos de editores, veto, foco e erros de edição têm cobertura própria na suíte do navegador. Select e controles especiais ainda usam setters/eventos sintéticos. Os schemas, eventos e custos de startup permanecem distintos.

O [relatório moderno anterior](benchmark-results/modern-comparison.md) registra a implementação anterior ao fill nativo e ao oracle reforçado; `modern-comparison-prevalidation.*` conserva a rodada anterior às correções da suíte completa. A [smoke que falhou](benchmark-results/modern-smoke-context-failure.md) preserva o timeout em contextos de frames durante o desenvolvimento; ela não representa a versão corrigida.

```powershell
go build -o bin/comparemodern.exe ./cmd/comparemodern
.\bin\comparemodern.exe -runs 3 -json benchmark-results/modern-native-fill-validation.json -markdown benchmark-results/modern-native-fill-validation.md
```

[comparebrowser](cmd/comparebrowser/README.md) compara tarefas verificáveis contra Chromium usando o mesmo executável e workers separados. O baseline `NavigateReady` espera o carregamento da navegação corrente sem uma espera fixa adicional. Os snapshots das duas implementações usam texto limitado a 5.000 caracteres, até 200 links e 100 headings, com diferenças de schema descritas no relatório.

[Relatório local](benchmark-results/local-comparison.md): três rodadas de sete tarefas, com assertions de resultado em leitura estática, HTML grande, Unicode, formulário GET, cookies/link, fill/click em SPA e DOM assíncrono. [Relatório público](benchmark-results/public-comparison.md): três rodadas de sete páginas públicas, apenas leitura, com status, título, trecho de texto e URL final. Os arquivos JSON guardam os dados por caso.

| Rodadas CDP históricas | AI Go | Chromium/CDP |
| --- | ---: | ---: |
| Assertions locais | 21/21 | 21/21 |
| Tempo médio do corpus local, com startup | 1.974,84 ms | 2.012,95 ms |
| Assertions públicas | 21/21 | 21/21 |
| Tempo médio do corpus público, com startup | 3.945,95 ms | 12.277,74 ms |
| Mediana de bytes do snapshot público | 25.149 | 18.255 |

Essas rodadas CDP precedem a remoção da aba inicial adicional e a configuração de serviços de fundo do launcher; seus arquivos conservam a evidência daquele estado. A comparação principal Playwright tem relatório e metodologia próprios.

O tempo local total ficou próximo. A leitura HTTP das páginas públicas evitou iniciar Chromium, que também carrega recursos da página. O snapshot do AI Go permaneceu maior e seu heap Go amostrado foi maior que o do harness CDP; vantagens e custos constam dos relatórios. Os testes públicos cobrem leitura destas páginas, enquanto interações JS são verificadas nos fixtures locais.

Os relatórios anteriores ao modo compacto foram preservados como `*-full-snapshot.*`; o comparativo inicial com pausa fixa do wrapper está em `local-comparison-navigation-delay.*`. A execução `public-comparison-validation-overlap.*` coincidiu com a suíte de testes e foi substituída pela rodada principal sem essa carga. Rodadas públicas dependem de rede, cache e ordem e não provam superioridade geral. Esses testes também não medem acerto de um agente com LLM, custos de inferência, contagem de tokens ou acesso universal.

O worker Duktape incluído no pacote é Linux AMD64. As integrações de eventos com esse worker são puladas neste Windows sem `AI_JS_WORKER`; os testes de parser, navegação HTTP, descoberta e renderer local são independentes dele.

## Perfil com menor consumo

```powershell
.\bin\aigobrowser.exe -auto-js=false -memory-limit-mib=64 -max-history=50 -max-text=3000
```

- `-auto-js=false`: mantém a navegação automática em Go. A renderização JS explícita continua disponível.
- `-memory-limit-mib=64`: solicita um limite flexível ao runtime. Ele pode ser ultrapassado; valores muito baixos aumentam trabalho do GC. O limite não inclui Chromium, Duktape ou toda a memória do sistema operacional.
- `-max-history=50`: limita URLs retidas; `0` desativa e `-1` permite histórico ilimitado.
- `-max-text=3000`: limita apenas o campo `text`; Markdown, DOM e listas de ações têm seus próprios comportamentos.

Na saída JSON, `heap_mib` é uma amostra do heap Go e `go_sys_mib` é a memória obtida pelo runtime. `rss_source=windows_working_set` indica medição do working set via API Windows; `proc_status` indica RSS do processo Go no Linux. `go_sys_estimate` identifica o fallback quando a plataforma não fornece a leitura física. Esses campos não incluem os processos JavaScript.

O comparativo amostra a árvore de processos Chromium separadamente. A soma dos working sets pode contar páginas compartilhadas mais de uma vez. Máximos amostrados depois das tarefas não são picos continuamente observados; valores indisponíveis são `null`, não zero.
