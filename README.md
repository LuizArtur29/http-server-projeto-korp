# http-server-projeto-korp

Projeto desenvolvido como parte de um desafio técnico DevOps, com foco em construção de serviço HTTP em Go, containerização, redes Docker, proxy reverso com NGINX, observabilidade e automação com Ansible.

## Status

- [x] Serviço HTTP em Go
- [x] Containerização da aplicação
- [x] Docker Compose
- [x] Rede Docker bridge
- [x] NGINX como proxy reverso
- [x] Prometheus
- [x] Grafana
- [x] Dashboard de observabilidade
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

# Parte 2 — Monitoramento e Observabilidade

A segunda etapa do projeto adiciona uma camada de monitoramento ao serviço `http-server-projeto-korp`.

A arquitetura utiliza Prometheus para coleta e armazenamento das métricas e Grafana para visualização dos dados.

## Arquitetura de observabilidade

```text
                              Host
                               │
                               │ :80
                               ▼
                        ┌─────────────┐
                        │    NGINX    │
                        └──────┬──────┘
                               │
                               │ :8080
                               ▼
                    ┌──────────────────────────┐
                    │ http-server-projeto-korp │
                    │                          │
                    │ /projeto-korp            │
                    │ /healthz                 │
                    │ /metrics                 │
                    └────────────┬─────────────┘
                                 │
                                 │ scrape /metrics
                                 ▼
                         ┌──────────────┐
                         │  Prometheus  │
                         │    :9090     │
                         └──────┬───────┘
                                │
                                │ datasource
                                ▼
                          ┌───────────┐
                          │  Grafana  │
                          │   :3000   │
                          └───────────┘

                         Docker bridge network
```

O Prometheus acessa diretamente o serviço Go através da rede interna do Docker.

O tráfego de monitoramento não passa pelo NGINX, pois o proxy reverso é utilizado como ponto de entrada da aplicação, enquanto a coleta de métricas ocorre internamente entre os containers.

---

## Prometheus

O Prometheus é responsável por realizar o scrape periódico das métricas expostas pela aplicação.

A aplicação disponibiliza as métricas através do endpoint:

```text
/metrics
```

O target configurado no Prometheus utiliza o DNS interno do Docker:

```text
http-server-projeto-korp:8080
```

Exemplo da configuração:

```yaml
scrape_configs:
  - job_name: "http-server-projeto-korp"
    metrics_path: /metrics

    static_configs:
      - targets:
          - "http-server-projeto-korp:8080"
```

Essa abordagem evita dependência de endereços IP dos containers.

---

## Disponibilidade do serviço

A disponibilidade é monitorada utilizando a métrica nativa do Prometheus:

```promql
up{job="http-server-projeto-korp"}
```

O Prometheus gera automaticamente essa métrica a cada tentativa de scrape.

Valores possíveis:

```text
1 → target disponível
0 → target indisponível
```

Essa abordagem evita a criação de uma métrica customizada apenas para representar disponibilidade.

O endpoint `/healthz` continua disponível para health checks, troubleshooting e futuras integrações, enquanto a métrica `up` é utilizada para monitoramento pelo Prometheus.

---

## Volume de requisições

A aplicação implementa um contador Prometheus:

```text
http_requests_total
```

A métrica utiliza as labels:

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

Isso permite analisar não somente o número total de requisições, mas também separar o tráfego por método, endpoint e código HTTP.

---

## Taxa de requisições

Para análise do comportamento do serviço ao longo do tempo, o dashboard utiliza `rate()` em vez de apenas apresentar o valor acumulado do contador.

Consulta utilizada:

```promql
sum(
  rate(
    http_requests_total{
      route="/projeto-korp"
    }[1m]
  )
)
```

O resultado representa aproximadamente a quantidade de requisições processadas por segundo.

---

## Latência

Além das métricas obrigatórias, a aplicação expõe um histograma:

```text
http_request_duration_seconds
```

