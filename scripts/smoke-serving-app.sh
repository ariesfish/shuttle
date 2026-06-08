#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="${ROOT_DIR}/src"
API_ADDR="${API_ADDR:-:18080}"
API_URL="${API_URL:-http://localhost:${API_ADDR#:}}"
PROM_LOCAL_PORT="${PROM_LOCAL_PORT:-19090}"
PROM_NAMESPACE="${PROM_NAMESPACE:-monitoring}"
PROM_SERVICE="${PROM_SERVICE:-svc/kube-prometheus-stack-prometheus}"
NAMESPACE="${NAMESPACE:-dynamo-system}"
APP_NAME="${APP_NAME:-DeepSeek V4 Flash SGLang Smoke}"
ENDPOINT_NAME="${ENDPOINT_NAME:-deepseek-v4-flash-sglang-smoke}"
RECIPE_ID="${RECIPE_ID:-deepseek-v4-flash-sglang-dgd-disagg}"
BACKEND="${BACKEND:-sglang}"
TOPOLOGY="${TOPOLOGY:-pd-disagg}"
MODEL_FAMILY="${MODEL_FAMILY:-deepseek-v4}"
MODEL_VARIANT="${MODEL_VARIANT:-flash}"
QUANTIZATION="${QUANTIZATION:-fp8}"
MODEL_REVISION="${MODEL_REVISION:-6976c7ff}"
PVC_MOUNT_PATH="${PVC_MOUNT_PATH:-/home/dynamo/.cache/huggingface}"
PVC_MODEL_PATH="${PVC_MODEL_PATH:-models--deepseek-ai--DeepSeek-V4-Flash/snapshots/6976c7ff1b30a1b2cb7805021b8ba4684041f136}"
HOST_CACHE_PATH="${HOST_CACHE_PATH:-/data/cache/hub}"
KEEP_RESOURCES="${KEEP_RESOURCES:-false}"
APPLY_TIMEOUT_SECONDS="${APPLY_TIMEOUT_SECONDS:-900}"
TASK_TIMEOUT_SECONDS="${TASK_TIMEOUT_SECONDS:-180}"
SMOKE_DIR="${SMOKE_DIR:-/tmp/inference-smoke-$(date +%s)}"

mkdir -p "${SMOKE_DIR}"

delete_workload() {
  kubectl delete dynamographdeployment -n "${NAMESPACE}" "${ENDPOINT_NAME}" --ignore-not-found --wait=true --timeout=120s >/dev/null 2>&1 || true
  kubectl delete dynamocomponentdeployment -n "${NAMESPACE}" -l "inference.aistudio.dev/serving-application=${ENDPOINT_NAME}" --ignore-not-found --wait=true --timeout=60s >/dev/null 2>&1 || true
  kubectl delete pod,deploy,rs,svc -n "${NAMESPACE}" -l "inference.aistudio.dev/serving-application=${ENDPOINT_NAME}" --ignore-not-found --wait=true --timeout=60s >/dev/null 2>&1 || true
}

cleanup() {
  local exit_code=$?
  if [[ "${KEEP_RESOURCES}" != "true" ]]; then
    delete_workload || true
  fi
  for pid_file in agent.pid management.pid prometheus-pf.pid; do
    if [[ -f "${SMOKE_DIR}/${pid_file}" ]]; then
      kill "$(cat "${SMOKE_DIR}/${pid_file}")" >/dev/null 2>&1 || true
    fi
  done
  exit "${exit_code}"
}
trap cleanup EXIT

json_post() {
  local path="$1"
  local payload="$2"
  curl -sf -H 'Content-Type: application/json' -d "${payload}" "${API_URL}${path}"
}

wait_task() {
  local app_id="$1"
  local task_type="$2"
  local timeout_seconds="$3"
  local elapsed=0
  while (( elapsed < timeout_seconds )); do
    local task status
    task="$(curl -sf "${API_URL}/v1/tasks" | jq -c --arg app "${app_id}" --arg type "${task_type}" '[.[] | select(.type==$type and .payload.servingApplicationId==$app)] | last')"
    status="$(printf '%s' "${task}" | jq -r '.status // empty')"
    if [[ "${status}" == "succeeded" || "${status}" == "failed" ]]; then
      printf '%s\n' "${task}"
      [[ "${status}" == "succeeded" ]]
      return
    fi
    sleep 1
    elapsed=$((elapsed + 1))
    if (( elapsed % 30 == 0 )); then
      echo "waiting ${task_type} ${elapsed}s status=${status}" >&2
      kubectl get dynamographdeployment,dynamocomponentdeployment,pod,svc -n "${NAMESPACE}" 2>/dev/null | grep "${ENDPOINT_NAME}" >&2 || true
    fi
  done
  echo "timed out waiting for ${task_type}" >&2
  return 1
}

echo "Smoke dir: ${SMOKE_DIR}"
echo "Checking existing resources for ${ENDPOINT_NAME}"
if kubectl get dynamographdeployment,dynamocomponentdeployment,deploy,rs,pod,svc -n "${NAMESPACE}" 2>/dev/null | grep -q "${ENDPOINT_NAME}"; then
  echo "Existing resources found; deleting before smoke"
  delete_workload
fi

