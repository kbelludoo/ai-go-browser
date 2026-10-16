# Validação final de fill nativo e frames

Checkpoint final de fill nativo preservado no commit funcional `b54efc9` e na referência `codex/validated-native-fill`, validado em 10/10/2026, Windows AMD64, Go 1.27.0, Edge Chromium 155.0.4283.45, Node v24.19.0 e Playwright 1.62.1. A recuperação anterior está salva no commit local `9cef3a0` e na referência `codex/recovered-browser`; o checkpoint anterior de frames está em `codex/validated-frames`. A atualização e a restauração do checkout original são protegidas por cópia de segurança e verificação dos hashes dos arquivos afetados.

## Verificações executadas

- `go test ./... -count=1`: passou, incluindo browser (85,401 s), comandos e benchmarks. A suíte Chromium cobre shadow DOM aberto, slots, contenteditable, formulários, frames aninhados, origens diferentes, troca de renderer, remoção, referências, política de hosts, status e histórico/BFCache.
- `go vet ./...`: passou.
- Fill nativo de texto Unicode, textarea, email/number/password/search/tel/url, shadow DOM e frames: passou, com testes de veto de `beforeinput`, desvio de foco e readonly. Campos de texto recebem edição nativa CDP; select e controles especiais conservam setters/eventos sintéticos. Os testes verificam confiança dos eventos nos campos comuns, enquanto os fixtures do benchmark preservam suas assertions anteriores.
- Contenteditable rico preserva o modelo de parágrafos do Chromium, incluindo linhas vazias e resultado de edição cancelada. Query e `Page.Inputs.Value` observam o mesmo texto renderizado, sem achatar filhos para igualar a string solicitada. Formatter/maxlength e `badInput` em number mantêm observação atual quando possível e relatam o erro; a combinação com falhas de atualização não descarta a causa nativa.
- O orçamento agregado de anotações de texto renderizado é 5 MiB UTF-8, verificado antes da cópia, com regressão de textboxes aninhados. O orçamento do DOM completo permanece separado.
- Na validação anterior de frames, as regressões `TestLivePOSTConfirmationAndSessionContext`, `TestFormPromotionNativeValidationAndMultipart` e `TestFormPromotionPreservesValuesAndUsesFreshHiddenToken` passaram em três repetições após a correção; também passaram na suíte atual. `submit` relata erro de validação; `press` deixa o navegador disparar o evento `invalid` nativo sem duplicá-lo.
- Fences de frames em attach/replacement, cache de contextos, restauração BFCache e merge de árvores têm testes controlados. O oracle moderno exige o `Page.FrameRef` do filho selecionado e, após as ações, confirma ID/URL original do main e identidade/seleção do filho. Regressões rejeitam navegar a aba principal para a URL do filho, trocar/remover frames e devolver seleção incorreta.
- Testes CDP com WebSocket local: fila cheia entrega respostas; overflow e eventos maiores que 1 MiB fecham a conexão e liberam pendências; respostas maiores que o limite de eventos continuam disponíveis. Falhas nas conexões de página e browser chegam à sessão com sua causa. Nenhum site externo participa desses testes.
- Builds Windows AMD64 de `aigobrowser`, `compareplaywright` e `comparemodern`: passaram. Cross-compilação Linux AMD64 dos mesmos comandos: passou; não houve execução desses binários Linux neste Windows.

O worker Duktape distribuído é Linux AMD64. Testes que dependem dele são pulados neste Windows sem `AI_JS_WORKER`. `CGO_ENABLED=0` neste ambiente; os testes concorrentes usam um transporte controlado, sem execução do race detector.

## Comparação atual

Os dois motores usam o mesmo executável Chromium, workers novos, ordem alternada e três repetições por corpus. O tempo completo inclui início e encerramento dos processos; as amostras de memória dos componentes permanecem separadas.

