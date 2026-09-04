# http-server-projeto-korp

Projeto desenvolvido como parte de um desafio técnico DevOps, com foco em construção de serviço HTTP em Go, containerização, redes Docker, proxy reverso com NGINX, observabilidade e automação com Ansible.

## Status

- [x] Serviço HTTP em Go
- [x] Containerização da aplicação
- [x] Docker Compose
- [x] Rede Docker bridge
- [x] NGINX como proxy reverso
- [ ] Prometheus
- [ ] Grafana
- [ ] Dashboard de observabilidade
- [ ] Automação completa com Ansible
- [ ] CI

---

## Parte 1 — Serviço e Arquitetura do Ambiente

### Arquitetura atual

```text
                         Host
                          │
                          │ :80
                          ▼
                  ┌───────────────┐
                  │     NGINX     │
                  │ Reverse Proxy │
                  └───────┬───────┘
                          │
                          │ :8080
                          ▼
              ┌──────────────────────────┐
              │ http-server-projeto-korp │
              │           Go             │
              └──────────────────────────┘

                   Docker bridge network
```

O NGINX é o único ponto de entrada HTTP exposto ao host.

A aplicação Go permanece acessível apenas pela rede interna do Docker.

---

## Serviço HTTP

O serviço foi implementado em Go e escuta na porta `8080`.

### Endpoints

#### `GET /projeto-korp`

Endpoint principal solicitado pelo desafio.

Exemplo de resposta:

```json
{
  "nome": "Projeto Korp",
  "horario": "2026-09-04T00:30:00Z"
}
```

O valor de `horario` é calculado dinamicamente a cada requisição utilizando UTC no formato RFC3339.

#### `GET /healthz`

Endpoint adicional utilizado para verificação da saúde da aplicação.

Exemplo:

```json
{
  "status": "healthy"
}
```

#### `GET /metrics`

Endpoint utilizado para exposição de métricas no formato Prometheus.

Será utilizado posteriormente pelo Prometheus na etapa de observabilidade.

---

## Decisões da aplicação

### Biblioteca HTTP padrão do Go

Foi utilizada a biblioteca `net/http` em vez de um framework web externo.

A aplicação possui poucos endpoints e não necessita de recursos adicionais fornecidos por frameworks.

Essa escolha reduz:

- número de dependências;
- superfície de ataque;
- tamanho da aplicação;
- complexidade operacional.

### Horário UTC

O horário retornado pelo endpoint `/projeto-korp` utiliza:

```go
time.Now().UTC().Format(time.RFC3339)
```

A utilização de UTC evita dependência do timezone da máquina ou do container e mantém o formato consistente entre diferentes ambientes.

### Graceful shutdown

A aplicação trata sinais de encerramento, permitindo que o servidor finalize requisições em andamento antes de encerrar o processo.

Isso melhora o comportamento do serviço durante:

- reinicializações;
- atualizações;
- encerramento de containers.

### HTTP timeouts

O servidor possui limites configurados para leitura e escrita das conexões.

O objetivo é evitar conexões abertas indefinidamente e reduzir o risco de consumo desnecessário de recursos.

### Logs estruturados

A aplicação utiliza logs estruturados em JSON.

Isso facilita processamento e consulta dos logs por ferramentas externas de observabilidade.

---

## Métricas da aplicação

A aplicação já expõe métricas Prometheus através de `/metrics`.

As métricas adicionadas incluem:

```text
http_requests_total
http_request_duration_seconds
http_requests_in_flight
```

### `http_requests_total`

Contabiliza o volume total de requisições HTTP.

Labels:

```text
method
route
status
```

Exemplo:

```text
http_requests_total{
  method="GET",
  route="/projeto-korp",
  status="200"
}
```

### `http_request_duration_seconds`

Histograma utilizado para medir a duração das requisições HTTP.

Permitirá analisar posteriormente métricas como latência média e percentis no Grafana.

### `http_requests_in_flight`

Representa a quantidade de requisições sendo processadas simultaneamente.

---

# Containerização

A aplicação utiliza Docker com estratégia de build multi-stage.

## Build stage

O estágio de compilação utiliza:

```text
golang:1.27.0-alpine3.24
```

A versão da linguagem e da distribuição Linux são explicitamente definidas para melhorar a previsibilidade dos builds.

O binário é compilado utilizando:

```text
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
```

Também são utilizadas as opções:

```text
-trimpath
-ldflags="-s -w"
```

### `CGO_ENABLED=0`

Gera um binário Go sem dependências dinâmicas do sistema operacional.

Isso permite executar a aplicação em uma imagem extremamente mínima.

### `-trimpath`

Remove caminhos locais do ambiente de compilação do binário.

### `-ldflags="-s -w"`

Remove informações de símbolos e debug não necessárias para execução, reduzindo o tamanho do binário final.

---

## Runtime Distroless

O container final utiliza:

```text
gcr.io/distroless/static-debian12:nonroot
```

A imagem final contém essencialmente apenas o necessário para executar a aplicação.

Ao contrário de uma distribuição Linux convencional, a imagem distroless não possui:

- shell;
- package manager;
- compilador;
- utilitários administrativos desnecessários.

### Benefícios

- menor superfície de ataque;
- menor tamanho da imagem;
- menor quantidade de componentes vulneráveis;
- execução mais próxima do princípio de mínimo privilégio.

---

## Execução como usuário não-root

O processo é executado utilizando:

```dockerfile
USER nonroot:nonroot
```

A aplicação não necessita de privilégios administrativos dentro do container.

Durante a validação:

```bash
docker image inspect http-server-projeto-korp:local \
  --format='{{.Config.User}}'
```

Resultado:

```text
nonroot:nonroot
```

---

## Tamanho da imagem