cd "${SRC_DIR}"
(go run ./cmd/management-api -addr "${API_ADDR}" -data "${SMOKE_DIR}/management.json" >"${SMOKE_DIR}/management.log" 2>&1 & echo $! >"${SMOKE_DIR}/management.pid")
(kubectl -n "${PROM_NAMESPACE}" port-forward "${PROM_SERVICE}" "${PROM_LOCAL_PORT}:9090" >"${SMOKE_DIR}/prometheus-pf.log" 2>&1 & echo $! >"${SMOKE_DIR}/prometheus-pf.pid")

for _ in $(seq 1 30); do
  curl -sf "${API_URL}/healthz" >/dev/null && break
  sleep 1
done
curl -sf "${API_URL}/healthz" >/dev/null

project_id="$(json_post /v1/projects '{"name":"smoke"}' | jq -r .id)"
cluster_id="$(json_post /v1/clusters "$(jq -n --arg name "$(kubectl config current-context)" --arg prom "http://localhost:${PROM_LOCAL_PORT}" '{name:$name,prometheusUrl:$prom}')" | jq -r .id)"
artifact_id="$(json_post /v1/model-artifacts "$(jq -n --arg family "${MODEL_FAMILY}" --arg variant "${MODEL_VARIANT}" --arg revision "${MODEL_REVISION}" --arg mount "${PVC_MOUNT_PATH}" --arg model "${PVC_MODEL_PATH}" --arg host "${HOST_CACHE_PATH}" --arg quant "${QUANTIZATION}" '{family:$family,variant:$variant,revision:$revision,pvcMountPath:$mount,pvcModelPath:$model,hostCachePath:$host,quantization:$quant}')" | jq -r .id)"
app_payload="$(jq -n --arg project "${project_id}" --arg cluster "${cluster_id}" --arg artifact "${artifact_id}" --arg appName "${APP_NAME}" --arg namespace "${NAMESPACE}" --arg backend "${BACKEND}" --arg topology "${TOPOLOGY}" --arg recipe "${RECIPE_ID}" --arg endpoint "${ENDPOINT_NAME}" --arg family "${MODEL_FAMILY}" --arg variant "${MODEL_VARIANT}" --arg quant "${QUANTIZATION}" '{projectId:$project,name:$appName,model:{family:$family,variant:$variant,artifactId:$artifact,quantization:$quant},placement:{clusterId:$cluster,namespace:$namespace},runtime:{backend:$backend,topology:$topology,recipe:$recipe},service:{endpointName:$endpoint,protocol:"openai-compatible",exposure:"cluster-local"},optimization:{target:"throughput",profilingMode:"disabled"}}')"
app_id="$(json_post /v1/serving-applications "${app_payload}" | jq -r .id)"

(go run ./cmd/cluster-agent -management-url "${API_URL}" -cluster-id "${cluster_id}" -capability "dynamo=true,backend=${BACKEND}" -poll-interval 1s -heartbeat-interval 10s >"${SMOKE_DIR}/agent.log" 2>&1 & echo $! >"${SMOKE_DIR}/agent.pid")

curl -sf -X POST "${API_URL}/v1/serving-applications/${app_id}/preview-task" >/dev/null
preview_task="$(wait_task "${app_id}" PreviewDeploymentDiff "${TASK_TIMEOUT_SECONDS}")"
echo "Preview: $(printf '%s' "${preview_task}" | jq -c '{id,status,error,result:{mode:.result.mode,manifestCount:.result.manifestCount}}')"

curl -sf -X POST "${API_URL}/v1/serving-applications/${app_id}/apply-task" >/dev/null
apply_task="$(wait_task "${app_id}" ApplyDeployment "${APPLY_TIMEOUT_SECONDS}")"
echo "Apply: $(printf '%s' "${apply_task}" | jq -c '{id,status,error,result:{mode:.result.mode,phase:.result.phase,message:.result.message,endpointUrl:.result.endpointUrl}}')"

curl -sf -X POST "${API_URL}/v1/serving-applications/${app_id}/diagnostics-task" >/dev/null
diagnostics_task="$(wait_task "${app_id}" FetchDiagnostics "${TASK_TIMEOUT_SECONDS}")"
echo "Diagnostics: $(printf '%s' "${diagnostics_task}" | jq -c '{id,status,error,sections:[.result.sections[] | {name,error,bytes:(.output|length)}]}')"

echo "Transitions:"
curl -sf "${API_URL}/v1/serving-applications/${app_id}/transitions" | jq '[.[] | {from,to,actor,taskId,reason}]'
echo "Endpoint:"
curl -sf "${API_URL}/v1/endpoints" | jq --arg app "${app_id}" '[.[] | select(.servingApplicationId==$app)]'
echo "Observability summary:"
curl -sf "${API_URL}/v1/serving-applications/${app_id}/observability/summary" | jq '{results:[.results[] | {name,value,error}]}'

if [[ "${KEEP_RESOURCES}" != "true" ]]; then
  curl -sf -X POST "${API_URL}/v1/serving-applications/${app_id}/retire-task" >/dev/null
  retire_task="$(wait_task "${app_id}" RetireDeployment "${TASK_TIMEOUT_SECONDS}")"
  echo "Retire: $(printf '%s' "${retire_task}" | jq -c '{id,status,error,result:{mode:.result.mode,message:.result.message}}')"
fi

echo "Smoke succeeded: app=${app_id} data=${SMOKE_DIR}/management.json"
