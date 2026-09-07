# http-server-projeto-korp

Projeto desenvolvido para um desafio técnico DevOps, cobrindo serviço HTTP em Go, Docker, rede bridge, NGINX, Prometheus, Grafana, automação com Ansible e CI

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
- [x] Hardening da aplicação com filesystem read-only e no-new-privileges
- [x] Limites moderados de CPU e memória
- [x] Makefile para padronização dos comandos
- [x] Labels e metadados nos containers
- [x] Credenciais do Grafana parametrizadas por variáveis de ambiente
- [x] CI com GitHub Actions
- [x] CachyOS / Arch validado
- [x] Debian validado em host remoto de homelab


## Execução rápida

Crie o arquivo local de ambiente:
```bash
cp .env.example .env
```
Ajuste a senha do Grafana no .env e execute:
```bash
make up
```
Valide:
```bash
make smoke
```
Provisionamento completo com Ansible:

```bash
cd ansible
ansible-galaxy collection install -r requirements.yml
ansible-playbook site.yml -K
```
## Interface

Aplicação: `http://localhost/projeto-korp`

Prometheus: `http://localhost:9090`

Grafana: `http://localhost:3000`

## Comandos úteis
```bash
make test
```
```bash
make build
```
```bash
make up
```
```bash
make down
```
```bash
make smoke
```
```bash
make compose-validate
```
```bash
make nginx-validate
```
```bash
make prometheus-validate
```
```bash
make ansible-check
```
```bash
make ansible-run
```
```bash
make ci-local
```

## Documentação

- [Parte 1 — Serviço e Arquitetura](docs/parte-1-servico-e-arquitetura.md)
- [Parte 2 — Monitoramento e Observabilidade](docs/parte-2-monitoramento-observabilidade.md)
- [Parte 3 — Automação com Ansible](docs/parte-3-automacao-ansible.md)
- [CI e Validação](docs/ci-e-validacao.md)
- [Variáveis de Ambiente](docs/variaveis-de-ambiente.md)
- [Decisões Técnicas](docs/decisoes-tecnicas.md)

