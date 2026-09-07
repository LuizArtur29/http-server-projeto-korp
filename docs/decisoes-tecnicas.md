# Decisões Técnicas

Este documento registra decisões adotadas além dos requisitos mínimos do desafio.

## Aplicação

### `net/http` em vez de framework

Foi utilizada a biblioteca padrão do Go para reduzir dependências e complexidade.

### RFC3339 em UTC

```go
time.Now().UTC().Format(time.RFC3339)
```

### Graceful shutdown

A aplicação trata sinais de encerramento de forma controlada.

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

O Prometheus coleta métricas diretamente em:

```text
http-server-projeto-korp:8080/metrics
```

### Provisioning automático do Grafana

Datasource, provider e dashboard JSON são versionados.

### Portas administrativas em loopback

Prometheus e Grafana são publicados apenas em `127.0.0.1`.

## Containerização e segurança

### Multi-stage build

Separa compilação e runtime.

### Distroless e non-root

A aplicação utiliza imagem runtime distroless e executa como usuário não privilegiado.

### Filesystem read-only

A aplicação utiliza:

```yaml
read_only: true
```

Como o binário não precisa gravar no filesystem, isso reduz a superfície de modificação dentro do container.

### `no-new-privileges`

A aplicação utiliza:

```yaml
security_opt:
  - no-new-privileges:true
```

Isso impede elevação de privilégios através de mecanismos como binários setuid.

### Limites de recursos

Foram definidos limites moderados de CPU e memória para aplicação, NGINX, Prometheus e Grafana.

A intenção não é representar capacity planning definitivo, mas evitar consumo ilimitado no host.

### `.dockerignore` allow-list

O contexto de build começa bloqueado e libera apenas os arquivos necessários.

### Imagens pinadas

Foram evitadas tags flutuantes como `latest`.

### Volumes read-only

Configurações são montadas como somente leitura quando possível.

## Healthchecks

NGINX, Prometheus e Grafana possuem healthchecks no Docker Compose.

O NGINX valida:

```text
http://127.0.0.1/healthz
```

Essa verificação cobre processo do NGINX, configuração ativa, resolução do upstream, comunicação com a aplicação e resposta do endpoint de saúde.

Foi utilizado `127.0.0.1` em vez de `localhost` para evitar ambiguidades de resolução IPv4/IPv6.

A aplicação não possui healthcheck interno porque a imagem distroless não inclui cliente HTTP. A saúde é validada externamente pelo NGINX e pelo Ansible.

## Rede

### Rede criada pelo Ansible

A `korp-network` é criada explicitamente pela automação.

### Rede externa no Compose

Evita criação de uma segunda rede com prefixo do projeto.

## Ansible

### Roles separadas

```text
docker
deploy
validate
```

### Suporte a Debian e Arch/CachyOS

A seleção é feita por Ansible facts.

Além do ambiente local Arch/CachyOS, o playbook foi validado em um host Debian remoto de homelab.

Essa validação em um segundo ambiente foi importante para identificar diferenças reais de execução que não apareceram no teste local.

### Instalação específica por distribuição

No Debian, o Docker é instalado a partir do repositório oficial do Docker utilizando APT.

No Arch Linux / CachyOS, os pacotes são instalados com pacman.

O `rsync` também é instalado explicitamente nas duas famílias, evitando depender de uma ferramenta previamente existente no host.

### `community.docker`

Usado para rede, imagem e Compose.

### `ansible.posix.synchronize`

Usa `rsync` para sincronização eficiente.

### Sincronização remota sem exigir `NOPASSWD`

Durante o teste em Debian remoto, `ansible.posix.synchronize` com `become` tentou executar o `rsync` remoto através de `sudo`.

Em hosts onde `sudo` exige senha, esse fluxo falha porque o processo de `rsync` não dispõe de terminal interativo para solicitar a senha.

Erro observado:

```text
sudo: a terminal is required to read the password
sudo: a password is required
rsync: connection unexpectedly closed
```

Em vez de alterar a política de sudo do host ou exigir `NOPASSWD`, foi adotado um fluxo em duas etapas:

```text
rsync sem become para /tmp
        ↓
copy remote_src com become
        ↓
/opt/http-server-projeto-korp
```

A sincronização ocorre em um diretório temporário acessível pelo usuário remoto:

```yaml
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

Depois, os arquivos são copiados para o diretório final com uma task Ansible normal, permitindo que a elevação de privilégio seja tratada pelo próprio mecanismo de `become`:

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

Essa escolha melhora a portabilidade do playbook e evita modificar a política de segurança do host apenas para atender ao mecanismo de sincronização.

### Rebuild condicionado a mudanças

```yaml
force_source: "{{ project_sync.changed }}"
```

A imagem é rebuildada apenas quando a sincronização detecta mudanças no projeto.

### Validação de configuração

Foram adicionados:

```text
nginx -t
promtool check config
```

As tasks usam `changed_when: false` para preservar a idempotência.

### Validação do Grafana baseada no healthcheck

Durante o teste em Debian remoto, o endpoint HTTP publicado do Grafana ainda podia resetar conexões enquanto o serviço concluía sua inicialização.

Em vez de aumentar arbitrariamente tempos de espera, o Ansible passou a aguardar o estado de saúde informado pelo Docker:

```text
.State.Health.Status == healthy
```

Somente depois disso é realizada a validação HTTP em:

```text
http://127.0.0.1:3000/api/health
```

O fluxo fica:

```text
obter container ID
      ↓
aguardar container healthy
      ↓
validar endpoint /api/health
```

Essa abordagem utiliza o próprio healthcheck definido no Compose como sinal operacional de prontidão.

### Uso de `127.0.0.1` em validações locais

Foi preferido `127.0.0.1` a `localhost` em pontos sensíveis para evitar diferenças de resolução entre IPv4 e IPv6.

Esse comportamento foi observado tanto no healthcheck do NGINX quanto nas validações do Grafana.

### Check mode

O playbook suporta `--check`.

### Idempotência

A idempotência foi validada em dois ambientes diferentes:

```text
Arch Linux / CachyOS -> changed=0
Debian remoto        -> changed=0
```

No Debian remoto, a execução final apresentou:

```text
debian-homelab : ok=26 changed=0 unreachable=0 failed=0 skipped=4
```

Isso confirmou que as adaptações para execução remota não comprometeram a idempotência.

## Automação local

### Smoke test reutilizável

O script:

```text
scripts/smoke-test.sh
```

valida aplicação, endpoint de métricas, Prometheus e Grafana.

O mesmo script é reutilizado localmente e na CI, evitando duplicação da lógica de validação.

### Makefile

Foi adicionado um `Makefile` mínimo para padronizar comandos recorrentes sem esconder as ferramentas utilizadas.

Exemplos:

```text
make test
make build
make up
make smoke
make ci-local
```

## Metadados dos containers

Foram adicionados labels para facilitar identificação operacional dos serviços e do projeto.

Exemplos:

```text
com.korp.project
com.korp.service
com.korp.managed-by
```

Também foram utilizados labels OCI quando aplicáveis.

## Configuração do Grafana

As credenciais administrativas do Grafana são parametrizadas por variáveis de ambiente.

O repositório contém apenas:

```text
.env.example
```

O arquivo real:

```text
.env
```

não é versionado.

A senha é exigida pelo Compose e deve ser fornecida pelo ambiente local ou pela CI.

Não foi introduzido um secret manager porque isso adicionaria complexidade desnecessária para o escopo do desafio.