A partir dele, o dashboard calcula a latência no percentil 95.

Consulta:

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(
      http_request_duration_seconds_bucket{
        route="/projeto-korp"
      }[5m]
    )
  )
)
```

O p95 representa o tempo abaixo do qual aproximadamente 95% das requisições foram processadas.

Essa métrica fornece uma visão mais útil do comportamento do serviço do que apenas uma média de latência.

---

## Requisições simultâneas

A aplicação também expõe:

```text
http_requests_in_flight
```

Essa métrica representa o número de requisições sendo processadas simultaneamente.

Ela permite observar alterações no nível de concorrência durante testes de carga ou picos de tráfego.

---

# Grafana

O Grafana é utilizado como camada de visualização das métricas armazenadas no Prometheus.

O datasource Prometheus é configurado automaticamente através do mecanismo de provisioning do Grafana.

O Grafana se comunica com:

```text
http://prometheus:9090
```

utilizando a rede interna do Docker.

---

## Provisionamento automático do datasource

Arquivo:

```text
monitoring/grafana/provisioning/datasources/datasource.yml
```

Exemplo:

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

Foi definido um UID fixo:

```text
prometheus
```

Isso permite que o dashboard referencie o datasource de forma determinística, sem depender de IDs gerados automaticamente pelo Grafana.

---

## Provisionamento automático do dashboard

O carregamento de dashboards também é configurado através de provisioning.

Arquivo:

```text
monitoring/grafana/provisioning/dashboards/dashboards.yml
```

Os dashboards são armazenados em:

```text
monitoring/grafana/dashboards/
```

e montados no container no caminho:

```text
/var/lib/grafana/dashboards
```

O dashboard principal é definido no arquivo:

```text
http-server-projeto-korp-dashboard.json
```

Dessa forma, não é necessário criar manualmente o datasource ou reconstruir o dashboard após recriar o container.

---

# Dashboard

O dashboard:

```text
HTTP Server - Projeto Korp
```

foi criado para apresentar tanto os requisitos mínimos do desafio quanto métricas adicionais úteis para análise operacional.

Os painéis implementados são:

| Painel | Finalidade |
|---|---|
| Service Availability | Indica se o serviço está disponível para o Prometheus |
| Total Requests | Total de requisições recebidas |
| Request Rate | Taxa de requisições por segundo |
| Latency p95 | Percentil 95 da duração das requisições |
| Requests by HTTP Status | Distribuição das requisições por status HTTP |
| Requests In Flight | Requisições processadas simultaneamente |

---

## Service Availability

Consulta:

```promql
up{job="http-server-projeto-korp"}
```

O painel utiliza uma visualização `Stat`, apresentando:

```text
UP
```

quando o serviço está disponível.

---

## Total Requests

Consulta:

```promql
sum(
  http_requests_total{
    route="/projeto-korp"
  }
)
```

Apresenta o total acumulado de requisições do endpoint principal.

---

## Request Rate

Consulta:

```promql
sum(
  rate(
    http_requests_total{
      route="/projeto-korp"
    }[1m]
  )
)
```

O painel apresenta a evolução do throughput da aplicação ao longo do tempo.

---

## Latency p95

Consulta:

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(
      http_request_duration_seconds_bucket{
        route="/projeto-korp"
      }[5m]
    )
  )
)
```

Permite observar variações no tempo de resposta do serviço.

---

## Requests by HTTP Status

Consulta:

```promql
sum by (status) (
  rate(
    http_requests_total{
      route="/projeto-korp"
    }[5m]
  )
)
```

As séries são separadas utilizando o código HTTP.

Por exemplo:

```text
HTTP 200
HTTP 405
```

Isso facilita a identificação de aumento em respostas de erro.

---

## Requests In Flight

Consulta:

```promql
http_requests_in_flight
```

Esse painel permite acompanhar o nível de concorrência da aplicação.

---

