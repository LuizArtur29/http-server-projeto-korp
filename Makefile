APP_IMAGE=http-server-projeto-korp:local
NETWORK=korp-network

.PHONY: test build network up down ps logs smoke \
        ansible-check ansible-run ansible-idempotence \
        compose-validate nginx-validate prometheus-validate ci-local

test:
	go test ./...

build:
	docker build -t $(APP_IMAGE) .

network:
	@docker network inspect $(NETWORK) >/dev/null 2>&1 || \
		docker network create $(NETWORK)

up: network
	docker compose up -d

down:
	docker compose down

ps:
	docker compose ps

logs:
	docker compose logs -f

smoke:
	./scripts/smoke-test.sh

compose-validate:
	docker compose config

nginx-validate:
	docker compose exec -T nginx nginx -t

prometheus-validate:
	docker compose exec -T prometheus \
		promtool check config /etc/prometheus/prometheus.yml

ansible-check:
	cd ansible && ansible-playbook --syntax-check site.yml

ansible-run:
	cd ansible && ansible-playbook site.yml -K

ansible-idempotence:
	cd ansible && ansible-playbook site.yml -K
	cd ansible && ansible-playbook site.yml -K

ci-local: test compose-validate build up nginx-validate prometheus-validate smoke