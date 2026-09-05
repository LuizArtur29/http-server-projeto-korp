# Decisões Técnicas

Este documento registra decisões adotadas além dos requisitos mínimos do desafio.

## Aplicação

### `net/http` em vez de framework

Foi usada a biblioteca padrão do Go para reduzir dependências e complexidade.

### RFC3339 em UTC

```go
time.Now().UTC().Format(time.RFC3339)
```

Garante formato previsível e independente do timezone do host.

### Graceful shutdown

A aplicação trata sinais de encerramento e finaliza o servidor de forma controlada.

### Timeouts HTTP

Foram configurados timeouts de cabeçalho, leitura, escrita e conexão ociosa.

### Logs estruturados

Foi utilizado `slog` com saída JSON.

### `/healthz`

Endpoint adicional para diagnóstico e validação operacional.

## Observabilidade

### Métricas adicionais

```text
http_request_duration_seconds
http_requests_in_flight
```

### Labels controladas

```text
method
route
status
```

### Captura do status HTTP real

Um wrapper de `http.ResponseWriter` registra corretamente respostas como `200` e `405`.

### Scrape direto pelo Prometheus

Prometheus acessa `http-server-projeto-korp:8080/metrics` diretamente pela rede interna.

### UID fixo do datasource

```text
uid: prometheus
```

### Provisioning automático do Grafana

Datasource, provider e dashboard JSON são versionados.

### Portas administrativas em loopback

Prometheus e Grafana são expostos apenas em `127.0.0.1`.

## Containerização

### Multi-stage build

Separa compilação e runtime.

### Distroless e non-root

Runtime:

```text
gcr.io/distroless/static-debian12:nonroot
```

A aplicação executa como `nonroot:nonroot`.

### `.dockerignore` allow-list

O contexto começa bloqueado e libera somente arquivos necessários.

### Imagens pinadas

Foram evitadas tags como `latest`.

### Volumes read-only

Configurações são montadas com `:ro` quando possível.

## Rede

### Rede criada pelo Ansible

A `korp-network` é explicitamente criada pela automação.

### Rede externa no Compose

Evita uma rede duplicada com prefixo do projeto.

## Ansible

### Roles separadas

```text
docker
deploy
validate
```

### Suporte a Debian e Arch/CachyOS

A seleção é feita automaticamente por facts.

### Falha explícita em SO não suportado

Evita continuar em ambiente não validado.

### `community.docker`

Usado para rede, imagem e Compose, evitando shell desnecessário.

### `ansible.posix.synchronize`

Usa `rsync` para sincronização eficiente e idempotente.

### Rebuild condicionado a mudanças

```yaml
force_source: "{{ project_sync.changed }}"
```

### Check mode

O playbook suporta `--check`.

### Validação de configuração antes da validação funcional

Foram adicionadas verificações explícitas:

```text
nginx -t
promtool check config
```

Isso detecta erros de sintaxe/configuração antes dos testes HTTP.

As tasks usam `changed_when: false`, preservando a idempotência.

### Healthchecks no Docker Compose

NGINX, Prometheus e Grafana possuem healthchecks.

O NGINX verifica:

```text
http://127.0.0.1/healthz
```

Essa verificação valida o processo do NGINX, a configuração ativa, a resolução do upstream, a comunicação com a aplicação e a resposta do endpoint de saúde.

Foi utilizado `127.0.0.1` em vez de `localhost` para evitar ambiguidades de resolução IPv4/IPv6 dentro do container.

A aplicação Go não possui healthcheck interno porque a imagem distroless não contém cliente HTTP. Em vez de adicionar dependência apenas para healthcheck, sua saúde é validada externamente pelo NGINX e pelo Ansible.

### Idempotência

Uma segunda execução sem alterações foi validada com:

```text
changed=0
failed=0
```