# Exposição dos componentes de observabilidade

Prometheus e Grafana são ferramentas administrativas e não precisam ficar disponíveis publicamente.

Por esse motivo, suas portas são vinculadas apenas ao loopback do host:

```yaml
ports:
  - "127.0.0.1:9090:9090"
```

para o Prometheus e:

```yaml
ports:
  - "127.0.0.1:3000:3000"
```

para o Grafana.

Assim:

```text
Prometheus → localhost:9090
Grafana    → localhost:3000
```

e não ficam diretamente disponíveis em outras interfaces de rede da máquina.

---

# Imagens Docker

Foram utilizadas imagens oficiais dos respectivos projetos com versões explicitamente definidas.

Exemplo:

```text
prom/prometheus:<versão-pinada>
grafana/grafana:<versão-pinada>
```

Foi evitado o uso da tag:

```text
latest
```

para reduzir variações entre diferentes execuções do ambiente.

Para componentes de infraestrutura de terceiros, foi priorizada a utilização das imagens mantidas oficialmente pelos projetos em vez da criação de imagens customizadas.

---

# Persistência da configuração

As configurações do Prometheus e Grafana são mantidas no repositório e montadas nos containers como volumes.

Sempre que possível, arquivos de configuração são montados como somente leitura:

```text
:ro
```

Isso impede alterações acidentais dentro dos containers e mantém o repositório como fonte de verdade da configuração.

---

# Validação

## Verificar containers

```bash
docker compose ps
```

Os componentes esperados são:

```text
http-server-projeto-korp
nginx
prometheus
grafana
```

---

## Verificar Prometheus

A interface pode ser acessada em:

```text
http://localhost:9090
```

Consulta para disponibilidade:

```promql
up{job="http-server-projeto-korp"}
```

Resultado esperado:

```text
1
```

---

## Gerar tráfego

Exemplo:

```bash
seq 1 100 | xargs -n1 -P10   curl -s http://localhost/projeto-korp > /dev/null
```

Após a execução, as métricas de volume e taxa de requisições devem apresentar alteração.

---

## Testar códigos HTTP diferentes

Para validar a separação por status:

```bash
for i in $(seq 1 20); do
  curl -s -X POST     http://localhost/projeto-korp > /dev/null
done
```

Como o endpoint aceita apenas `GET`, essas requisições devem resultar em:

```text
405 Method Not Allowed
```

e aparecer no dashboard como uma série separada.

---

# Decisões arquiteturais da Parte 2

| Decisão | Motivação |
|---|---|
| Prometheus para coleta | Solução adequada ao formato de métricas utilizado |
| `/metrics` diretamente no serviço | Coleta interna sem necessidade de passar pelo proxy |
| Métrica nativa `up` | Evita duplicar conceito de disponibilidade |
| Counter para requisições | Modelo apropriado para eventos acumulativos |
| Histogram para latência | Permite cálculo de percentis como p95 |
| Labels controladas | Permitem análise sem cardinalidade desnecessária |
| Grafana para visualização | Dashboards operacionais sobre Prometheus |
| UID fixo no datasource | Referência determinística nos dashboards |
| Provisioning por arquivos | Ambiente reproduzível sem configuração manual |
| Dashboard versionado em JSON | Infraestrutura e observabilidade como código |
| Portas administrativas em loopback | Redução da superfície de exposição |
| Configurações montadas read-only | Evita alterações acidentais |
| Imagens com versão pinada | Maior previsibilidade do ambiente |
| Dashboard com p95 e status HTTP | Observabilidade além do mínimo exigido |

---

# Resultado da Parte 2

Com a implementação da camada de observabilidade, o ambiente permite acompanhar:

```text
Disponibilidade
Volume de requisições
Taxa de requisições
Latência p95
Status HTTP
Concorrência
```

O Prometheus coleta automaticamente as métricas do serviço e o Grafana é inicializado com datasource e dashboard provisionados através de arquivos versionados no repositório.

