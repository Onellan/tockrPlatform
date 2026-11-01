#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CTRL_ROOT="$(cd -- "$ROOT/../tockrCTRL" && pwd)"
IMS_ROOT="$(cd -- "$ROOT/../tockrIMS" && pwd)"
STATE_ROOT="${TOCKR_DEMO_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/tockr-demo}"
ENV_FILE="$STATE_ROOT/demo.env"
MANIFEST_FILE="$STATE_ROOT/platform-demo-fixture.json"
DEMO_PASSWORD='Tockr-Demo-2026!'

platform_compose() {
  docker compose -p tockr-platform-demo --env-file "$ENV_FILE" -f "$ROOT/compose.yaml" "$@"
}

ctrl_compose() {
  docker compose -p tockrctrl-demo --env-file "$ENV_FILE" \
    -f "$CTRL_ROOT/docker-compose.yml" -f "$CTRL_ROOT/docker-compose.local-demo.yml" "$@"
}

ims_compose() {
  docker compose -p tockrims-demo --env-file "$ENV_FILE" \
    -f "$IMS_ROOT/docker-compose.yml" -f "$IMS_ROOT/docker-compose.local-demo.yml" "$@"
}

usage() {
  cat <<'EOF'
Usage: scripts/demo-stack.sh [start|status|logs|stop|help]

start   Build, start and seed all three local applications (default).
status  Show the three Compose projects and their health.
logs    Follow logs from all three applications.
stop    Stop all three projects while preserving their demo databases.
EOF
}

ensure_env_file() {
  mkdir -p "$STATE_ROOT"
  chmod 700 "$STATE_ROOT"
  if [[ -f "$ENV_FILE" ]]; then
    chmod 600 "$ENV_FILE"
    return
  fi
  command -v openssl >/dev/null || { echo 'openssl is required to create local service keys.' >&2; exit 1; }

  local key_file="$STATE_ROOT/assertion-private.pem"
  umask 077
  openssl genpkey -algorithm Ed25519 -out "$key_file" >/dev/null 2>&1
  local seed public private
  seed="$(openssl pkey -in "$key_file" -outform DER 2>/dev/null | tail -c 32 | od -An -vtx1 | tr -d ' \n')"
  public="$(openssl pkey -in "$key_file" -pubout -outform DER 2>/dev/null | tail -c 32 | od -An -vtx1 | tr -d ' \n')"
  private="${seed}${public}"
  rm -f "$key_file"
  if [[ ${#private} -ne 128 ]]; then
    echo 'Could not generate the local Platform assertion key.' >&2
    exit 1
  fi

  cat >"$ENV_FILE" <<EOF
PLATFORM_MFA_KEY=$(openssl rand -hex 32)
PLATFORM_ASSERTION_ISSUER=https://tockr-platform.local-demo
PLATFORM_ASSERTION_KEY_ID=local-demo-v1
PLATFORM_ASSERTION_PRIVATE_KEY=$private
PLATFORM_ASSERTION_AUDIENCES=tockrctrl,tockrims
PLATFORM_ALLOW_INSECURE_COOKIES=1
TOCKR_ADMIN_EMAIL=admin@example.com
TOCKR_ADMIN_PASSWORD=$DEMO_PASSWORD
TOCKR_DEFAULT_TIMEZONE=Africa/Johannesburg
TOCKR_DEFAULT_CURRENCY=ZAR
TOCKRIMS_ADMIN_EMAIL=admin@example.com
TOCKRIMS_ADMIN_PASSWORD=$DEMO_PASSWORD
TOCKRIMS_ADMIN_NAME=Tockr Demo Administrator
TOCKRIMS_DEFAULT_ORGANISATION=Default Organisation
TOCKRIMS_SESSION_SECRET=$(openssl rand -hex 32)
TOCKRIMS_ENCRYPTION_KEY=$(openssl rand -hex 32)
EOF
  chmod 600 "$ENV_FILE"
}

preflight() {
  [[ -d "$CTRL_ROOT" && -d "$IMS_ROOT" ]] || {
    echo 'Expected sibling repositories to be named tockrCTRL and tockrIMS.' >&2
    exit 1
  }
  command -v docker >/dev/null || { echo 'docker is required inside WSL.' >&2; exit 1; }
  command -v curl >/dev/null || { echo 'curl is required inside WSL.' >&2; exit 1; }
  docker info >/dev/null 2>&1 || {
    echo 'Docker is not reachable. Start Docker Desktop and enable this WSL distribution in Settings > Resources > WSL Integration.' >&2
    exit 1
  }
}

wait_for_http() {
  local name="$1" url="$2"
  for _ in $(seq 1 90); do
    if curl --noproxy '*' --silent --show-error --fail --max-time 2 "$url" >/dev/null 2>&1; then
      printf '%s is ready at %s\n' "$name" "$url"
      return 0
    fi
    sleep 2
  done
  printf '%s did not become ready at %s. Run: %s logs\n' "$name" "$url" "$0" >&2
  return 1
}

start_stack() {
  preflight
  ensure_env_file
  echo 'Building and starting Tockr Platform...'
  platform_compose up -d --build
  wait_for_http 'Tockr Platform' 'http://127.0.0.1:8080/readyz'
  platform_compose exec -T platform /app/platform-demo-seed --manifest - >"$MANIFEST_FILE"
  chmod 644 "$MANIFEST_FILE"

  echo 'Building and starting TockrCTRL...'
  ctrl_compose up -d --build
  wait_for_http 'TockrCTRL' 'http://127.0.0.1:8029/readyz'
  ctrl_compose exec -T tockr /app/tockrctrl-demo-seed --platform-manifest - <"$MANIFEST_FILE"

  echo 'Building and starting TockrIMS...'
  ims_compose up -d --build
  wait_for_http 'TockrIMS' 'http://127.0.0.1:8030/readyz'
  ims_compose exec -T app /app/tockrims-demo-seed --manifest - <"$MANIFEST_FILE"

  echo
  echo 'All three local demo apps are running:'
  echo '  Platform  http://localhost:8080'
  echo '  TockrCTRL http://localhost:8029'
  echo '  TockrIMS  http://localhost:8030'
  echo '  Login     admin@example.com'
  echo "  Password  $DEMO_PASSWORD"
  echo 'Use the same password for anika.demo@tockrctrl.local, sipho.demo@tockrctrl.local, maya.demo@tockrctrl.local, and daniel.demo@tockrctrl.local.'
}

show_status() {
  ensure_env_file
  platform_compose ps
  ctrl_compose ps
  ims_compose ps
}

show_logs() {
  ensure_env_file
  docker compose -p tockr-platform-demo --env-file "$ENV_FILE" -f "$ROOT/compose.yaml" logs --tail=100 --no-color
  docker compose -p tockrctrl-demo --env-file "$ENV_FILE" -f "$CTRL_ROOT/docker-compose.yml" -f "$CTRL_ROOT/docker-compose.local-demo.yml" logs --tail=100 --no-color
  docker compose -p tockrims-demo --env-file "$ENV_FILE" -f "$IMS_ROOT/docker-compose.yml" -f "$IMS_ROOT/docker-compose.local-demo.yml" logs --tail=100 --no-color
}

stop_stack() {
  ensure_env_file
  platform_compose down
  ctrl_compose down
  ims_compose down
  echo 'Stopped the three demo stacks. Their volumes and demo data are preserved.'
}

case "${1:-start}" in
  start) start_stack ;;
  status) show_status ;;
  logs) show_logs ;;
  stop) stop_stack ;;
  help|-h|--help) usage ;;
  *) usage >&2; exit 2 ;;
esac