Após o build, a imagem apresentou aproximadamente:

```text
CONTENT SIZE: 5.2 MB
```

O tamanho reduzido é resultado principalmente da utilização de:

- compilação estática;
- multi-stage build;
- runtime distroless;
- remoção de símbolos desnecessários.

---

# Docker build context

O projeto utiliza uma estratégia de allow list no `.dockerignore`.

Todo o conteúdo é ignorado inicialmente:

```dockerignore
**
```

e apenas os arquivos necessários para compilação são liberados:

```dockerignore
!go.mod
!go.sum
!cmd/
!cmd/**
!internal/
!internal/**
```

Dessa forma, arquivos que não participam da compilação não são enviados para o contexto de build.

Além disso, o próprio Dockerfile copia explicitamente somente:

```text
go.mod
go.sum
cmd/
internal/
```

Essa abordagem reduz o contexto enviado ao Docker e evita inclusão acidental de arquivos desnecessários na imagem.

---

# Rede e Docker Compose

Os containers são executados utilizando Docker Compose.

Uma rede dedicada é criada utilizando o driver:

```text
bridge
```

Essa rede permite comunicação entre os containers utilizando o DNS interno do Docker.

---

## Serviço da aplicação

O container:

```text
http-server-projeto-korp
```

escuta internamente na porta:

```text
8080
```

Essa porta não é publicada diretamente no host.

Portanto, não existe:

```yaml
ports:
  - "8080:8080"
```

no serviço da aplicação.

Isso garante que o acesso externo passe obrigatoriamente pelo proxy reverso.

---

# NGINX

O NGINX funciona como único ponto de entrada HTTP da aplicação.

O container utiliza uma imagem oficial do projeto NGINX e está conectado à mesma rede Docker da aplicação.

A porta:

```text
80
```

do host é mapeada para a porta:

```text
80
```

do container.

Fluxo:

```text
localhost:80
     │
     ▼
   NGINX
     │
     ▼
http-server-projeto-korp:8080
```

---

## Proxy reverso

O arquivo:

```text
nginx/http-server-projeto-korp.conf
```

é montado em:

```text
/etc/nginx/conf.d/
```

dentro do container NGINX.

O encaminhamento utiliza o nome do serviço Docker:

```nginx
proxy_pass http://http-server-projeto-korp:8080;
```

Isso evita dependência de endereços IP internos, que podem mudar quando containers são recriados.

O DNS interno do Docker Compose resolve automaticamente:

```text
http-server-projeto-korp
```

para o container correspondente.

---

## Headers do proxy

O NGINX encaminha informações importantes da requisição original:

```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

Isso preserva informações úteis para logs, auditoria e possíveis integrações futuras.

---

# Princípio de exposição mínima

Uma das decisões da arquitetura foi limitar os serviços acessíveis diretamente pelo host.

Atualmente:

```text
NGINX       → exposto na porta 80
Aplicação   → somente rede interna Docker
```

A aplicação não publica `8080` no host.

Isso cria um único ponto de entrada e mantém a comunicação entre os serviços dentro da rede Docker.

---

# Testes da aplicação

Os handlers HTTP possuem testes automatizados utilizando os pacotes padrão:

```text
testing
net/http/httptest
```

São validados:

- status HTTP;
- estrutura da resposta;
- valor de `nome`;
- formato RFC3339;
- timezone UTC;
- endpoint de saúde;
- rejeição de métodos HTTP não permitidos.

Os testes podem ser executados com:

```bash
go test ./...
```

Validação adicional:

```bash
go vet ./...
```

---

# Execução

## Build da aplicação

```bash
docker build \
  -t http-server-projeto-korp:local .
```

## Subir o ambiente

```bash
docker compose up -d
```

## Verificar os containers

```bash
docker compose ps
```

## Testar o serviço através do NGINX

```bash
curl http://localhost:80/projeto-korp
```

Resposta esperada:

```json
{
  "nome": "Projeto Korp",
  "horario": "<horario-atual-em-UTC>"
}
```

---

# Validação da arquitetura

A porta interna da aplicação não deve estar disponível diretamente no host.

Portanto:

```bash
curl http://localhost:8080/projeto-korp
```

deve falhar.

Enquanto:

```bash
curl http://localhost:80/projeto-korp
```

deve funcionar.

Isso comprova que o fluxo externo ocorre exclusivamente através do NGINX.

---

# Decisões arquiteturais da Parte 1

| Decisão | Motivação |
|---|---|
| Go `net/http` | Menor número de dependências e simplicidade |
| UTC + RFC3339 | Formato consistente independente de timezone |
| Graceful shutdown | Encerramento controlado do serviço |
| HTTP timeouts | Proteção contra conexões excessivamente longas |
| Logs estruturados | Melhor integração futura com observabilidade |
| Multi-stage build | Separação entre build e runtime |
| Alpine no builder | Ambiente de compilação pequeno |
| Distroless no runtime | Redução da superfície de ataque |
| Usuário non-root | Princípio de menor privilégio |
| `.dockerignore` allow list | Contexto de build mínimo |
| Rede Docker bridge dedicada | Isolamento e comunicação interna |
| Aplicação sem porta publicada | Menor superfície de exposição |
| NGINX como único entrypoint | Controle centralizado de acesso |
| DNS interno do Docker | Evita dependência de IPs estáticos |
| Volume NGINX read-only | Configuração disponível sem permissão de escrita |
| Testes automatizados | Garantia do contrato HTTP |

---

# Próximas etapas

A próxima etapa adicionará observabilidade utilizando:

```text
Prometheus
Grafana
```

O Prometheus realizará o scrape do endpoint `/metrics` da aplicação.

O Grafana será configurado de forma automatizada através de provisioning para disponibilizar dashboards sem configuração manual.