Isso permite recriar a camada de observabilidade sem configuração manual.

---

# Parte 3 — Automação com Ansible

A terceira etapa do projeto automatiza o provisionamento completo da infraestrutura utilizada pelo `http-server-projeto-korp`.

O objetivo é permitir que todo o ambiente seja preparado e validado com um único comando Ansible, contemplando instalação do Docker, sincronização dos arquivos do projeto, criação da rede Docker, build da imagem da aplicação, inicialização da stack com Docker Compose e validação dos principais serviços.

## Objetivos da automação

A automação cobre:

- instalação do Docker;
- habilitação do serviço Docker;
- detecção automática da distribuição Linux;
- suporte a Debian e Arch Linux/CachyOS;
- sincronização do projeto para o host;
- criação da rede Docker;
- build da imagem da aplicação;
- inicialização da stack via Docker Compose;
- configuração do NGINX;
- configuração do Prometheus;
- configuração do Grafana;
- validação HTTP da aplicação;
- validação do Prometheus;
- validação do Grafana.

## Estrutura do Ansible

```text
ansible/
├── ansible.cfg
├── inventory/
│   └── hosts.ini
├── group_vars/
│   └── all.yml
├── requirements.yml
├── roles/
│   ├── docker/
│   │   ├── defaults/
│   │   │   └── main.yml
│   │   └── tasks/
│   │       ├── main.yml
│   │       ├── debian.yml
│   │       └── archlinux.yml
│   ├── deploy/
│   │   └── tasks/
│   │       └── main.yml
│   └── validate/
│       └── tasks/
│           └── main.yml
└── site.yml
```

| Role | Responsabilidade |
|---|---|
| `docker` | Instalação e configuração do Docker |
| `deploy` | Sincronização do projeto, rede, imagem e Compose |
| `validate` | Validação da aplicação e dos componentes de monitoramento |

## Execução

Dentro da pasta `ansible`:

```bash
ansible-playbook site.yml -K
```

A opção `-K` solicita a senha usada pelo `become`, permitindo executar tarefas administrativas com privilégios elevados.

O playbook principal:

```yaml
---
- name: Provision Projeto Korp environment
  hosts: korp
  become: true

  roles:
    - docker
    - deploy
    - validate
```

## Inventory

Para validação local:

```ini
[korp]
localhost ansible_connection=local
```

Para um host remoto, o inventory pode ser alterado, por exemplo:

```ini
[korp]
192.168.1.100 ansible_user=usuario
```

## Variáveis

Arquivo:

```text
ansible/group_vars/all.yml
```

Exemplo:

```yaml
project_name: http-server-projeto-korp
project_dir: /opt/http-server-projeto-korp
docker_network_name: korp-network

service_url: http://localhost/projeto-korp
```

## Dependências Ansible

Arquivo:

```text
ansible/requirements.yml
```

```yaml
---
collections:
  - name: community.docker
  - name: community.general
  - name: ansible.posix
```

Instalação:

```bash
ansible-galaxy collection install -r requirements.yml
```

## Role Docker

A role `docker` detecta automaticamente o sistema operacional com Ansible facts.

```yaml
- name: Display detected operating system
  ansible.builtin.debug:
    msg: >-
      Distribution={{ ansible_facts["distribution"] }}
      Family={{ ansible_facts["os_family"] }}
      Architecture={{ ansible_facts["architecture"] }}
```

Durante a validação em CachyOS:

```text
Distribution=Archlinux
Family=Archlinux
Architecture=x86_64
```

### Seleção da distribuição

```yaml
- name: Install Docker on Debian-based systems
  ansible.builtin.include_tasks: debian.yml
  when: ansible_facts["os_family"] == "Debian"

- name: Install Docker on Arch Linux-based systems
  ansible.builtin.include_tasks: archlinux.yml
  when: ansible_facts["os_family"] == "Archlinux"
```

