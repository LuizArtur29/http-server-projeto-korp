# Parte 2 — Monitoramento e Observabilidade

## Métricas

A aplicação expõe:

```text
http_requests_total
http_request_duration_seconds
http_requests_in_flight
```

## Disponibilidade

```promql
up{job="http-server-projeto-korp"}
```

## Volume de requisições

```text
http_requests_total
```

Labels:

```text
method
route
status
```

Exemplo:

```text
http_requests_total{method="POST",route="/projeto-korp",status="405"}
```

## Taxa de requisições

```promql
sum(rate(http_requests_total{route="/projeto-korp"}[1m]))
```

## Latência p95

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(http_request_duration_seconds_bucket{route="/projeto-korp"}[5m])
  )
)
```

## Prometheus

O Prometheus coleta diretamente da aplicação:

```yaml
scrape_configs:
  - job_name: "http-server-projeto-korp"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "http-server-projeto-korp:8080"
```

## Grafana

Datasource provisionado:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    uid: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

Dashboard versionado em:

```text
monitoring/grafana/dashboards/http-server-projeto-korp-dashboard.json
```

Painéis:

| Painel | Finalidade |
|---|---|
| Service Availability | Disponibilidade |
| Total Requests | Volume total |
| Request Rate | Requisições por segundo |
| Latency p95 | Percentil 95 |
| Requests by HTTP Status | Distribuição por status |
| Requests In Flight | Concorrência |

## Validação do status HTTP

```bash
for i in $(seq 1 20); do
  curl -s -X POST http://localhost/projeto-korp > /dev/null
done
```

```bash
curl -s http://localhost/metrics   | grep 'http_requests_total.*projeto-korp'
```

Exemplo esperado:

```text
http_requests_total{method="POST",route="/projeto-korp",status="405"} 20
```

## Acesso

Prometheus:

```text
http://localhost:9090
```

Grafana:

```text
http://localhost:3000
```

Ambos são publicados apenas em `127.0.0.1`.
