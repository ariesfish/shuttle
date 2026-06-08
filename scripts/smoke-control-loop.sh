#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="${ROOT_DIR}/src"
SMOKE_DIR="${SMOKE_DIR:-$(mktemp -d -t inference-control-loop-smoke.XXXXXX)}"
API_HOST="${API_HOST:-127.0.0.1}"
API_PORT="${API_PORT:-0}"
POLL_INTERVAL="${POLL_INTERVAL:-250}"
TASK_TIMEOUT_SECONDS="${TASK_TIMEOUT_SECONDS:-45}"
KEEP_SMOKE_DIR="${KEEP_SMOKE_DIR:-true}"

mkdir -p "${SMOKE_DIR}"

cleanup() {
  local exit_code=$?
  if [[ -n "${AGENT_PID:-}" ]]; then
    kill "${AGENT_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${API_PID:-}" ]]; then
    kill "${API_PID}" >/dev/null 2>&1 || true
  fi
  if [[ "${KEEP_SMOKE_DIR}" != "true" && ${exit_code} -eq 0 ]]; then
    rm -rf "${SMOKE_DIR}"
  fi
  exit "${exit_code}"
}
trap cleanup EXIT

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require_cmd curl
require_cmd go
require_cmd python3

choose_port() {
  if [[ "${API_PORT}" != "0" ]]; then
    printf '%s\n' "${API_PORT}"
    return
  fi
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

API_PORT="$(choose_port)"
API_ADDR="${API_HOST}:${API_PORT}"
API_URL="http://${API_ADDR}"

json_request() {
  local method="$1"
  local path="$2"
  local payload="${3:-}"
  python3 - "$API_URL" "$method" "$path" "$payload" <<'PY'
import json
import sys
import urllib.request
import urllib.error

base, method, path, payload = sys.argv[1:]
data = None if payload == "" else payload.encode()
headers = {"Content-Type": "application/json"} if data is not None else {}
request = urllib.request.Request(base + path, data=data, method=method, headers=headers)
try:
    with urllib.request.urlopen(request, timeout=10) as response:
        body = response.read().decode()
except urllib.error.HTTPError as exc:
    body = exc.read().decode()
    print(body, file=sys.stderr)
    raise
print(body)
PY
}

json_get() {
  json_request GET "$1"
}

json_post() {
  json_request POST "$1" "$2"
}

json_post_empty() {
  json_request POST "$1"
}

json_field() {
  local field="$1"
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "${field}"
}

wait_until() {
  local label="$1"
  local timeout_seconds="$2"
  local command="$3"
  local deadline=$((SECONDS + timeout_seconds))
  local last_status=0
  while (( SECONDS < deadline )); do
    if bash -c "${command}" >/dev/null 2>&1; then
      return 0
    fi
    last_status=$?
    sleep "$(python3 - <<PY
print(${POLL_INTERVAL} / 1000)
PY
)"
  done
  echo "timed out waiting for ${label}; last status=${last_status}" >&2
  return 1
}

wait_task() {
  local task_id="$1"
  local deadline=$((SECONDS + TASK_TIMEOUT_SECONDS))
  local status=""
  while (( SECONDS < deadline )); do
    local tasks
    tasks="$(json_get /v1/tasks)"
    status="$(python3 -c 'import json,sys; tid=sys.argv[1]; data=json.load(sys.stdin); print(next((t["status"] for t in data if t["id"] == tid), ""))' "${task_id}" <<<"${tasks}")"
    if [[ "${status}" == "succeeded" ]]; then
      return 0
    fi
    if [[ "${status}" == "failed" ]]; then
      echo "task ${task_id} failed" >&2
      printf '%s\n' "${tasks}" >&2
      return 1
    fi
    sleep "$(python3 - <<PY
print(${POLL_INTERVAL} / 1000)
PY
)"
  done
  echo "timed out waiting for task ${task_id}; status=${status}" >&2
  json_get /v1/tasks >&2 || true
  return 1
}

create_task_and_wait() {
  local label="$1"
  local path="$2"
  local task task_id
  task="$(json_post_empty "${path}")"
  task_id="$(json_field id <<<"${task}")"
  wait_task "${task_id}"
  printf '%s\n' "${task_id}"
  echo "${label}: ${task_id} succeeded" >&2
}