Sistemas não suportados geram falha explícita:

```yaml
- name: Fail when operating system is unsupported
  ansible.builtin.fail:
    msg: >-
      Unsupported operating system:
      {{ ansible_facts["distribution"] }}
      ({{ ansible_facts["os_family"] }}).
  when:
    - ansible_facts["os_family"] != "Debian"
    - ansible_facts["os_family"] != "Archlinux"
```

## Docker em Arch Linux / CachyOS

```yaml
---
- name: Install Docker packages on Arch Linux
  community.general.pacman:
    name:
      - docker
      - docker-compose
    state: present
```

## Docker em Debian

Para Debian, a automação utiliza o repositório oficial do Docker.

O fluxo contempla:

```text
dependências do repositório
/etc/apt/keyrings
chave de assinatura
repositório Docker
cache APT
Docker Engine
Docker Compose Plugin
```

Mapeamento de arquitetura:

```yaml
docker_apt_architecture_map:
  x86_64: amd64
  aarch64: arm64
```

## Serviço Docker

```yaml
- name: Ensure Docker service is enabled and running
  ansible.builtin.service:
    name: docker
    state: started
    enabled: true
```

## Role Deploy

Fluxo:

```text
Criar diretório
      ↓
Sincronizar arquivos
      ↓
Criar rede Docker
      ↓
Build da imagem
      ↓
Executar Docker Compose
```

### Diretório do projeto

```yaml
- name: Create project directory
  ansible.builtin.file:
    path: "{{ project_dir }}"
    state: directory
    mode: "0755"
```

O destino utilizado é:

```text
/opt/http-server-projeto-korp
```

### Sincronização do projeto

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
```

A sincronização via `rsync` evita alterações desnecessárias quando os arquivos já estão atualizados.

### Rede Docker

```yaml
- name: Ensure Docker network exists
  community.docker.docker_network:
    name: "{{ docker_network_name }}"
    driver: bridge
    state: present
```

A rede utilizada é:

```text
korp-network
```

No Compose:

```yaml
networks:
  korp-network:
    name: korp-network
    external: true
```

Assim, a criação da rede fica explicitamente sob responsabilidade do Ansible.

### Build da aplicação

```yaml
- name: Build application Docker image
  community.docker.docker_image:
    name: http-server-projeto-korp
    tag: local
    source: build
    build:
      path: "{{ project_dir }}"
    state: present
    force_source: false
```

Imagem resultante:

```text
http-server-projeto-korp:local
```

### Docker Compose

```yaml
- name: Start Docker Compose stack
  community.docker.docker_compose_v2:
    project_src: "{{ project_dir }}"
    state: present
    pull: missing
```

A stack inclui:

```text
http-server-projeto-korp
nginx
prometheus
grafana
```

## Check mode

O playbook aceita:

```bash
ansible-playbook site.yml --check -K
```

Operações que dependem de alterações reais são puladas com:

```yaml
when: not ansible_check_mode
```

## Role Validate

A role valida:

```text
Aplicação
Prometheus
Grafana
```

### Validação da aplicação

```yaml
- name: Wait for HTTP service to become available
  ansible.builtin.uri:
    url: "{{ service_url }}"
    method: GET
    status_code: 200
    return_content: true
  register: service_response
  retries: 10
  delay: 3
  until: service_response.status == 200
  when: not ansible_check_mode
```

A resposta é exibida:

```yaml
- name: Display service response
  ansible.builtin.debug:
    var: service_response.json
  when: not ansible_check_mode
```

Exemplo validado:

```json
{
  "horario": "2026-09-05T01:00:56Z",
  "nome": "Projeto Korp"
}
```

### Validação do Prometheus

Endpoint:

```text
http://localhost:9090/-/ready
```

```yaml
- name: Validate Prometheus
  ansible.builtin.uri:
    url: http://localhost:9090/-/ready
    method: GET
    status_code: 200
  register: prometheus_response
  retries: 10
  delay: 3
  until: prometheus_response.status == 200
  when: not ansible_check_mode
