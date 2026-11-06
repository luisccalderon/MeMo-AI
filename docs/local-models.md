# Local models

MeMo-AI can point OpenCode at any `provider/model` pair, including a model
served locally. This guide covers detection, choosing a model that can
actually drive the agent stack, and the context limits that matter.

## Detection

During install and in `memo-ai agents`, MeMo-AI probes three runtimes with a
1.5 s timeout:

| Provider | Default endpoint | Env override |
| --- | --- | --- |
| Ollama | `http://127.0.0.1:11434/api/tags` | `OLLAMA_HOST` |
| LM Studio | `http://127.0.0.1:1234/api/v1/models` | `LMSTUDIO_HOST` |
| vLLM | `http://127.0.0.1:8000/v1/models` | `VLLM_HOST` |

Detected models appear in the wizard and in `memo-ai agents`. Detection is
advisory: you can always type a model manually.

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lmstudio/qwen2.5-coder-7b
memo-ai install --yes --model vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

If OpenCode uses a different provider id for your runtime, use that id. The
value is written verbatim to `model` and to the agent definitions.

## What the primary agent needs

The primary agent is an orchestrator. It must reliably:

1. Call tools (read, grep, edit, shell, memory).
2. Follow a structured prompt without drifting.
3. Hold the system prompt plus several files of context.
4. Emit valid tool arguments, especially for edits and memory writes.

Tool calling is the hard requirement. A model that cannot emit tool calls can
still chat in OpenCode, but it cannot run the MeMo-AI workflow.

### Practical minimums

| Use | Suggested capability |
| --- | --- |
| Primary orchestrator | Strong tool calling, 32k+ context, instruction following |
| Subagents | Same tool calling, smaller context is usually fine |
| Memory-only setup | Any tool-capable model; run `--no-subagents` |

For 8 GB machines, a 7–8B coder model with a 16k–32k context is a realistic
starting point. For orchestration with subagents, prefer 14B+ or a hosted
model.

## Context window

OpenCode sends the system prompt, agent prompt, skills and conversation on
every turn. Budget for:

- Base system and MeMo-AI prompts: a few thousand tokens.
- Each enabled skill description plus the invoked skill body.
- File reads and tool results — usually the largest consumer.
- Room for the reply and the next tool call.

If a model is configured with a 4k context, the stack will truncate and the
agent will look "forgetful" even with memory enabled. Raise the runtime's
context setting or pick a smaller prompt surface (disable subagents, keep
skills concise).

## Ollama example

```sh
ollama pull qwen3:8b
memo-ai install --yes --model ollama/qwen3:8b
memo-ai doctor
```

To raise the context window, set it in the model definition or start Ollama
with a larger `num_ctx`. Consult your Ollama version's documentation; the
exact flag has changed across releases.

## LM Studio example

1. Load a tool-capable model in LM Studio and start the local server.
2. Confirm it responds: `curl http://127.0.0.1:1234/api/v1/models`.
3. Install:

```sh
memo-ai install --yes --model lmstudio/qwen2.5-coder-7b
```

## vLLM example

```sh
vllm serve Qwen/Qwen3-Coder-30B-A3B-Instruct --port 8000
memo-ai install --yes --model vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

Use the model id exactly as vLLM reports it under `/v1/models`.

## Verify the choice

```sh
memo-ai agents            # detected runtimes, models and current config
memo-ai doctor            # installation and environment health
memo-ai doctor --json     # machine-readable output for scripts
```

Then in OpenCode, run `/memo` to see the active configuration and ask for a
small task that requires a tool call (for example, "read README.md and list
the headings"). If the model answers from memory instead of calling the read
tool, it is not ready for the orchestrator role.

## Fallbacks

- Keep a hosted model configured as the primary and use a local model for
  subagents via `--subagent-model`.
- If the local model lacks tool calling, run with `--no-subagents` and use it
  as a plain chat model; memory tools still require tool support.
- Reconfigure at any time: `memo-ai configure`.
