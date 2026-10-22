# AI Go Browser — Plano de independência e evolução

**Data:** 22 de outubro de 2026  
**Repositório:** https://github.com/kbelludoo/ai-go-browser  
**Status:** planejamento técnico baseado em leitura inicial de `README.md`, `go.mod`, `browser/browser.go` e `cmd/aigobrowser/main.go`. **Não** representa auditoria integral nem execução de testes.

## 1. Objetivo decidido

Criar um navegador **sem interface gráfica**, independente, escrito principalmente em **Go**, para agentes de IA e automação HTTP/WebSocket. O produto final **não deve precisar instalar, abrir ou controlar Chrome, Chromium, Firefox ou outro navegador** — nem como fallback. Consumir pouca CPU e RAM é requisito importante, especialmente para uma VPS Oracle de recursos limitados.

O alvo não é reproduzir todos os recursos visuais de um navegador moderno: priorizar acesso a sites, execução de scripts necessários, DOM suficiente para automação, identificação de ações e comunicação HTTP/WebSocket. Falhas por recursos não suportados devem ser explícitas; não simular sucesso.

## 2. O que já existe (confirmado na leitura de código)

- Cliente `net/http` em Go com cookies, redirecionamentos, timeouts e reuso de conexões.
- Carregamento e interpretação de páginas HTML, extração de conteúdo e representação da página.
- Comandos de navegação e automação (abrir, descobrir ações, consultar, clicar, preencher, enviar formulários).
- Limites já disponíveis: corpo HTML de 5 MiB (`MaxBody`), histórico configurável, opção de limite de memória do runtime.
- Integração com execução JavaScript e modo híbrido que pode acionar navegador externo; `go.mod` inclui `wazero`.
- Verificação de URLs com lista de hosts, mas ainda há ressalva explícita no código de que isso **não é proteção completa contra SSRF**.
- A CLI define `-allow-all` como `true` por padrão quando `-allow-hosts` não foi informado.

Observação: o README cita métricas de desempenho, camadas e resultados de testes externos. Essas alegações **não foram reproduzidas** nesta revisão.

## 3. O que falta ou precisa ser comprovado para independência real

### A. Remover a dependência operacional de navegadores externos — prioridade P0
- Criar o modo `native-only` (ou equivalente) que impeça qualquer chamada a navegador externo, inclusive fallback automático.
- Separar claramente caminhos de execução: rede/HTML/JS próprio versus integrações legadas.
- Expor relatório de capacidades e mensagens de recurso não suportado; nunca acionar fallback silencioso.
- Testar em máquina/contêiner **sem navegador instalado**.

**Critério de aceite:** fluxo básico de navegação e automação executa sem processos externos de navegador; páginas não suportadas retornam erro identificado.

### B. Executar JavaScript no ambiente próprio — P0/P1
- Verificar o que a implementação atual de JavaScript realmente suporta e como é invocada.
- Escolher estratégia clara: evoluir QuickJS/Wazero já descrito no projeto ou avaliar Goja; evitar dois motores sem justificativa.
- Implementar limites por tempo, memória, número de tarefas e cancelamento por contexto.
- Isolar por origem/sessão conforme necessário; garantir encerramento de sessões e descarte de recursos.

**Critério:** scripts simples, timers e callbacks funcionam com testes reproduzíveis e limites aplicados.

### C. DOM, eventos e APIs web mínimas — P1
- Integrar `document`, `window`, `querySelector`, `getElementById`, `textContent`, atributos e mutações ao motor JS.
- Implementar `addEventListener`, propagação básica de eventos, clique, input, submit e fila de tarefas.
- Integrar `fetch`, Promises, XMLHttpRequest quando necessário, cookies, URL, storage e WebSocket.
- Expandir por compatibilidade medida em casos de teste; **não** prometer padrão web completo.

**Critério:** uma página local com clique que dispara `fetch`, recebe JSON e atualiza DOM funciona sem navegador externo.

