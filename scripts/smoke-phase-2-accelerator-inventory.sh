#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
DATA_FILE="${DATA_FILE:-$SRC_DIR/temp/phase2-smoke/management.json}"
ADDR="${ADDR:-127.0.0.1:18080}"
BASE_URL="http://$ADDR"
API_PID=""

cleanup() {
  if [[ -n "$API_PID" ]]; then
    kill "$API_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

mkdir -p "$(dirname "$DATA_FILE")"
rm -f "$DATA_FILE"

cd "$SRC_DIR"
go run ./cmd/management-api -addr "$ADDR" -data "$DATA_FILE" > temp/phase2-smoke/management-api.log 2>&1 &
API_PID="$!"

for _ in {1..50}; do
  if curl -fs "$BASE_URL/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done
curl -fsS "$BASE_URL/healthz" >/dev/null

json_field() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

post_json() {
  local path="$1"
  local body="$2"
  curl -fsS "$BASE_URL$path" -H 'Content-Type: application/json' -d "$body"
}

project_id=$(post_json /v1/projects '{"name":"platform"}' | json_field id)
cluster_id=$(post_json /v1/clusters '{"name":"h200-a","prometheusUrl":"http://prometheus.local","grafanaUrl":"http://grafana.local"}' | json_field id)
agent_id=$(post_json /v1/agents/register "{\"clusterId\":\"$cluster_id\",\"version\":\"fixture-agent\",\"capabilities\":{\"inventory\":\"fixture\"}}" | json_field id)

python3 - "$ROOT_DIR/src/testdata/accelerator-inventory/h200-compatible.json" "$agent_id" > temp/phase2-smoke/inventory.json <<'PY'
import json,sys
path, agent = sys.argv[1], sys.argv[2]
payload = json.load(open(path))
payload['agentId'] = agent
print(json.dumps(payload))
PY
curl -fsS "$BASE_URL/v1/clusters/$cluster_id/accelerator-inventory" -H 'Content-Type: application/json' --data-binary @temp/phase2-smoke/inventory.json >/dev/null
curl -fsS "$BASE_URL/v1/clusters/$cluster_id/accelerator-inventory" | grep -q 'NVIDIA H200 SXM'

pool_id=$(post_json /v1/accelerator-pools "{\"clusterId\":\"$cluster_id\",\"name\":\"h200\",\"nodeSelector\":{\"pool\":\"h200\"}}" | json_field id)
curl -fsS "$BASE_URL/v1/accelerator-pools/summaries?clusterId=$cluster_id" | grep -q 'NVIDIA H200 SXM'

artifact_id=$(post_json /v1/artifacts '{"family":"deepseek-v4","variant":"flash","revision":"rev1","pvcMountPath":"/models","pvcModelPath":"snapshot","quantization":"fp8"}' | json_field id)
app_body=$(cat <<JSON
{"projectId":"$project_id","name":"DeepSeek V4 Flash","model":{"family":"deepseek-v4","variant":"flash","artifactId":"$artifact_id","quantization":"fp8"},"placement":{"clusterId":"$cluster_id","acceleratorPoolId":"$pool_id","namespace":"dynamo-system"},"runtime":{"backend":"vllm","topology":"pd-disagg","recipe":"deepseek-v4-flash-vllm-dgd-disagg"},"service":{"endpointName":"deepseek-v4-flash","protocol":"openai-compatible","exposure":"cluster-local"},"optimization":{"target":"throughput","profilingMode":"disabled"}}
JSON
)
app_id=$(post_json /v1/apps "$app_body" | json_field id)

tuning_id=$(post_json /v1/tuning-records "{\"servingApplicationId\":\"$app_id\",\"benchmarkSummary\":{\"throughputTokensPerSecond\":1234},\"plannerSettings\":{\"prefillTp\":4},\"recommendations\":[\"keep h200 pool\"],\"reason\":\"phase2 smoke\"}" | json_field id)
curl -fsS "$BASE_URL/v1/tuning-records?servingApplicationId=$app_id" | grep -q "$tuning_id"
curl -fsS "$BASE_URL/v1/apps/$app_id/observability/entry-points" | grep -q 'accelerator-inventory'

bad_cluster_id=$(post_json /v1/clusters '{"name":"partial"}' | json_field id)
bad_agent_id=$(post_json /v1/agents/register "{\"clusterId\":\"$bad_cluster_id\",\"version\":\"fixture-agent\"}" | json_field id)
python3 - "$ROOT_DIR/src/testdata/accelerator-inventory/partial-missing-rdma.json" "$bad_agent_id" > temp/phase2-smoke/bad-inventory.json <<'PY'
import json,sys
path, agent = sys.argv[1], sys.argv[2]
payload = json.load(open(path))
payload['agentId'] = agent
print(json.dumps(payload))
PY
curl -fsS "$BASE_URL/v1/clusters/$bad_cluster_id/accelerator-inventory" -H 'Content-Type: application/json' --data-binary @temp/phase2-smoke/bad-inventory.json >/dev/null
bad_app_body="${app_body//$cluster_id/$bad_cluster_id}"
if curl -fsS "$BASE_URL/v1/apps" -H 'Content-Type: application/json' -d "$bad_app_body" > temp/phase2-smoke/bad-app.json 2>/dev/null; then
  echo "expected incompatible Serving Application validation to fail" >&2
  exit 1
fi

echo "Phase 2 accelerator inventory smoke passed: app=$app_id tuning=$tuning_id"
