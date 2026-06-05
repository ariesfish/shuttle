# Observability Dashboards

本目录保存 Dynamo 相关 Grafana dashboard ConfigMap。dashboard 通过 Prometheus 查询指标，覆盖 Dynamo 应用、disaggregated serving、Planner、Operator、GPU 和 Kubernetes 基础设施。

## 文件说明

| 文件 | Dashboard | 主要用途 |
|---|---|---|
| `grafana-dynamo-dashboard-configmap.yaml` | Dynamo Dashboard | 查看 serving 主链路：请求吞吐、成功率、TTFT、ITL、E2E latency、token 分布、KV routing、worker 处理耗时。 |
| `grafana-disagg-dashboard-configmap.yaml` | Dynamo Disaggregated Analysis | 查看 prefill/decode 分离部署：frontend 体验、prefill/ decode worker、KV cache、GPU、NVLink、CPU、数据传输。 |
| `grafana-planner-dashboard-configmap.yaml` | Dynamo Planner Dashboard | 查看 Planner/autoscaling：当前 replica、观察流量、预测流量、推荐 replica、校正因子、GPU hours。 |
| `grafana-operator-dashboard-configmap.yaml` | Dynamo Operator | 查看 operator 自身健康：CRD reconcile、webhook admission、资源状态和成功率。 |

安装后可通过 Grafana dashboard sidecar 自动加载这些 ConfigMap。常用应用方式：

```bash
kubectl apply -n monitoring -f deployment/observability/grafana-dynamo-dashboard-configmap.yaml
kubectl apply -n monitoring -f deployment/observability/grafana-disagg-dashboard-configmap.yaml
kubectl apply -n monitoring -f deployment/observability/grafana-planner-dashboard-configmap.yaml
kubectl apply -n monitoring -f deployment/observability/grafana-operator-dashboard-configmap.yaml
```

## 指标来源总览

| 指标前缀/指标 | 上报模块 | 含义 | 常见 scrape 来源 |
|---|---|---|---|
| `dynamo_frontend_*` | Dynamo Frontend | frontend 侧请求、延迟、token、队列、KV-router bookkeeping | frontend pod HTTP `/metrics` |
| `dynamo_component_*` | Dynamo worker/component | worker 请求、错误、耗时、KV cache、字节吞吐、router component metrics | worker pod system metrics endpoint |
| `dynamo_component_router_*` | LocalRouter / standalone router | router 侧请求、TTFT、ITL、ISL/OSL、KV hit rate | router/component metrics endpoint |
| `dynamo_router_overhead_*` | Frontend 内 KV router | routing 阶段耗时：block hashing、indexer matching、sequence hashing、scheduling、total | frontend Prometheus registry |
| `dynamo_frontend_worker_*` | Frontend KV-router bookkeeping | 每个 worker active decode blocks / prefill tokens | frontend Prometheus registry |
| `dynamo_planner_*` | Dynamo Planner | replica、observed traffic、predicted traffic、推荐扩缩容、校正因子、GPU hours | Planner 的 `PLANNER_PROMETHEUS_PORT` |
| `dynamo_operator_*` | Dynamo Operator | controller reconcile、webhook、资源 inventory | operator metrics ServiceMonitor |
| `DCGM_FI_*` | NVIDIA dcgm-exporter | GPU compute、memory、NVLink 等硬件遥测 | GPU Operator / dcgm-exporter ServiceMonitor |
| `container_cpu_usage_seconds_total` | kubelet/cAdvisor | container CPU 使用 | kubelet/cAdvisor |
| `node_cpu_seconds_total` | node-exporter | node CPU 使用 | node-exporter |
| `kube_pod_status_phase` | kube-state-metrics | pod phase，用于过滤 Running pod | kube-state-metrics |

## Dynamo Dashboard

`grafana-dynamo-dashboard-configmap.yaml` 是主服务 dashboard，主要看一次 LLM serving 请求从 frontend 到 worker 的完整链路。

