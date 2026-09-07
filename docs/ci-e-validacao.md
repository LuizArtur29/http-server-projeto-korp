# CI e Validação

## Objetivo

Automatizar as validações do projeto em pushes para `main` e pull requests.

A CI foi mantida pequena e alinhada às ferramentas já utilizadas no projeto.

## Pipeline

```text
checkout
   ↓
setup Go
   ↓
go mod download
   ↓
go test ./...
   ↓
docker compose config
   ↓
criação da korp-network
   ↓
docker build
   ↓
docker compose up
   ↓
nginx -t
   ↓
promtool check config
   ↓
scripts/smoke-test.sh
   ↓
logs em caso de falha
   ↓
docker compose down
```

## Variáveis do Grafana na CI

A CI fornece credenciais descartáveis apenas para o ambiente efêmero do runner:

```yaml
env:
  GF_SECURITY_ADMIN_USER: admin
  GF_SECURITY_ADMIN_PASSWORD: ci-only-password
```

Essa senha não representa um segredo de ambiente real e existe apenas durante a execução do job.

## Smoke test

O workflow reutiliza:

```text
scripts/smoke-test.sh
```

O script valida:

- aplicação;
- endpoint `/metrics`;
- readiness do Prometheus;
- health do Grafana.

O script possui retry para absorver o tempo normal de inicialização dos serviços.

## Validação do NGINX

```bash
docker compose exec -T nginx nginx -t
```

## Validação do Prometheus

```bash
docker compose exec -T prometheus   promtool check config /etc/prometheus/prometheus.yml
```

## Diagnóstico em falha

Quando qualquer etapa falha:

```bash
docker compose logs
```

## Cleanup

O encerramento da stack usa `if: always()`, garantindo `docker compose down` mesmo em caso de falha.

## Execução local equivalente

```bash
make ci-local
```
