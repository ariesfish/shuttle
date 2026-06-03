> For clean Markdown content of this page, append .md to this URL. For the complete documentation index, see https://docs.nvidia.com/dynamo/llms.txt. For full content including API reference and SDK examples, see https://docs.nvidia.com/dynamo/llms-full.txt.

# Tool Calling

<p align="left">
  <a href="./README.zh-CN.md" hreflang="zh-CN"><img src="https://files.buildwithfern.com/dynamo.docs.buildwithfern.com/dynamo/2546607964ca3bb29badfbef8f4af73dee49dd6709242f791dfa79f7914b8f2e/pages-v1.2.0/assets/img/readme-zh-cn-link.svg" alt="简体中文" height="28" /></a>
</p>

Dynamo can connect models to external tools and services by parsing tool-call
syntax out of raw model output and surfacing it as OpenAI-compatible
`tool_calls` on the response. Tool calling is controlled by the `tool_choice`
and `tools` request parameters on the chat completions API.

There are two ways to parse tool calls in Dynamo, depending on whether the
parser lives in Dynamo's own registry or in an upstream engine frontend
(`vllm serve`, `sglang serve`, or `trtllm-serve`).

## Choose a parsing path

| Path | When to use | Page |
|------|-------------|------|
| **Dynamo** | Dynamo ships a framework-agnostic Rust parser for the model's tool-call format. Default path. | [Tool Call Parsing (Dynamo)](/dynamo/user-guides/tool-calling/tool-call-parsing-dynamo) |
| **Engine Fallback** | Use the framework's parser implementation (vLLM or SGLang today; TRTLLM in progress) for pre/post processing, including tool call and reasoning parsing - ensure consistency with framework behavior. | [Tool Call Parsing (Engine Fallback)](/dynamo/user-guides/tool-calling/tool-call-parsing-engine-fallback) |

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

## Troubleshooting

If a tool call comes back wrong, add `logprobs: true` to a single repro
request and share the response. See
[Troubleshooting Tool Calls](/dynamo/user-guides/tool-calling/troubleshooting-tool-calls) for what to capture and
include when reporting an issue.

## Optional: structural tags

You can optionally turn on **xgrammar structural tags** so guided decoding matches the parser's tool-call format at token granularity. See [Structural tag (guided decoding for tool calls)](structural-tag.md).

## See Also

- [Troubleshooting Tool Calls](/dynamo/user-guides/tool-calling/troubleshooting-tool-calls) -- capture raw model
  output with `logprobs` so tool-call issues can be localized.
- [Reasoning](/dynamo/user-guides/reasoning) -- separate `reasoning_content` from
  assistant content for chain-of-thought models. Several models need both a
  tool-call parser and a reasoning parser configured together.
- [Frontend Configuration Reference](/dynamo/components/frontend/configuration-reference) --
  full CLI flag reference.
- [Structural tag (guided decoding)](structural-tag.md) — optional xgrammar
  constraints aligned with Dynamo tool-call parsers.