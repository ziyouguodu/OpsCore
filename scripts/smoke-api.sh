#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deploy/docker-compose.yml}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-ChangeMe123!}"

docker compose -f "$COMPOSE_FILE" exec -T \
  -e ADMIN_USERNAME="$ADMIN_USERNAME" \
  -e ADMIN_PASSWORD="$ADMIN_PASSWORD" \
  backend sh -s <<'EOS'
set -eu

HOST=127.0.0.1
PORT=8080
LAST_STATUS=
LAST_BODY=
ADMIN_TOKEN=
OPS_TOKEN=
SMOKE_USER_ID=
SMOKE_ASSET_ID=
SMOKE_TASK_ID=
SMOKE_INCIDENT_ID=
SMOKE_ONCALL_ID=

json_string() {
  key="$1"
  printf '%s' "$LAST_BODY" | sed -n "s/.*\"$key\":\"\([^\"]*\)\".*/\1/p"
}

json_id() {
  key="$1"
  printf '%s' "$LAST_BODY" | sed -n "s/.*\"$key\":\([0-9][0-9]*\).*/\1/p"
}

request() {
  method="$1"
  path="$2"
  body="${3:-}"
  token="${4:-}"
  expected="$5"
  body_file=$(mktemp)
  if [ -n "$token" ] && [ -n "$body" ]; then
    LAST_STATUS=$(curl -sS -o "$body_file" -w '%{http_code}' -X "$method" -H 'Accept: application/json' -H 'Content-Type: application/json' -H "Authorization: Bearer $token" -d "$body" "http://$HOST:$PORT$path")
  elif [ -n "$token" ]; then
    LAST_STATUS=$(curl -sS -o "$body_file" -w '%{http_code}' -X "$method" -H 'Accept: application/json' -H "Authorization: Bearer $token" "http://$HOST:$PORT$path")
  elif [ -n "$body" ]; then
    LAST_STATUS=$(curl -sS -o "$body_file" -w '%{http_code}' -X "$method" -H 'Accept: application/json' -H 'Content-Type: application/json' -d "$body" "http://$HOST:$PORT$path")
  else
    LAST_STATUS=$(curl -sS -o "$body_file" -w '%{http_code}' -X "$method" -H 'Accept: application/json' "http://$HOST:$PORT$path")
  fi
  LAST_BODY=$(cat "$body_file")
  rm -f "$body_file"

  if [ "$LAST_STATUS" != "$expected" ]; then
    echo "FAIL $method $path expected $expected got ${LAST_STATUS:-no-status}" >&2
    echo "$LAST_BODY" >&2
    if [ "$method" = "POST" ] && [ "$path" = "/api/auth/login" ] && [ "$expected" = "200" ] && [ "$LAST_STATUS" = "401" ]; then
      echo "HINT admin may already use an initialized password. Rerun with ADMIN_PASSWORD='your-current-password' scripts/smoke-api.sh" >&2
    fi
    exit 1
  fi
  echo "OK   $method $path -> $LAST_STATUS"
}

cleanup_request() {
  method="$1"
  path="$2"
  token="$3"
  curl -sS -o /dev/null -X "$method" -H "Authorization: Bearer $token" "http://$HOST:$PORT$path" >/dev/null 2>&1 || true
}

cleanup() {
  if [ -n "$ADMIN_TOKEN" ]; then
    [ -n "$SMOKE_ONCALL_ID" ] && cleanup_request DELETE "/api/oncall/$SMOKE_ONCALL_ID" "$ADMIN_TOKEN"
    [ -n "$SMOKE_INCIDENT_ID" ] && cleanup_request DELETE "/api/incidents/$SMOKE_INCIDENT_ID" "$ADMIN_TOKEN"
    [ -n "$SMOKE_TASK_ID" ] && cleanup_request DELETE "/api/tasks/$SMOKE_TASK_ID" "$ADMIN_TOKEN"
    [ -n "$SMOKE_ASSET_ID" ] && cleanup_request DELETE "/api/assets/$SMOKE_ASSET_ID" "$ADMIN_TOKEN"
    [ -n "$SMOKE_USER_ID" ] && cleanup_request DELETE "/api/users/$SMOKE_USER_ID" "$ADMIN_TOKEN"
  fi
  true
}
trap cleanup EXIT

stamp=$(date +%s)
ops_username="smoke_ops_$stamp"
ops_password="SmokePass123!"

request POST /api/auth/login "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}" "" 200
ADMIN_TOKEN=$(json_string token)
if [ -z "$ADMIN_TOKEN" ]; then
  echo "FAIL admin login did not return token" >&2
  exit 1
fi

