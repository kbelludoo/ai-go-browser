# Bateria de integração — GoBrowser 0.5

## O que foi de fato executado

As validações em `e2e_backends.py` acessam **servidores HTTP locais reais**:
- backend **Python** com `http.server` e cookie de sessão;
- backend **PHP** no servidor nativo `php -S`, executando arquivos `.php` de verdade.

Nenhum endpoint da internet pública é contatado. Não são apenas rotas PHP ou Python simuladas por Go. O navegador testado é o binário Go `bin/ai-go-browser-linux-amd64`.

A suíte cobre sete verificações em cada backend:
1. `discover` não consulta a API nem executa POST.
2. Botão associado a `fetch` expõe a ação GET, que exige confirmação e recupera JSON via HTTP com cookies de sessão; o campo sensível `api_token` é ocultado.
3. `link rel=alternate` anuncia JSON diretamente, sem renderizar.
4. Formulário GET aplica o valor preenchido e obtém os resultados.
5. `hx-get` devolve o fragmento de HTML como texto estruturado.
6. Formulário POST é bloqueado por padrão.
7. Domínio não autorizado é bloqueado.

## Resultado desta execução

- **140/140 verificações passaram** (10 repetições × 2 backends × 7 cenários).
- **12,46 MiB** foi o maior RSS amostrado do **processo Go**, após os comandos, neste teste local. Não mede o processo PHP/Python nem representa o pico contínuo da árvore de processos; não prova 300 MiB em sites reais.
- Testes unitários Go: `go test ./... -count=1` passou; `go vet ./...` passou.
- Runner pré-existente `webtest` em modo local: 3/3 aberturas, HTTP 200, texto extraído.
- Há evidência em `results.json`, `repeated10.txt` e `local-webtest-results.json`.

## Como repetir localmente no Linux AMD64

```bash
cd ai-go-browser-action-resolver
python3 integration/e2e_backends.py --repeat 10
# PHP deve estar instalado; caso contrário o teste PHP é automaticamente pulado.
# Para testar só Python: --no-php

go test ./... -count=1
go vet ./...
```

## Na Oracle Linux ARM64

Sem compilar nada (o núcleo Go já está pré-compilado):

```bash
chmod +x bin/ai-go-browser-linux-arm64 bin/webtest-linux-arm64
python3 integration/e2e_backends.py \
  --browser bin/ai-go-browser-linux-arm64 \
  --repeat 10 --out integration/oracle-backends.json
```

Se PHP não estiver instalado, o runner omite esse conjunto e deixa claro no JSON. Para teste completo, instale o PHP apropriado à distribuição.

## Teste em sites públicos (somente numa máquina com acesso à internet)

```bash
./bin/webtest-linux-arm64 \
  -sites webtest/sites.txt -repeat 3 \
  -out webtest/sites-publicos.json
```

O runner `webtest` usa somente GET e não faz login, envia formulários ou contorna antibot. As páginas selecionadas são Example.com, Hacker News, GitHub, Wikipedia e MDN. Alterações na rede/robots/servidor podem alterar o resultado. Não afirmar que funcionou antes de observar o arquivo de saída.

## Próximos testes que ainda faltam

- Navegação com autenticação real e autorização expressa.
- Requisições GraphQL, WebSocket, APIs descobertas via bundles JS e URLs montadas dinamicamente.
- Compatibilidade de conteúdo comparada com Chromium **em páginas idênticas**.
- Carga/latência sob várias páginas, DNS/redirects/rate limits e pico **sincronizado da árvore de processos** com limite efetivo de memória (cgroups).
- Avaliação de segurança incluindo SSRF via DNS/rebinding, exfiltração e permissões por operação.

`discover` apenas identifica possíveis ações; a presença de uma URL no JavaScript não assegura que o botão de fato usaria aquela API com os mesmos argumentos e cabeçalhos.
