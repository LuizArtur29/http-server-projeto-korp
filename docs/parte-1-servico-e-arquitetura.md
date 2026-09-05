# Parte 1 — Serviço e Arquitetura do Ambiente

## Serviço HTTP

O serviço `http-server-projeto-korp` foi implementado em Go e escuta na porta `8080`.

Endpoint obrigatório:

```text
GET /projeto-korp
```

Resposta:

```json
{
  "nome": "Projeto Korp",
  "horario": "<horário_atual>"
}
```

O horário é resolvido dinamicamente em UTC a cada requisição.

Endpoints adicionais:

```text
GET /healthz
GET /metrics
```

## Dockerfile

A imagem usa build multi-stage.

Builder:

```text
golang:1.27.0-alpine3.24
```

Runtime:

```text
gcr.io/distroless/static-debian12:nonroot
```

O binário é compilado com:

```text
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
-trimpath
-ldflags="-s -w"
```

A aplicação executa como `nonroot:nonroot`.

## Docker Compose

A aplicação não publica `8080` no host:

```yaml
http-server-projeto-korp:
  image: http-server-projeto-korp:local
  expose:
    - "8080"
  networks:
    - korp-network
```

O NGINX publica:

```yaml
ports:
  - "80:80"
```

e monta:

```text
./nginx:/etc/nginx/conf.d:ro
```

## NGINX

Arquivo:

```text
nginx/http-server-projeto-korp.conf
```

Configuração:

```nginx
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://http-server-projeto-korp:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Rede Docker

A rede utilizada é:

```text
korp-network
```

Tipo:

```text
bridge
```

No estado final, a rede é criada pelo Ansible e consumida pelo Compose como externa:

```yaml
networks:
  korp-network:
    name: korp-network
    external: true
```

## Validação

```bash
curl http://localhost:80/projeto-korp
```

## Testes

```bash
go test ./...
```

Os testes cobrem o endpoint principal, UTC/RFC3339, métodos inválidos e `/healthz`.
