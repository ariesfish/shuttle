> For clean Markdown content of this page, append .md to this URL. For the complete documentation index, see https://docs.nvidia.com/dynamo/llms.txt. For full content including API reference and SDK examples, see https://docs.nvidia.com/dynamo/llms-full.txt.

# Reasoning

Some models emit reasoning or thinking content separately from their final
response. Dynamo can split that output into `reasoning_content` and normal
assistant content by configuring a reasoning parser.

There are two ways to parse reasoning in Dynamo, depending on whether the
parser lives in Dynamo's own registry or in an upstream engine frontend
(`vllm serve`, `sglang serve`, or `trtllm-serve`).

## Choose a parsing path

| Path | When to use | Page |
|------|-------------|------|
| **Dynamo** | Dynamo ships a framework-agnostic Rust parser for the model's reasoning format. Default path. | [Reasoning Parsing (Dynamo)](/dynamo/user-guides/reasoning/reasoning-parsing-dynamo) |
| **Engine Fallback** | Use the framework's parser implementation (vLLM or SGLang today; TRTLLM in progress) for pre/post processing, including tool call and reasoning parsing - ensure consistency with framework behavior. | [Reasoning Parsing (Engine Fallback)](/dynamo/user-guides/reasoning/reasoning-parsing-engine-fallback) |

Start with the Dynamo path. Fall back to the engine path only when Dynamo's
registry does not list a parser for your model.

## Why Dynamo implements tool-call and reasoning parsers

In `vllm serve`, `sglang serve`, and `trtllm-serve`, tool-call parsing and
reasoning parsing happens in the engine's frontend server, with subtle
behavioral differences across each. For performance purposes, Dynamo orchestrates
routing and tokenization, passing tokens directly to each LLM engine and circumventing
each engine's frontend OpenAI API server to avoid duplicate work per request.

Dynamo therefore implements tool-call parsing and reasoning parsing in its
frontend as a framework-agnostic Rust layer. This gives Dynamo one tested
OpenAI-compatible contract across vLLM, SGLang, TRTLLM, and other workers,
while keeping the serving hot path highly concurrent and scalable, avoiding
Python GIL bottlenecks.

## See Also

- [Tool Calling](/dynamo/user-guides/tool-calling) -- parse tool calls out of model
  output. Several models need both a reasoning parser and a tool-call parser
  configured together.
- [Frontend Configuration Reference](/dynamo/components/frontend/configuration-reference) --
  full CLI flag reference.