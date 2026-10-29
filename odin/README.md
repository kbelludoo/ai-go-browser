# AI Odin Browser — port incremental do AI Go Browser

**Estado:** protótipo funcional de navegador semântico **estático**, escrito
em Odin. O Go continua sendo a implementação principal. Este diretório não é
uma conversão completa do repositório.

## Funcionalidades implementadas

| Componente | Odin | Limitações |
|---|---|---|
| Compactação MarkdownToCSO, ToCSO e estimativa de tokens | Implementado | Não é tokenizador BPE |
| Semantic_Page, links e headings | Implementado | Representação menor que a estrutura Go |
| Títulos, texto, links, inputs e forms de HTML | Implementado | Scanner limitado; não é um parser HTML5 compatível |
| Referências de link l1, l2 | Implementado | Navegação estática com GET |
| Ações de botões antes do clique | Parcial | Targets de form, formaction, htmx e data-url; sem executar scripts |
| Formulários GET f1, f2 | Implementado | POST recusado, sem estado de sessão |
| JSON-LD, __NEXT_DATA__, __NUXT_DATA__ | Implementado | JSON validado, extrai payload bruto |
| Exportação JSON para agentes de IA | Implementado | Esquema Odin menor que o JSON Go |
| Transporte HTTP/HTTPS | Implementado | libcurl, redirecionamentos explícitos até 5 saltos |
| Cookies e sessões HTTP | Implementado | Cookie jar em memória, limitado à execução atual |
| Agent mode JSON Lines | Implementado | Sessão persistente, mas não é MCP |
| Seletores CSS estáticos e query | Implementado | Bounded scanner (#id, .class, [attr], descendant); máx 32 resultados |
| Histórico e comando back | Implementado | Stack de até 32 URLs na sessão do agente |
| Proteção SSRF | Parcial | Rejeita IPs IPv4 privados/reservados em cada conexão; sem IPv6 restrito |
| JavaScript, CDP, DOM dinâmico, SQLite e MCP | Não implementado | Permanecem no navegador Go |

## Dependências

- Odin (https://odin-lang.org/docs/install/)
- libcurl instalada para linkagem. Ubuntu: sudo apt install libcurl4-openssl-dev.
- O CI atual valida Linux. Outras plataformas exigem sua respectiva instalação
  da biblioteca vendor:curl.

## Compilar e executar na raiz do repositório

    odin test odin
    odin build odin -out:ai-odin-browser

    # Compactação de Markdown para agentes
    ./ai-odin-browser cso README.md
    ./ai-odin-browser tokens README.md

    # Página HTML/HTTPS e links de referência
    ./ai-odin-browser open https://example.com
    ./ai-odin-browser follow https://example.com l1

    # Formulário GET explícito, apenas campos conhecidos
    ./ai-odin-browser submit https://example.com f1 q=odin

    # Payloads SSR, sem executar JavaScript
    ./ai-odin-browser hydrate https://example.com

    # Saída estruturada JSON para agentes
    ./ai-odin-browser export https://example.com

    # Modo interativo JSON Lines para agentes (stdin/stdout)
    ./ai-odin-browser agent

No modo agente, envie um JSON por linha e leia uma resposta JSON por linha:

    {"cmd":"open","url":"https://example.com"}
    {"cmd":"snapshot"}
    {"cmd":"follow","ref":"l1"} (ou {"cmd":"click","ref":"l1"})
    {"cmd":"query","selector":"main a"}
    {"cmd":"back"}
    {"cmd":"submit","ref":"f1","fields":["q=odin"]}
    {"cmd":"quit"}

A saída JSON de `export` e `snapshot` inclui ações declaradas em botões
(`action`, `method`, `source`, `disabled`) para inspeção pré-clique.
O navegador conserva cookies, histórico, URL e página atuais no processo.
Não salva sessões em disco. Erros em comandos não destroem a página anterior.
Uma sessão JSONL é sequencial; não é um motor JavaScript ou protocolo MCP.

Por padrão, IPs privados e de loopback são bloqueados. Para testar um
servidor local sob seu controle, adicione --allow-private ao final.
Isso relaxa a proteção SSRF e não deve ser usado em URLs de terceiros.

## Validação

O workflow .github/workflows/odin.yml instala Odin e libcurl, testa e
compila. O smoke inicia um servidor HTTP local, busca páginas, segue
um link, submete um formulário GET, extrai JSON-LD/Next/Nuxt,
valida a saída estruturada JSON e testa cookies/redirects com servidor
local. O teste de agente JSONL mantém página, cookies e histórico entre
comandos (incluindo `query`, `click` e `back`).

Os testes **não demonstram paridade** com Go, nem comprovam capacidade
de navegar em todo site público ou suportar apps dependentes de JS.

## Segurança e limitações

- HTTP GET apenas, sem submissão POST ou execução de scripts.
- Botões com destino HTTP declarado mostram metadados, sem executar POST
  ou chamadas JavaScript implicitamente; botão desabilitado é sinalizado.
- Verificação TLS ligada, somente protocolos http/https, sem proxy e
  com até cinco redirecionamentos manuais, cada um revalidado.
- Timeout de conexão 4 segundos; transferência 12 segundos.
- Limites: corpo de 2 MiB, scanner de 50 mil tags, máximo de 2048 itens.
- O conteúdo extraído da web não é confiável; nunca deve ser tratado
  como instruções ao agente.
- Resolução de URLs relativas simplificada (ainda sem canonicalização
  RFC3986 completa), sem layout, sem DOM dinâmico.
- A CLI usa arena por operação para parsing e CSO. Testes de longa duração
  e auditorias de memória não estão concluídos. Não assumir os
  mesmos números de RAM ou velocidade do Go.

## Etapas ainda necessárias para reescrita completa

1. Parser HTML5 completo e testes comparativos de DOM, formulários e encoding.
2. Melhorar a cobertura de cookies/sessões, classificação IPv6 segura,
   URLs relativas complexas, cache, persistência opcional e ataques de DNS.
3. Actions, navegação e estado de sessão compatíveis com Go.
4. Adaptador JavaScript/CDP e eventos controlados por limites de recursos.
5. MCP, memória SQLite, API e comandos auxiliares.
6. Conjunto de paridade Go↔Odin e benchmarks reproduzíveis com o mesmo hardware.

Referência: pasta browser/ (Go). Mantida a licença do repositório.
