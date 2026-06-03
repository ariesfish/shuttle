> For clean Markdown content of this page, append .md to this URL. For the complete documentation index, see https://docs.nvidia.com/dynamo/llms.txt. For full content including API reference and SDK examples, see https://docs.nvidia.com/dynamo/llms-full.txt.

# Reasoning Parsing (Engine Fallback)

When Dynamo's registry does not list a reasoning parser for your model, fall
back to the upstream engine's parser via a **chat-processor swap**, which
keeps frontend tokenization and KV routing.

For Dynamo-native parsers, see [Reasoning Parsing (Dynamo)](/dynamo/user-guides/reasoning/reasoning-parsing-dynamo). For
the equivalent tool-call fallback, see
[Tool Call Parsing (Engine Fallback)](/dynamo/user-guides/tool-calling/tool-call-parsing-engine-fallback).

<Warning>
**Known Issue:** Engine-fallback reasoning parsing does not currently work
with [disaggregated serving](/dynamo/user-guides/disaggregated-serving)
(support coming soon). Use the [Dynamo-native reasoning parser](/dynamo/user-guides/reasoning/reasoning-parsing-dynamo)
for disaggregated deployments today.
</Warning>

## Configurations

| | Frontend flags | Worker flags | KV routing | Notes |
|---|---|---|---|---|
| **vLLM chat processor** | `--dyn-chat-processor vllm --reasoning-parser <name>` | *(none)* | Yes | Parsing runs in vLLM's Python preprocessor. See [vLLM Chat Processor](/dynamo/backends/v-llm/frontend-processor-fallback). |
| **SGLang chat processor** | `--dyn-chat-processor sglang --reasoning-parser <name>` | *(none)* | Yes | Parsing runs in SGLang's Python preprocessor. See [SGLang Chat Processor](/dynamo/backends/sg-lang/frontend-processor-fallback). |
| **TRTLLM chat processor** | *(work in progress)* | *(work in progress)* | -- | Engine-fallback support for TRTLLM is in progress. Use the [Dynamo-native reasoning parser](/dynamo/user-guides/reasoning/reasoning-parsing-dynamo) for TRTLLM today. |

<Note>
`--dyn-reasoning-parser` selects the **Dynamo-native** parser path, while
`--reasoning-parser` selects the **engine fallback** (vLLM or SGLang)
parser path. The accepted values for each flag come from a different
registry and may differ slightly based on the definitions from each
framework (e.g., vLLM's `nemotron_v3` vs Dynamo's `nemotron3`).
</Note>

## Examples

```bash
# vLLM chat processor
python -m dynamo.vllm ...
python -m dynamo.frontend --dyn-chat-processor vllm --reasoning-parser deepseek_r1

# SGLang chat processor
python -m dynamo.sglang ...
python -m dynamo.frontend --dyn-chat-processor sglang --reasoning-parser kimi_k25
```

## See Also

- [Reasoning Parsing (Dynamo)](/dynamo/user-guides/reasoning/reasoning-parsing-dynamo) -- Dynamo-native parsers and common pairings
- [Tool Call Parsing (Engine Fallback)](/dynamo/user-guides/tool-calling/tool-call-parsing-engine-fallback) -- Equivalent fallback for tool-call parsers
- [vLLM Chat Processor](/dynamo/backends/v-llm/frontend-processor-fallback) -- vLLM chat-processor details
- [SGLang Chat Processor](/dynamo/backends/sg-lang/frontend-processor-fallback) -- SGLang chat-processor details
- [Frontend Configuration Reference](/dynamo/components/frontend/configuration-reference) -- Full CLI flag reference