### D. Segurança de rede e execução — P0
- Alterar o padrão `-allow-all` para `false` em contexto de agentes; manter opção explícita para usuários que assumirem o risco.
- Bloquear IPs privados, loopback, link-local e endpoints de metadados por padrão quando agentes não confiáveis navegam; verificar resolução DNS, redirecionamentos e endereço efetivo no dial.
- Definir política para uploads, POST e ações com efeitos colaterais; exigir autorização explícita por operação quando apropriado.
- Limitar tamanho de respostas, descompressão, conexões, scripts, páginas e tempo total.
- Tratar conteúdo da web como **dados não confiáveis**; comandos encontrados em páginas não são instruções para o agente.

**Critério:** testes SSRF, redirecionamentos, DNS rebinding, corpo gigante, loop JS infinito e abuso de WebSocket são negados ou terminam dentro dos limites.

### E. Recursos e observabilidade — P1
- Medir RSS real do processo, heap, picos, goroutines e latência por operação.
- Comparar HTTP-only, HTML, JS e páginas com WebSocket usando a mesma carga.
- Fazer testes de vazamento em ciclos abrir/executar/fechar e em centenas de sessões sequenciais.
- Manter orçamento de RAM por cenário, sem afirmar 10 MB, 75 MB ou outra meta como resultado garantido.

**Critério:** benchmarks automatizados com ambiente, versão Go, dataset, percentis e pico de RSS registrados.

## 4. Referências para aprender, sem copiar um navegador inteiro

- **Go `net/http`:** transport, pooling, timeouts, redirects e segurança do cliente.
- **Colly:** controle de requisições, sessões, coleta e limitação de carga.
- **Goja:** exemplo de engine JS integrada ao Go; avaliar como alternativa, não como dependência obrigatória.
- **Wazero:** execução WebAssembly com controle de recursos; já aparece nas dependências.
- **Servo:** referência conceitual para DOM, eventos e isolamento; não pressupõe migração para Rust.
- **Biscuit:** pesquisa sobre Go em sistemas operacionais; menos útil para implementar recursos de navegador, portanto prioridade baixa.

## 5. Próximas tarefas, em ordem

1. **P0:** inventariar caminhos que executam navegador externo e criar um modo nativo estrito.
2. **P0:** corrigir padrão de permissões da CLI e adicionar testes de bloqueio SSRF.
3. **P0:** adicionar suíte local de compatibilidade: HTML, cookies, formulários, clique → JS → fetch → JSON → DOM.
4. **P1:** consolidar uma única estratégia de engine JS com limites verificáveis.
5. **P1:** implementar DOM/eventos/APIs web por testes de casos reais.
6. **P1:** criar benchmark de memória/CPU e relatórios reproduzíveis.
7. **P2:** ampliar compatibilidade com sites modernos somente quando houver necessidade demonstrada.

## 6. Decisões e limites

- **Não** mudar o compilador Go nem iniciar um fork da linguagem.
- **Não** transformar o projeto em extensão de Firefox ou em controlador de outro navegador.
- **Não** aceitar fallback silencioso para outro navegador no modo independente.
- **Não** alegar que o projeto já roda toda aplicação JavaScript: isso precisa ser demonstrado por testes.
- Evitar implementar CSS/layout/renderização gráfica se o requisito continuar sendo automação sem UI.
- Modificações no código-fonte devem ocorrer separadamente, com testes e commits pequenos.

## 7. Primeiro marco sugerido

**Marco: “Navegador Go nativo mínimo”**

Rodar sem navegador externo instalado, visitar página local, manter cookie, interpretar HTML, identificar botão, executar JS necessário para disparar `fetch`, receber JSON e atualizar DOM. Mostrar erro claro caso a página exija uma API ainda não implementada. Registrar o RSS máximo e executar os testes sob limite de tempo.

---

*Documento criado a partir da conversa e de inspeção parcial do repositório. É um plano de trabalho, não uma declaração de funcionalidades já concluídas.*