summary() {
  local app_id="$1"
  local project_id="$2"
  local cluster_id="$3"
  local preview_id="$4"
  local apply_id="$5"
  local redeploy_id="$6"
  local diagnostics_id="$7"
  local retire_id="$8"
  local apps transitions endpoints tasks audit
  apps="$(json_get /v1/serving-applications)"
  transitions="$(json_get "/v1/serving-applications/${app_id}/transitions")"
  endpoints="$(json_get /v1/endpoints)"
  tasks="$(json_get /v1/tasks)"
  audit="$(json_get /v1/audit-records)"
  printf '%s' "${apps}" >"${SMOKE_DIR}/apps.json"
  printf '%s' "${transitions}" >"${SMOKE_DIR}/transitions.json"
  printf '%s' "${endpoints}" >"${SMOKE_DIR}/endpoints.json"
  printf '%s' "${tasks}" >"${SMOKE_DIR}/tasks.json"
  printf '%s' "${audit}" >"${SMOKE_DIR}/audit.json"
  python3 - "$project_id" "$cluster_id" "$app_id" "$preview_id" "$apply_id" "$redeploy_id" "$diagnostics_id" "$retire_id" "$SMOKE_DIR" <<'PY'
import json
import sys
from pathlib import Path

project_id, cluster_id, app_id, preview_id, apply_id, redeploy_id, diagnostics_id, retire_id, smoke_dir = sys.argv[1:]
smoke_dir = Path(smoke_dir)
apps = json.loads((smoke_dir / "apps.json").read_text())
transitions = json.loads((smoke_dir / "transitions.json").read_text())
endpoints = json.loads((smoke_dir / "endpoints.json").read_text())
tasks = json.loads((smoke_dir / "tasks.json").read_text())
audit = json.loads((smoke_dir / "audit.json").read_text())
final_app = next(app for app in apps if app["id"] == app_id)
expected = {preview_id, apply_id, redeploy_id, diagnostics_id, retire_id}
actual_succeeded = {task["id"] for task in tasks if task["status"] == "succeeded"}
missing = expected - actual_succeeded
if missing:
    raise SystemExit(f"missing succeeded tasks: {sorted(missing)}")
if final_app["phase"] != "Retired":
    raise SystemExit(f"expected final phase Retired, got {final_app['phase']}")
if endpoints:
    raise SystemExit(f"expected no endpoints after retire, got {endpoints}")
summary = {
    "projectId": project_id,
    "clusterId": cluster_id,
    "appId": app_id,
    "taskIds": {
        "preview": preview_id,
        "apply": apply_id,
        "redeploy": redeploy_id,
        "diagnostics": diagnostics_id,
        "retire": retire_id,
    },
    "finalPhase": final_app["phase"],
    "transitionPhases": [f"{item.get('from') or ''}->{item['to']}" for item in transitions],
    "endpointCountAfterRetire": len(endpoints),
    "taskStatuses": {task["id"]: {"type": task["type"], "status": task["status"]} for task in tasks},
    "auditActions": [record["action"] for record in audit],
    "smokeDir": str(smoke_dir),
}
print(json.dumps(summary, indent=2, ensure_ascii=False))
PY
}

echo "Smoke dir: ${SMOKE_DIR}"
echo "API URL: ${API_URL}"

(
  cd "${SRC_DIR}"
  go run ./cmd/management-api -addr "${API_ADDR}" -data "${SMOKE_DIR}/management.json" >"${SMOKE_DIR}/management-api.log" 2>&1
) &
API_PID=$!

wait_until "Management API" 30 "curl -sf '${API_URL}/healthz'"

project="$(json_post /v1/projects '{"name":"platform"}')"
project_id="$(json_field id <<<"${project}")"
cluster="$(json_post /v1/clusters '{"name":"h200-a","prometheusUrl":"http://prometheus.local","grafanaUrl":"http://grafana.local"}')"
cluster_id="$(json_field id <<<"${cluster}")"

(
  cd "${SRC_DIR}"
  go run ./cmd/cluster-agent -management-url "${API_URL}" -cluster-id "${cluster_id}" -executor-mode fake -poll-interval 1s -heartbeat-interval 10s -capability dynamo=true,backend=vllm >"${SMOKE_DIR}/cluster-agent.log" 2>&1
) &
AGENT_PID=$!

wait_until "Cluster Agent registration" 30 "python3 - <<'PY'
import json
import urllib.request
with urllib.request.urlopen('${API_URL}/v1/agents', timeout=5) as response:
    raise SystemExit(0 if json.load(response) else 1)
PY"

artifact="$(json_post /v1/model-artifacts '{"family":"deepseek-v4","variant":"flash","revision":"rev1","pvcMountPath":"/home/dynamo/.cache/huggingface","pvcModelPath":"models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1","hostCachePath":"/data/cache/hub","quantization":"fp8"}')"
artifact_id="$(json_field id <<<"${artifact}")"
app_payload="$(python3 - <<PY
import json
print(json.dumps({
  "projectId": "${project_id}",
  "name": "DeepSeek V4 Flash",
  "model": {"family":"deepseek-v4","variant":"flash","artifactId":"${artifact_id}","quantization":"fp8"},
  "placement": {"clusterId":"${cluster_id}","namespace":"dynamo-system"},
  "runtime": {"backend":"vllm","topology":"pd-disagg","recipe":"deepseek-v4-flash-vllm-dgd-disagg"},
  "service": {"endpointName":"deepseek-v4-flash","protocol":"openai-compatible","exposure":"cluster-local"},
  "optimization": {"target":"throughput","profilingMode":"disabled"}
}))
PY
)"
app="$(json_post /v1/serving-applications "${app_payload}")"
app_id="$(json_field id <<<"${app}")"

preview_id="$(create_task_and_wait Preview "/v1/serving-applications/${app_id}/preview-task")"
apply_id="$(create_task_and_wait Apply "/v1/serving-applications/${app_id}/apply-task")"
redeploy_id="$(create_task_and_wait Redeploy "/v1/serving-applications/${app_id}/redeploy-task")"
diagnostics_id="$(create_task_and_wait Diagnostics "/v1/serving-applications/${app_id}/diagnostics-task")"
retire_id="$(create_task_and_wait Retire "/v1/serving-applications/${app_id}/retire-task")"

summary "${app_id}" "${project_id}" "${cluster_id}" "${preview_id}" "${apply_id}" "${redeploy_id}" "${diagnostics_id}" "${retire_id}"
echo "Smoke succeeded: app=${app_id} data=${SMOKE_DIR}/management.json"
