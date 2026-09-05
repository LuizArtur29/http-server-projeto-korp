# http-server-projeto-korp

Projeto desenvolvido para um desafio técnico DevOps, cobrindo serviço HTTP em Go, Docker, rede bridge, NGINX, Prometheus, Grafana e automação com Ansible.

## Arquitetura

```text
Host :80
   |
   v
 NGINX
   |
   v
http-server-projeto-korp:8080
   |
   +--> /projeto-korp
   +--> /healthz
   +--> /metrics
             |
             v
         Prometheus
             |
             v
           Grafana
```

Todos os serviços se comunicam pela rede Docker `korp-network`.

## Status

- [x] Serviço HTTP em Go
- [x] Dockerfile e container
- [x] Rede Docker bridge
- [x] Docker Compose
- [x] NGINX reverse proxy
- [x] Métricas Prometheus
- [x] Prometheus
- [x] Grafana
- [x] Dashboard provisionado automaticamente
- [x] Automação completa com Ansible
- [x] Validação automatizada
- [x] Healthchecks no Docker Compose
- [x] Validação de configuração com `nginx -t` e `promtool`
- [x] Idempotência validada com `changed=0`

## Execução rápida

```bash
curl http://localhost/projeto-korp
```

```bash
cd ansible
ansible-galaxy collection install -r requirements.yml
ansible-playbook site.yml -K
```

Prometheus: `http://localhost:9090`

Grafana: `http://localhost:3000`

## Documentação

- [Parte 1 — Serviço e Arquitetura](docs/parte-1-servico-e-arquitetura.md)
- [Parte 2 — Monitoramento e Observabilidade](docs/parte-2-monitoramento-observabilidade.md)
- [Parte 3 — Automação com Ansible](docs/parte-3-automacao-ansible.md)
- [Decisões Técnicas](docs/decisoes-tecnicas.md)

## Testes principais

```bash
go test ./...
docker build -t http-server-projeto-korp:local .
```

```bash
cd ansible
ansible-playbook --syntax-check site.yml
ansible-playbook site.yml -K
ansible-playbook site.yml -K
```

Na segunda execução:

```text
changed=0
failed=0
```
