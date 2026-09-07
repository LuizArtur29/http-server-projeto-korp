#!/usr/bin/env bash

set -euo pipefail

APP_URL="${APP_URL:-http://localhost/projeto-korp}"
METRICS_URL="${METRICS_URL:-http://localhost/metrics}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090/-/ready}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000/api/health}"

retry() {
  local attempts="$1"
  local delay="$2"
  shift 2

  for ((i = 1; i <= attempts; i++)); do
    if "$@"; then
      return 0
    fi

    echo "Attempt ${i}/${attempts} failed. Retrying in ${delay}s..."
    sleep "$delay"
  done

  return 1
}

echo "==> Validating application"
retry 20 3 curl --fail --silent --show-error "$APP_URL"
echo

echo "==> Validating metrics endpoint"
retry 20 3 bash -c \
  "curl --fail --silent --show-error '$METRICS_URL' | grep -q 'http_requests_total'"

echo "Metrics endpoint: OK"

echo "==> Validating Prometheus"
retry 20 3 curl --fail --silent --show-error "$PROMETHEUS_URL" > /dev/null
echo "Prometheus: OK"

echo "==> Validating Grafana"
retry 20 3 curl --fail --silent --show-error "$GRAFANA_URL" > /dev/null
echo "Grafana: OK"

echo
echo "Smoke test completed successfully."