| 面板/数据 | 看什么 | 指标 | 上报模块 |
|---|---|---|---|
| Request Success Rate / Total Requests / Request Outcome Breakdown | frontend 收到的请求总量、成功率、终态错误类型，如 `cancelled`、`validation`、`not_found`、`overload`、`internal` | `dynamo_frontend_requests_total` | Dynamo Frontend |
| Frontend RPS | frontend 每秒请求数 | `rate(dynamo_frontend_requests_total)` | Dynamo Frontend |
| Average TTFT / TTFT p50/p90/p99 | Time To First Token，从请求开始到首 token 返回 | `dynamo_frontend_time_to_first_token_seconds_*` | Dynamo Frontend |
| Average ITL / ITL p50/p90/p99 | Inter-token latency，流式输出 token 间隔 | `dynamo_frontend_inter_token_latency_seconds_*` | Dynamo Frontend |
| Average E2E Latency / E2E Request Latency | 端到端请求耗时，从进入 frontend 到响应完成 | `dynamo_frontend_request_duration_seconds_*` | Dynamo Frontend |
| Input Tokens / Output Tokens | 时间范围内输入/输出 token 总量 | `dynamo_frontend_input_sequence_tokens_sum`, `dynamo_frontend_output_sequence_tokens_sum` | Dynamo Frontend |
| ISL Distribution / Output Size Distribution | 输入长度和输出长度分布，p50/p90/p99/avg | `dynamo_frontend_input_sequence_tokens_*`, `dynamo_frontend_output_sequence_tokens_*` | Dynamo Frontend |
| Cached Tokens | prefix cache 命中的 token 分布 | `dynamo_frontend_cached_tokens_*` | Dynamo Frontend，来自 backend usage accounting 归一化结果 |
| Inflight vs Queued Requests | 当前正在处理请求数和排队请求数；Queued 高通常表示 worker 饱和 | `dynamo_frontend_inflight_requests`, `dynamo_frontend_queued_requests` | Dynamo Frontend |
| Per-Worker Active Decode Blocks | 每个 worker 当前 active decode KV blocks | `dynamo_frontend_worker_active_decode_blocks` | Frontend KV-router bookkeeping |
| Per-Worker Active Prefill Tokens | 每个 worker 当前 active prefill tokens；KV routing 模式下有意义 | `dynamo_frontend_worker_active_prefill_tokens` | Frontend KV-router bookkeeping |
| KV Hit Rate Distribution | 路由时预测的 KV cache hit rate，通常为 `overlap_blocks / input_sequence_blocks` | `dynamo_component_router_kv_hit_rate_*` | Router / LocalRouter component metrics |
| Routing Overhead Breakdown | KV router 各阶段开销：block hashing、indexer matching、seq hashing、scheduling、total | `dynamo_router_overhead_*` | Frontend 内 KV router |
| KV Events Applied Breakdown | KV cache index 事件应用状态，如 `stored`、`removed`、`cleared` 及 apply error | `dynamo_component_kv_cache_events_applied` | Worker/component KV cache event/index 逻辑 |
| Worker Request Breakdown Per Worker | 每个 worker 的请求数、取消数、错误类型 | `dynamo_component_requests_total`, `dynamo_component_cancellation_total`, `dynamo_component_errors_total` | Dynamo worker/component |
| Worker Request Duration Per Worker | worker work handler 内部处理耗时，p50/p90/p99/avg | `dynamo_component_request_duration_seconds_*` | Dynamo worker/component |
| Component Throughput bytes/sec | `generate` endpoint 的 request/response 字节吞吐 | `dynamo_component_request_bytes_total`, `dynamo_component_response_bytes_total` | Dynamo worker/component |

## Disaggregated Analysis Dashboard

`grafana-disagg-dashboard-configmap.yaml` 面向 prefill/decode 分离部署。它把 frontend 体验、prefill worker、decode worker、KV cache 和 GPU/NVLink 资源放在一起看。

