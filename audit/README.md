# Auditoria reproduzível

Scripts usados em [`../AUDITORIA.md`](../AUDITORIA.md) para testar as afirmações do README.
Todos rodam localmente, sem rede externa e sem Chromium.

| Arquivo | Mede |
| :--- | :--- |
| `payload/main.go` | bytes de HTML vs. snapshot completo vs. `-compact` vs. `view:"read"`, refs, latência |
| `headers_server.py` | cabeçalhos HTTP realmente enviados pelo motor Go |
| `tls_hello.py` | ClientHello real + JA3/MD5 (sem certificado: a conexão é cortada após o hello) |
| `h2_probe.py` + `h2decode/main.go` | preface HTTP/2, SETTINGS e ordem dos pseudo-headers |
| `agents.py` | RSS real de N agentes concorrentes |

## Como rodar

```bash
go build -o bin/aigobrowser ./cmd/aigobrowser

# 1. payload / tokens
go run ./audit/payload

# 2. cabeçalhos HTTP
python3 audit/headers_server.py 8099 &
./bin/aigobrowser -block-private=false -engine=http -format=json http://127.0.0.1:8099/
./bin/aigobrowser -block-private=false -format=json http://127.0.0.1:8099/ | head -c 400   # engine=auto

# 3. TLS
python3 audit/tls_hello.py 8443 &
./bin/aigobrowser -block-private=false -engine=http -format=json https://localhost:8443/

# 4. HTTP/2
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 2 -nodes \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
python3 audit/h2_probe.py 8444 cert.pem key.pem &
SSL_CERT_FILE=$PWD/cert.pem ./bin/aigobrowser -block-private=false -engine=http \
  -format=json https://localhost:8444/
go run ./audit/h2decode /tmp/lab/h2.json

# 5. agentes simultâneos
python3 audit/agents.py ./bin/aigobrowser http://127.0.0.1:8099/ 20
```

`payload` e `h2decode` são `package main` em subdiretórios próprios, então `go build ./...`
continua funcionando. Nada aqui altera o código do navegador.
