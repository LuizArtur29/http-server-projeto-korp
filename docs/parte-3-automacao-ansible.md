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

O playbook foi validado em dois cenários reais:

```text
Arch Linux / CachyOS
Debian em host remoto de homelab
```

## Dependências

```yaml
collections:
  - name: community.docker
  - name: community.general
  - name: ansible.posix
```

## Role Docker

Responsável por instalar Docker e `rsync`, habilitar/iniciar o serviço e selecionar APT ou pacman conforme o sistema.

No Debian, a instalação utiliza o repositório oficial do Docker.

No Arch Linux / CachyOS, os pacotes são instalados via pacman.

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

### Sincronização dos arquivos

Em ambiente local, a sincronização pode ocorrer diretamente para o diretório final.

Durante a validação em Debian remoto, foi identificado que `ansible.posix.synchronize` com `become` tenta executar o `rsync` remoto através de `sudo`.

Em hosts onde o `sudo` exige senha, esse fluxo pode falhar porque o `rsync` não possui terminal interativo para solicitar a senha.

Erro observado:

```text
sudo: a terminal is required to read the password
sudo: a password is required
rsync: connection unexpectedly closed
```

Para manter o playbook portável e evitar exigir `NOPASSWD` no host remoto, a sincronização foi ajustada para ocorrer sem `become` em um diretório temporário acessível pelo usuário remoto.

```yaml
- name: Create temporary project directory
  ansible.builtin.file:
    path: /tmp/http-server-projeto-korp
    state: directory
    mode: "0755"
  become: false

- name: Synchronize project files
  ansible.posix.synchronize:
    src: "{{ playbook_dir }}/../"
    dest: /tmp/http-server-projeto-korp/
    archive: true
    delete: true
    rsync_opts:
      - "--exclude=.git"
      - "--exclude=.idea"
      - "--exclude=ansible"
  register: project_sync
  become: false
```

Depois, os arquivos são copiados para o diretório definitivo usando uma task Ansible normal, onde o `become` funciona através do próprio mecanismo do Ansible:

```yaml
- name: Copy synchronized project to deployment directory
  ansible.builtin.copy:
    src: /tmp/http-server-projeto-korp/
    dest: "{{ project_dir }}/"
    remote_src: true
    owner: root
    group: root
    mode: preserve
  when: project_sync.changed
```

Essa abordagem evita alterar a política de sudo do host apenas para viabilizar o `rsync`.

### Rebuild condicionado a mudanças

A sincronização registra mudanças através de:

```yaml
register: project_sync
```

O rebuild ocorre somente quando necessário:

```yaml
force_source: "{{ project_sync.changed }}"
```

Assim:

```text
arquivos alterados -> rebuild
sem alterações     -> nenhum rebuild
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
http://127.0.0.1:3000/api/health
```

Resumo esperado:

```text
NGINX configuration: OK
Prometheus configuration: OK
Application: OK
Prometheus readiness: OK
Grafana: ok
```

## Validação do Grafana

Durante o teste em Debian remoto, foi observado que o endpoint publicado do Grafana poderia resetar conexões enquanto o serviço ainda finalizava seu startup.

Como o Docker Compose já possui healthcheck para o Grafana, o playbook passou a aguardar primeiro o estado `healthy` do container antes de executar a validação HTTP.

Fluxo:

```text
obter container ID
   ↓
aguardar State.Health.Status == healthy
   ↓
validar /api/health
```

Obtenção do container:

```yaml
- name: Get Grafana container ID
  ansible.builtin.command:
    cmd: docker compose ps -q grafana
    chdir: "{{ project_dir }}"
  register: grafana_container_id
  changed_when: false
  when: not ansible_check_mode
```

Espera pelo healthcheck:

```yaml
- name: Wait for Grafana container to become healthy
  ansible.builtin.command:
    cmd: >-
      docker inspect
      --format='{{ "{{" }}.State.Health.Status{{ "}}" }}'
      {{ grafana_container_id.stdout }}
  register: grafana_health
  changed_when: false
  retries: 20
  delay: 3
  until: grafana_health.stdout == "healthy"
  when: not ansible_check_mode
```

Validação HTTP:

```yaml
- name: Validate Grafana health endpoint
  ansible.builtin.uri:
    url: http://127.0.0.1:3000/api/health
    method: GET
    status_code: 200
    return_content: true
  register: grafana_response
  retries: 5
  delay: 3
  until: grafana_response.status == 200
  when: not ansible_check_mode
```

O uso de `127.0.0.1` evita ambiguidades de resolução entre IPv4 e IPv6.

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

## Validação em Debian remoto

O provisionamento foi validado em um host Debian de homelab.

Primeira execução após os ajustes:

```text
debian-homelab : ok=26 changed=1 unreachable=0 failed=0 skipped=4
```

Segunda execução, sem alterações:

```text
debian-homelab : ok=26 changed=0 unreachable=0 failed=0 skipped=4
```

Isso confirmou a idempotência também em ambiente Debian remoto.

## Idempotência

A idempotência foi validada em dois ambientes:

```text
Arch Linux / CachyOS -> changed=0
Debian remoto        -> changed=0
```

O playbook mantém:

```text
failed=0
```

em execuções subsequentes sem alterações.