request GET /api/auth/me "" "$ADMIN_TOKEN" 200
if printf '%s' "$LAST_BODY" | grep -q '"mustChangePassword":true'; then
  request GET /api/dashboard "" "$ADMIN_TOKEN" 403
  echo "PASS initial-password gate is active. Initialize the admin password, then rerun this smoke test for the full flow."
  exit 0
fi

request GET /api/dashboard "" "$ADMIN_TOKEN" 200

request POST /api/users "{\"username\":\"$ops_username\",\"displayName\":\"Smoke Ops\",\"password\":\"$ops_password\",\"mustChangePassword\":false,\"roles\":[\"ops_engineer\"]}" "$ADMIN_TOKEN" 201
SMOKE_USER_ID=$(json_id id)

request POST /api/auth/login "{\"username\":\"$ops_username\",\"password\":\"$ops_password\"}" "" 200
OPS_TOKEN=$(json_string token)

request POST /api/users "{\"username\":\"blocked_$stamp\",\"displayName\":\"Blocked\",\"password\":\"$ops_password\",\"roles\":[\"ops_engineer\"]}" "$OPS_TOKEN" 403

request POST /api/assets "{\"type\":\"物理机\",\"cpuArch\":\"x86_64\",\"business\":\"smoke-business\",\"ipv4\":\"10.255.0.10\",\"environment\":\"生产\",\"os\":\"Ubuntu\",\"networkZone\":\"smoke-zone\",\"cpu\":\"4C\",\"memory\":\"8GB\",\"disk\":\"100GB\",\"deploymentInfo\":\"smoke-deploy\",\"owner\":\"Smoke Ops\",\"connectedStatus\":\"已并网\",\"status\":\"运行中\"}" "$OPS_TOKEN" 201
SMOKE_ASSET_ID=$(json_id id)

request PUT "/api/assets/$SMOKE_ASSET_ID/credential" "{\"loginUrl\":\"ssh://10.255.0.10\",\"username\":\"root\",\"secret\":\"SmokeSecret123\",\"notes\":\"smoke\"}" "$OPS_TOKEN" 403
request PUT "/api/assets/$SMOKE_ASSET_ID/credential" "{\"loginUrl\":\"ssh://10.255.0.10\",\"username\":\"root\",\"secret\":\"SmokeSecret123\",\"notes\":\"smoke\"}" "$ADMIN_TOKEN" 200
if printf '%s' "$LAST_BODY" | grep -q 'SmokeSecret123'; then
  echo "FAIL credential save response exposed plaintext secret" >&2
  exit 1
fi
request POST "/api/assets/$SMOKE_ASSET_ID/credential/reveal" "{\"password\":\"$ADMIN_PASSWORD\"}" "$ADMIN_TOKEN" 200
if ! printf '%s' "$LAST_BODY" | grep -q 'SmokeSecret123'; then
  echo "FAIL credential reveal did not return secret after password verification" >&2
  exit 1
fi

request POST /api/tasks "{\"title\":\"Smoke task\",\"assignee\":\"Smoke Ops\"}" "$ADMIN_TOKEN" 201
SMOKE_TASK_ID=$(json_id id)
if ! printf '%s' "$LAST_BODY" | grep -q '"type":"任务"'; then
  echo "FAIL task create did not default type" >&2
  exit 1
fi
request POST /api/tasks "{\"title\":\"Bad task\",\"status\":\"挂起\"}" "$ADMIN_TOKEN" 400
request PATCH "/api/tasks/$SMOKE_TASK_ID" "{\"status\":\"处理中\"}" "$ADMIN_TOKEN" 200

request POST /api/incidents "{\"title\":\"Smoke incident\"}" "$ADMIN_TOKEN" 201
SMOKE_INCIDENT_ID=$(json_id id)
if ! printf '%s' "$LAST_BODY" | grep -q '"level":"P3"'; then
  echo "FAIL incident create did not default level" >&2
  exit 1
fi
request POST /api/incidents "{\"title\":\"Bad incident\",\"level\":\"P0\"}" "$ADMIN_TOKEN" 400
request PATCH "/api/incidents/$SMOKE_INCIDENT_ID" "{\"status\":\"处理中\"}" "$ADMIN_TOKEN" 200

request POST /api/oncall "{\"ruleType\":\"monthly\",\"primary\":\"Smoke Ops\"}" "$ADMIN_TOKEN" 400
request POST /api/oncall "{\"ruleType\":\"daily\",\"date\":\"2026-06-08\",\"primary\":\"Smoke Ops\",\"backup\":\"Smoke Backup\"}" "$ADMIN_TOKEN" 201
SMOKE_ONCALL_ID=$(json_id id)

echo "PASS OpsCore API smoke flow completed."
EOS
