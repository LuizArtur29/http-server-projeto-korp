# Parte 3 — Automação com Ansible

## Objetivo

Provisionar as Partes 1 e 2 com um único comando.

## Estrutura

```text
ansible/
├── ansible.cfg
├── inventory/
├── group_vars/
├── requirements.yml
├── roles/
│   ├── docker/
│   ├── deploy/
│   └── validate/
└── site.yml
```

## Distribuições suportadas

```text
Debian
Arch Linux / CachyOS
```

A seleção é feita com `ansible_facts["os_family"]`.

## Dependências

```yaml
collections:
  - name: community.docker
  - name: community.general
  - name: ansible.posix
```

## Role Docker

Responsável por instalar Docker e `rsync`, habilitar/iniciar o serviço e selecionar APT ou pacman conforme o sistema.

## Role Deploy

Fluxo:

```text
criar diretório
   ↓
sincronizar arquivos
   ↓
criar korp-network
   ↓
buildar imagem
   ↓
subir Docker Compose
```

A sincronização registra mudanças:

```yaml
- name: Synchronize project files
  ansible.posix.synchronize:
    src: "{{ playbook_dir }}/../"
    dest: "{{ project_dir }}/"
    archive: true
    delete: true
    rsync_opts:
      - "--exclude=.git"
      - "--exclude=.idea"
      - "--exclude=ansible"
  register: project_sync
```

O rebuild ocorre somente quando necessário:

```yaml
force_source: "{{ project_sync.changed }}"
```

## Validação de configuração

Antes das validações funcionais, o playbook verifica as configurações do NGINX e do Prometheus.

### NGINX

```yaml
- name: Validate NGINX configuration
  ansible.builtin.command:
    cmd: docker compose exec -T nginx nginx -t
    chdir: "{{ project_dir }}"
  register: nginx_config_check
  changed_when: false
  when: not ansible_check_mode
```

Resultado esperado:

```text
syntax is ok
test is successful
```

### Prometheus

```yaml
- name: Validate Prometheus configuration
  ansible.builtin.command:
    cmd: >-
      docker compose exec -T prometheus
      promtool check config /etc/prometheus/prometheus.yml
    chdir: "{{ project_dir }}"
  register: prometheus_config_check
  changed_when: false
  when: not ansible_check_mode
```

Resultado esperado:

```text
SUCCESS: /etc/prometheus/prometheus.yml is valid prometheus config file syntax
```

## Validação funcional

O playbook verifica:

```text
Aplicação
Prometheus
Grafana
```

Endpoints:

```text
http://localhost/projeto-korp
http://localhost:9090/-/ready
http://localhost:3000/api/health
```

Resumo esperado:

```text
NGINX configuration: OK
Prometheus configuration: OK
Application: OK
Prometheus readiness: OK
Grafana: ok
```

## Healthchecks no Docker Compose

Foram adicionados healthchecks para NGINX, Prometheus e Grafana.

O NGINX valida o endpoint `/healthz` através do próprio proxy:

```yaml
healthcheck:
  test:
    - CMD-SHELL
    - "wget -q -O /dev/null http://127.0.0.1/healthz || exit 1"
  interval: 10s
  timeout: 3s
  retries: 5
  start_period: 10s
```

O uso de `127.0.0.1` evita ambiguidades de resolução entre IPv4 e IPv6 dentro do container.

Prometheus e Grafana utilizam seus endpoints de readiness/health.

A aplicação Go não possui healthcheck interno no Compose porque sua imagem final é distroless e não inclui `curl` ou `wget`. Sua saúde é validada externamente pelo NGINX e pelo Ansible.

## Execução

```bash
cd ansible
ansible-galaxy collection install -r requirements.yml
ansible-playbook site.yml -K
```

## Check mode

```bash
ansible-playbook site.yml --check -K
```

## Idempotência

Uma segunda execução sem mudanças foi validada com:

```text
localhost : ok=19 changed=0 unreachable=0 failed=0 skipped=3
```
