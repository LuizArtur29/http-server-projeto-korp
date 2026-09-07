# Variáveis de Ambiente

## Grafana

As credenciais administrativas do Grafana são configuradas por variáveis de ambiente para evitar valores hardcoded no Docker Compose.

Arquivo de exemplo:

```text
.env.example
```

Conteúdo:

```env
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=minha-senha
```

Criação do arquivo local:

```bash
cp .env.example .env
```

Depois, substitua `minha-senha` por uma senha local.

O arquivo `.env` real deve permanecer fora do versionamento.

## Docker Compose

O Grafana consome:

```yaml
environment:
  GF_SECURITY_ADMIN_USER: ${GF_SECURITY_ADMIN_USER:-admin}
  GF_SECURITY_ADMIN_PASSWORD: ${GF_SECURITY_ADMIN_PASSWORD:?GF_SECURITY_ADMIN_PASSWORD is required}
```

A senha não possui fallback inseguro: se não for informada, o Compose falha explicitamente.

## CI

No GitHub Actions, são utilizadas credenciais descartáveis do próprio job:

```yaml
env:
  GF_SECURITY_ADMIN_USER: admin
  GF_SECURITY_ADMIN_PASSWORD: ci-only-password
```

Não é necessário utilizar GitHub Secrets para essa credencial de teste efêmera, pois ela não protege nenhum ambiente persistente ou externo.
