# Port incremental para Odin

Estado: **primeiro módulo portado, sem paridade do navegador completo**.

A pasta odin/ mantém uma implementação independente em Odin do trecho
browser/cso.go, com:
- markdown_to_cso: compactação semântica para agentes (CSO)
- estimate_tokens: estimativa heurística de tokens
- CLI para arquivos locais
- testes unitários próprios

## Compilar e experimentar

Instale o compilador Odin: https://odin-lang.org/docs/install/

    odin test odin
    odin build odin -out:ai-odin-browser
    ./ai-odin-browser cso README.md
    ./ai-odin-browser tokens README.md

## Estratégia de migração

1. Validar este módulo contra uma seleção de fixtures do Go,
   com comparação byte a byte da saída.
2. Portar as estruturas Page/Link/Form/Field e o tokenizer HTML5
   (não usar parser de tags simplista como equivalente a HTML5).
3. Implementar transporte HTTP/TLS com cookies, redirecionamentos,
   compressão, limites de tamanho, timeout e bloqueio de SSRF.
4. Portar comandos, ações, formulários e persistência de sessões.
5. Implementar um adaptador de JavaScript e CDP/WebSocket, se necessário,
   sem alegar que isso seja funcionalidade nativa de Odin.
6. Reexecutar todos os cenários de benchmark e auditoria com ambos
   os executáveis no mesmo hardware, limites e amostras.

## Limitações deliberadas desta etapa

O port Odin **não navega na web**, não executa JavaScript, não abre
conexões TLS, não reproduz CDP, não salva cookies, nem integra SQLite
ou MCP. O Go é a implementação de referência enquanto a migração avança.

O módulo CSO ainda não promete equivalência perfeita em todos os
casos-limite: o regex Go para markdown e o uso de url.Parse foram
substituídos por analisadores locais menores. Em especial, links
Markdown aninhados, URLs com parâmetros complexos e caracteres
Unicode na borda do max_chars exigem fixtures adicionais.

Não existe medição de economia de memória, velocidade ou superioridade
de Odin nesta alteração; os números da documentação Go não se transferem
automaticamente ao port.