| 面板/数据 | 看什么 | 指标 | 上报模块 |
|---|---|---|---|
| Frontend Requests / Sec | frontend RPS | `dynamo_frontend_requests_total` | Dynamo Frontend |
| Frontend Avg Time to First Token | 首 token 延迟，包含 prefill queue delay、GPU compute、NIXL transfer | `dynamo_frontend_time_to_first_token_seconds_*` | Dynamo Frontend |
| Frontend Avg Request Duration | frontend 端到端请求耗时 | `dynamo_frontend_request_duration_seconds_*` | Dynamo Frontend |
| Frontend Avg Inter-Token Latency | token 间隔延迟 | `dynamo_frontend_inter_token_latency_seconds_*` | Dynamo Frontend |
| Frontend Avg Input/Output Sequence Length | 平均输入/输出 token 长度 | `dynamo_frontend_input_sequence_tokens_*`, `dynamo_frontend_output_sequence_tokens_*` | Dynamo Frontend |
| Frontend Queued Requests | frontend 排队请求数；诊断 prefill worker 瓶颈的关键指标 | `dynamo_frontend_queued_requests` | Dynamo Frontend |
| Prefill Worker Processing Time | prefill worker 处理耗时，包含 NIXL KV transfer | `dynamo_component_request_duration_seconds_*{dynamo_component="prefill"}` | Prefill worker component |
| Prefill Worker Throughput | prefill worker RPS | `dynamo_component_requests_total{dynamo_component="prefill"}` | Prefill worker component |
| Component Latency - Prefill vs Decode | prefill 和 decode/backend component 处理延迟对比 | `dynamo_component_request_duration_seconds_*{dynamo_component="prefill"}`, `dynamo_component_request_duration_seconds_*{dynamo_component="backend"}` | Prefill / decode worker |
| Decode Worker - Request Throughput | decode/backend worker RPS | `dynamo_component_requests_total{dynamo_component="backend"}` | Decode/backend worker |
| Decode Worker - Avg Request Duration | decode/backend worker 平均处理耗时 | `dynamo_component_request_duration_seconds_*{dynamo_component="backend"}` | Decode/backend worker |
| KV Cache Utilization | decode worker KV cache GPU memory 使用率；高于 90% 通常说明 decode 接近容量上限 | `dynamo_component_gpu_cache_usage_percent` | Worker backend，通常 decode worker 暴露 |
| KV Cache Blocks Total | decode worker 可用 KV cache block 容量 | `dynamo_component_total_blocks` | Worker backend |
| GPU Compute Utilization | GPU 计算利用率 | `DCGM_FI_DEV_GPU_UTIL` | NVIDIA dcgm-exporter |
| GPU Memory Bandwidth | GPU memory copy 利用率；NIXL/KV transfer 会体现为峰值 | `DCGM_FI_DEV_MEM_COPY_UTIL` | NVIDIA dcgm-exporter |
| GPU Memory Used | GPU FB memory 使用量 | `DCGM_FI_DEV_FB_USED / 1024` | NVIDIA dcgm-exporter |
| Worker CPU Usage | worker pod `main` container CPU 使用 | `container_cpu_usage_seconds_total` | kubelet/cAdvisor |
| Node CPU Utilization | 节点 CPU 使用率 | `node_cpu_seconds_total` | node-exporter |
| Worker Request Throughput | 所有 worker component 的 `generate` endpoint RPS | `dynamo_component_requests_total` | Dynamo worker/component |
| Worker Data Transfer | worker request/response bytes/sec | `dynamo_component_request_bytes_total`, `dynamo_component_response_bytes_total` | Dynamo worker/component |
| NVLink Bandwidth | NVLink TX/RX GB/s，包含 intra-pod TP 通信和可能的 KV transfer | `DCGM_FI_PROF_NVLINK_TX_BYTES`, `DCGM_FI_PROF_NVLINK_RX_BYTES` | NVIDIA dcgm-exporter profiling metrics |

该 dashboard 中很多 PromQL 都乘了：

```promql
* on(pod, namespace) group_left() kube_pod_status_phase{phase="Running"}
```

这不是业务指标，而是用 kube-state-metrics 的 pod phase 过滤，只显示 Running pod。

## Planner Dashboard

`grafana-planner-dashboard-configmap.yaml` 用于看 Planner/autoscaling 行为。Planner 自己暴露 `dynamo_planner_*`；这些指标既包含当前状态，也包含从 Prometheus 或 event plane 读入后计算出的 observed/predicted 结果。

| 面板/数据 | 看什么 | 指标 | 上报模块 |
|---|---|---|---|
| Prefill Workers | 当前 prefill worker 数 | `dynamo_planner_num_prefill_replicas` | Dynamo Planner |
| Decode Workers | 当前 decode worker 数 | `dynamo_planner_num_decode_replicas` | Dynamo Planner |
| Cumulative GPU Hours | planner 启动以来累计 GPU hours | `dynamo_planner_gpu_hours` | Dynamo Planner |
| Worker Count History | prefill/decode worker 数历史 | `dynamo_planner_num_prefill_replicas`, `dynamo_planner_num_decode_replicas` | Dynamo Planner |
| Observed Latency (TTFT & ITL) | planner 观察到的 TTFT/ITL | `dynamo_planner_observed_ttft_ms`, `dynamo_planner_observed_itl_ms` | Dynamo Planner |
| Observed Request Rate & Duration | planner 观察到的 RPS 和请求耗时 | `dynamo_planner_observed_requests_per_second`, `dynamo_planner_observed_request_duration_seconds` | Dynamo Planner |
| Observed Sequence Lengths (ISL & OSL) | 观察到的输入/输出 token 长度 | `dynamo_planner_observed_input_sequence_tokens`, `dynamo_planner_observed_output_sequence_tokens` | Dynamo Planner |
| Predicted Request Rate | 预测 RPS | `dynamo_planner_predicted_requests_per_second` | Dynamo Planner throughput prediction |
| Predicted Sequence Lengths (ISL & OSL) | 预测输入/输出 token 长度 | `dynamo_planner_predicted_input_sequence_tokens`, `dynamo_planner_predicted_output_sequence_tokens` | Dynamo Planner throughput prediction |
| Predicted Replica Counts | 推荐 prefill/decode replica 数 | `dynamo_planner_predicted_num_prefill_replicas`, `dynamo_planner_predicted_num_decode_replicas` | Dynamo Planner |
| Prefill Correction Factor | prefill 校正因子，约等于 observed TTFT / expected TTFT | `dynamo_planner_p_correction_factor` | Dynamo Planner |
| Decode Correction Factor | decode 校正因子，约等于 observed ITL / expected ITL | `dynamo_planner_d_correction_factor` | Dynamo Planner |
| Correction Factor History | prefill/decode 校正因子历史；接近 1 表示预测更准 | `dynamo_planner_p_correction_factor`, `dynamo_planner_d_correction_factor` | Dynamo Planner |

