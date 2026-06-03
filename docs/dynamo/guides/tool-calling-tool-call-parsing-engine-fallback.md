> For clean Markdown content of this page, append .md to this URL. For the complete documentation index, see https://docs.nvidia.com/dynamo/llms.txt. For full content including API reference and SDK examples, see https://docs.nvidia.com/dynamo/llms-full.txt.

# Tool Call Parsing (Engine Fallback)

When Dynamo's registry does not list a tool-call parser for your model, fall
back to the upstream engine's parser via a **chat-processor swap**, which
keeps frontend tokenization and KV routing.

For Dynamo-native parsers, see [Tool Call Parsing (Dynamo)](/dynamo/user-guides/tool-calling/tool-call-parsing-dynamo). For
the equivalent reasoning fallback, see
[Reasoning Parsing (Engine Fallback)](/dynamo/user-guides/reasoning/reasoning-parsing-engine-fallback).

<Warning>
**Known Issue:** Engine-fallback tool call parsing does not currently work
with [disaggregated serving](/dynamo/user-guides/disaggregated-serving)
(support coming soon). Use the [Dynamo-native tool call parser](/dynamo/user-guides/tool-calling/tool-call-parsing-dynamo)
for disaggregated deployments today.
</Warning>

## Configurations

| | Frontend flags | Worker flags | KV routing | Notes |
|---|---|---|---|---|
| **vLLM chat processor** | `--dyn-chat-processor vllm --tool-call-parser <name>` | *(none)* | Yes | Parsing runs in vLLM's Python preprocessor. See [vLLM Chat Processor](/dynamo/backends/v-llm/frontend-processor-fallback). |
| **SGLang chat processor** | `--dyn-chat-processor sglang --tool-call-parser <name>` | *(none)* | Yes | Parsing runs in SGLang's Python preprocessor. See [SGLang Chat Processor](/dynamo/backends/sg-lang/frontend-processor-fallback). |
| **TRTLLM chat processor** | *(work in progress)* | *(work in progress)* | -- | Engine-fallback support for TRTLLM is in progress. Use the [Dynamo-native tool call parser](/dynamo/user-guides/tool-calling/tool-call-parsing-dynamo) for TRTLLM today. |

<Note>
`--dyn-tool-call-parser` selects the **Dynamo-native** parser path, while
`--tool-call-parser` selects the **engine fallback** (vLLM or SGLang)
parser path. The accepted values for each flag come from a different
registry and may differ slightly based on the definitions from each
framework (e.g., SGLang's `deepseekv3` vs Dynamo's `deepseek_v3`).
</Note>

## Examples

```bash
# vLLM chat processor
python -m dynamo.vllm ...
python -m dynamo.frontend --dyn-chat-processor vllm --tool-call-parser hermes

# SGLang chat processor
python -m dynamo.sglang ...
python -m dynamo.frontend --dyn-chat-processor sglang --tool-call-parser kimi_k2
```

<Tip>
If a tool call comes back wrong, add `"logprobs": true` to a single repro
request and share the response. See
[Troubleshooting Tool Calls](/dynamo/user-guides/tool-calling/troubleshooting-tool-calls) for what to capture and
include when reporting an issue.
</Tip>

## See Also

- [Troubleshooting Tool Calls](/dynamo/user-guides/tool-calling/troubleshooting-tool-calls) -- capture raw model output with `logprobs` so tool-call issues can be localized
- [Tool Call Parsing (Dynamo)](/dynamo/user-guides/tool-calling/tool-call-parsing-dynamo) -- Dynamo-native parsers and request examples
- [Reasoning Parsing (Engine Fallback)](/dynamo/user-guides/reasoning/reasoning-parsing-engine-fallback) -- Equivalent fallback for reasoning
- [vLLM Chat Processor](/dynamo/backends/v-llm/frontend-processor-fallback) -- vLLM chat-processor details
- [SGLang Chat Processor](/dynamo/backends/sg-lang/frontend-processor-fallback) -- SGLang chat-processor details
- [Frontend Configuration Reference](/dynamo/components/frontend/configuration-reference) -- Full CLI flag reference