```

### Validação do Grafana

Endpoint:

```text
http://localhost:3000/api/health
```

```yaml
- name: Validate Grafana
  ansible.builtin.uri:
    url: http://localhost:3000/api/health
    method: GET
    status_code: 200
    return_content: true
  register: grafana_response
  retries: 10
  delay: 3
  until: grafana_response.status == 200
  when: not ansible_check_mode
```

Resumo final:

```text
Application: OK
Prometheus: OK
Grafana: ok
```

## Idempotência

Após o primeiro provisionamento, uma nova execução produziu:

```text
PLAY RECAP
localhost : ok=15 changed=0 unreachable=0 failed=0 skipped=3
```

O resultado `changed=0` demonstra que o ambiente já estava no estado desejado e nenhuma alteração desnecessária foi aplicada.

As validações continuam sendo executadas, mesmo sem alterações.

## Decisões arquiteturais

| Decisão | Motivação |
|---|---|
| Roles separadas | Organização de responsabilidades |
| Ansible facts | Detecção automática do sistema operacional |
| Suporte Debian e Arch/CachyOS | Maior portabilidade |
| Falha explícita em SO não suportado | Evita execução em ambiente não validado |
| `community.docker` | Gerenciamento declarativo de recursos Docker |
| `ansible.posix.synchronize` | Sincronização eficiente e idempotente |
| Rede criada pelo Ansible | Atende explicitamente ao requisito |
| Rede externa no Compose | Evita duplicação de redes |
| `docker_compose_v2` | Gerenciamento declarativo da stack |
| Validação com `uri` | Evita dependência de shell/curl |
| Retries | Trata o tempo de inicialização dos serviços |
| Check mode | Permite simular o playbook |
| `changed=0` na segunda execução | Demonstra idempotência |
| Um único playbook | Simplifica o provisionamento completo |

## Fluxo completo

```text
ansible-playbook site.yml -K
            │
            ▼
      Gathering Facts
            │
            ▼
    Detecta distribuição
            │
     ┌──────┴──────┐
     ▼             ▼
  Debian       Arch/CachyOS
     │             │
     └──────┬──────┘
            ▼
      Instala Docker
            │
            ▼
     Habilita Docker
            │
            ▼
     Sincroniza projeto
            │
            ▼
     Cria korp-network
            │
            ▼
        Build image
            │
            ▼
   Docker Compose stack
            │
     ┌──────┼────────┐
     ▼      ▼        ▼
   NGINX Prometheus Grafana
     │
     ▼
 Aplicação Go
            │
            ▼
       Validações
            │
            ▼
     Provisionamento OK
```

## Validação recomendada

### Verificar sintaxe

```bash
ansible-playbook --syntax-check site.yml
```

### Executar em check mode

```bash
ansible-playbook site.yml --check -K
```

### Executar provisionamento

```bash
ansible-playbook site.yml -K
```

### Validar idempotência

Execute novamente:

```bash
ansible-playbook site.yml -K
```

Resultado esperado:

```text
changed=0
failed=0
```

## Resultado da Parte 3

A infraestrutura completa do projeto pode ser provisionada através de um único comando Ansible.

O processo automatiza:

```text
Docker
Rede Docker
Build da aplicação
Docker Compose
NGINX
Prometheus
Grafana
Validação HTTP
Validação da observabilidade
```

A implementação apresenta comportamento idempotente, permitindo repetir o playbook sem alterações desnecessárias.

## Próximos passos

Com as três partes principais concluídas, os próximos passos recomendados são:

```text
Revisão final contra os critérios do desafio
Documentação consolidada no README
Validação em uma instalação limpa Debian
Revisão do repositório público
CI para testes, lint e build
```

Possíveis verificações de CI:

```text
gofmt
go vet
go test
docker build
ansible-lint
```