| Corpus | AI Go | Playwright/Chromium | Tempo médio completo AI Go / Playwright |
| --- | ---: | ---: | ---: |
| Original, sete tarefas | 21/21 | 21/21 | 2.954,29 / 3.696,33 ms |
| Moderno, cinco tarefas | 15/15 | 15/15 | 3.134,30 / 3.161,67 ms |

No corpus original, AI Go levou cerca de 20,1% menos tempo. No moderno, os tempos ficaram próximos (AI Go ~0,9% menor). A maior amostra RSS Chromium foi 494,43 / 524,24 MiB no original e 582,58 / 577,96 MiB no moderno. São somas de working sets amostradas após tarefas, que podem contar páginas compartilhadas mais de uma vez; não são picos contínuos nem memória física total. Máximos de Go, Node e Chromium não devem ser somados como pico simultâneo. JSONs têm schemas diferentes, e bytes não são tokens de modelo; as medianas foram 313 / 236 B no original e 652 / 385 B no moderno. As tabelas em [PERFORMANCE.md](PERFORMANCE.md) separam RSS e heap dos componentes.

Relatórios finais: [original com fill nativo](benchmark-results/playwright-native-fill-validation.md) e [JSON](benchmark-results/playwright-native-fill-validation.json); [moderno com fill nativo e oracle reforçado](benchmark-results/modern-native-fill-validation.md) e [JSON](benchmark-results/modern-native-fill-validation.json). Ambos usam os fixtures anteriores, sem mudança de valores ou das assertions de eventos/HTTP. O adapter AI Go acrescenta a verificação de identidade/seleção de frames; o moderno também verifica contadores de eventos e conteúdo no documento selecionado, além de submissões contadas e validadas pelo servidor local.

Histórico preservado: [original após frames](benchmark-results/playwright-frame-validation.md), [moderno antes do fill nativo](benchmark-results/modern-comparison.md), [recuperação anterior](benchmark-results/playwright-recovery-validation.md) e [launcher anterior](benchmark-results/playwright-comparison.md). `modern-smoke-context-failure.*` conserva os timeouts de desenvolvimento; `modern-comparison-prevalidation.*` registra a rodada anterior à correção das regressões da suíte completa. Essas rodadas não são medições do checkpoint atual nem uma comparação controlada entre revisões.

As otimizações de 11/10/2026 (launcher e porta do DevTools, round-trips de setup e observação `view:"read"`) estão em [PERFORMANCE.md](PERFORMANCE.md), com A/B próprio da mesma data. `compareplaywright` e `comparemodern` **não foram re-mediados** depois delas: esta máquina não tem Node nem npm. As tabelas acima continuam válidas para o checkpoint de 10/10/2026 e não comparam a esta revisão.

## Escopo ainda não comprovado

Esses resultados ampliam a compatibilidade e demonstram ganhos específicos, mas não comprovam acesso a qualquer site nem que o navegador seja o melhor para toda IA. O corpus moderno não mede agentes com LLM, uploads, shadow roots fechados, autenticação entre origens ou sites públicos. Frames aninhados, parágrafos ricos de contenteditable, veto/foco e erros de edição têm testes próprios; não fazem parte das cinco tarefas do comparativo moderno. Os fixtures históricos e atuais não cobram `isTrusted` nos inputs comuns; essa cobertura vem da suíte do navegador. Select e controles especiais ainda usam setters/eventos sintéticos. Rotação, skew, perspectiva, reflexão e motion paths de frames são rejeitados explicitamente. Jev permanece uma proposta opcional, sem chamada autenticada.

```powershell
go test ./... -count=1
go vet ./...
go build -o bin/compareplaywright.exe ./cmd/compareplaywright
.\bin\compareplaywright.exe -runs 3 -json benchmark-results/playwright-native-fill-validation.json -markdown benchmark-results/playwright-native-fill-validation.md
go build -o bin/comparemodern.exe ./cmd/comparemodern
.\bin\comparemodern.exe -runs 3 -json benchmark-results/modern-native-fill-validation.json -markdown benchmark-results/modern-native-fill-validation.md
```