Planner 的输入来源取决于配置：

| Planner 配置/模式 | 读取来源 |
|---|---|
| `throughput_metrics_source: frontend` | 从 Prometheus 读取 `dynamo_frontend_*`，适合单 DGD 部署。 |
| `throughput_metrics_source: router` | 从 Prometheus 读取 `dynamo_component_router_*`，适合 GlobalPlanner / multi-pool，本地 pool Planner 不应读共享 public Frontend。 |
| load-based scaling | 读取 Dynamo event plane 的 ForwardPassMetrics，包括 per-iteration wall time、scheduled tokens、queue 状态等；不需要 router `/metrics` scraping。 |

## Operator Dashboard

`grafana-operator-dashboard-configmap.yaml` 看 Kubernetes operator 自身，不看推理性能。

| 面板/数据 | 看什么 | 指标 | 上报模块 |
|---|---|---|---|
| Reconciliation Rate | 各 CRD reconcile 频率，按 `resource_type` 和 `result` 分组 | `dynamo_operator_reconcile_total` | Dynamo Operator controller |
| Reconciliation Duration (P95) | reconcile P95 耗时 | `dynamo_operator_reconcile_duration_seconds_bucket` | Dynamo Operator controller |
| Reconciliation Errors | reconcile 错误速率，按 `error_type` 分组 | `dynamo_operator_reconcile_errors_total` | Dynamo Operator controller |
| Webhook Request Rate | admission webhook 请求速率，按 resource 和 operation 分组 | `dynamo_operator_webhook_requests_total` | Dynamo Operator webhook |
| Webhook Duration (P95) | webhook validation P95 耗时 | `dynamo_operator_webhook_duration_seconds_bucket` | Dynamo Operator webhook |
| Webhook Denials | webhook 拒绝速率和原因 | `dynamo_operator_webhook_denials_total` | Dynamo Operator webhook |
| Resource Inventory by State | operator 管理资源按 namespace、type、status 的库存 | `dynamo_operator_resources_total` | Dynamo Operator controller |
| Resource Count by State | 当前资源状态计数 | `dynamo_operator_resources_total` | Dynamo Operator controller |
| Reconciliation Success Rate | reconcile 成功率 | `dynamo_operator_reconcile_total{result="success"}` / total | Dynamo Operator controller |
| Webhook Admission Success Rate | webhook allowed 比例 | `dynamo_operator_webhook_requests_total{result="allowed"}` / total | Dynamo Operator webhook |

Operator 指标通过 Helm chart 默认创建的 ServiceMonitor 采集；它和应用指标不同，应用 frontend/worker 通常用 PodMonitor。

## 排障阅读顺序

1. **用户体验差**：先看 `TTFT`、`ITL`、`E2E Request Latency`、`Request Outcome Breakdown`。
2. **吞吐是否够**：看 `Frontend RPS`、`Queued Requests`、`Inflight Requests`。
3. **prefill 是否瓶颈**：看 `Frontend Queued Requests`、`Prefill Worker Processing Time`、prefill GPU utilization。
4. **decode 是否瓶颈**：看 `ITL`、`Decode Worker Duration`、`KV Cache Utilization`。
5. **KV/NIXL/通信问题**：看 `GPU Memory Bandwidth`、`NVLink Bandwidth`、`KV Hit Rate`、`Routing Overhead`。
6. **autoscaling 是否合理**：看 Planner 的 `observed_*`、`predicted_*`、`p/d_correction_factor`。
7. **部署/operator 问题**：看 Operator dashboard 的 reconcile errors、webhook denials、resource inventory。

## 相关文档

- `docs/dynamo/deployment/metrics.md`
- `docs/dynamo/deployment/operator-metrics.md`
- `docs/dynamo/components/planner.md`
- `docs/dynamo/components/router-configuration-and-tuning.md`
