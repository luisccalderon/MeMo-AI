# Local models

MeMo-AI can point Pi (`pi`) at any `provider/model` pair, including a
model served locally. This guide covers detection, declaring the server to Pi,
choosing a model that can actually drive the agent stack, and the context
limits that matter.

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
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
memo-ai install --yes --model vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

`--model` writes `defaultProvider` and `defaultModel` in
`~/.pi/agent/settings.json`; the reference is split at the first `/`. It does
**not** declare the server. Pi only lists a model whose provider is known, so
Ollama, LM Studio, vLLM and other OpenAI-compatible servers need an entry in
`~/.pi/agent/models.json` (see the examples below). llama.cpp is built in: use
Pi's `/llama` command. Open `/model` in Pi to check what it can use.

## What the main session needs

The main session runs the orchestrator. It must reliably:

1. Call tools (read, grep, edit, shell, memory).
2. Follow a structured prompt without drifting.
3. Hold the system prompt plus several files of context.
4. Emit valid tool arguments, especially for edits and memory writes.

Tool calling is the hard requirement. A model that cannot emit tool calls can
still chat in Pi, but it cannot run the MeMo-AI workflow.

### Practical minimums

| Use | Suggested capability |
| --- | --- |
| Main session (orchestrator) | Strong tool calling, 32k+ context, instruction following |
| Memory-only setup (`--no-orchestrator`) | Any tool-capable model |

Pi has no subagents, so everything runs in the main session. For 8 GB
machines, a 7–8B coder model with a 16k–32k context is a realistic starting
point for light work. For sustained orchestration, prefer 14B+ or a hosted
model.

## Context window

Pi sends the system prompt, the MeMo block, skill descriptions and conversation on
every turn. Budget for:

- Base system and MeMo-AI prompts: a few thousand tokens.
- Each skill description plus the invoked skill body.
- File reads and tool results — usually the largest consumer.
- Room for the reply and the next tool call.

If a model is configured with a 4k context, the stack will truncate and the
agent will look "forgetful" even with memory enabled. Raise the runtime's
context setting, or install with `--no-orchestrator` to shrink the prompt
surface.

## Ollama example

```sh
ollama pull qwen3:8b
```

Declare the provider in `~/.pi/agent/models.json`:

```json
{
  "providers": {
    "ollama": {
      "baseUrl": "http://localhost:11434/v1",
      "api": "openai-completions",
      "apiKey": "ollama",
      "models": [{ "id": "qwen3:8b" }]
    }
  }
}
```

The dummy key makes the model available to Pi; Ollama ignores it. Then:

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai doctor
```

To raise the context window, start Ollama with a larger `num_ctx`. Consult
your Ollama version's documentation; the exact flag has changed across
releases.

## LM Studio example

1. Load a tool-capable model in LM Studio and start the local server.
2. Confirm it responds: `curl http://127.0.0.1:1234/api/v1/models`.
3. Declare a `lm-studio` provider in `models.json` with
   `baseUrl: http://127.0.0.1:1234/v1` and `api: openai-completions`.
4. Install:

```sh
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
```

## vLLM example

```sh
vllm serve Qwen/Qwen3-Coder-30B-A3B-Instruct --port 8000
```

Declare the provider in `~/.pi/agent/models.json`:

```json
{
  "providers": {
    "vllm": {
      "baseUrl": "http://127.0.0.1:8000/v1",
      "api": "openai-completions",
      "apiKey": "vllm",
      "models": [{ "id": "Qwen/Qwen3-Coder-30B-A3B-Instruct" }]
    }
  }
}
```

```sh
memo-ai install --yes --model vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

Use the model id exactly as vLLM reports it under `/v1/models`. Pi reloads
`models.json` when `/model` opens. See Pi's models documentation for every
field, including `contextWindow` and `maxTokens`.

## Verify the choice

```sh
memo-ai agents            # detected runtimes, models and current config
memo-ai doctor            # installation and environment health
memo-ai doctor --json     # machine-readable output for scripts
```

Then in Pi, run `/memo` to see the active configuration and ask for a
small task that requires a tool call (for example, "read README.md and list
the headings"). If the model answers from memory instead of calling the read
tool, it is not ready for the orchestrator role.

## Fallbacks

- Keep a hosted model as the default and switch to the local one with `/model`
  for light work.
- If the local model lacks tool calling, install with `--no-orchestrator` and
  use it as a plain chat model; memory tools still require tool support.
- Reconfigure at any time: `memo-ai configure`.
