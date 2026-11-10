---
name: memo-local-models
description: Use Ollama, LM Studio, vLLM or llama.cpp models safely with Pi, including tool-capability and context-limit checks.
---

# MeMo Local Models

Pi connects to llama.cpp through its built-in router (`/llama`). Ollama, LM Studio, vLLM and any
other OpenAI-compatible server need a provider entry in `~/.pi/agent/models.json`.

| Runtime | Provider ID | Default endpoint |
| --- | --- | --- |
| Ollama | `ollama` | `http://127.0.0.1:11434/v1` |
| LM Studio | `lm-studio` | `http://127.0.0.1:1234/v1` |
| vLLM | your choice, for example `vllm` | `http://127.0.0.1:8000/v1` |

Model references use `provider/model`:

```text
ollama/qwen3:8b
lm-studio/qwen2.5-coder-7b-instruct
vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

Open `/model` to see which models Pi can use. A model only appears when its provider resolves
credentials; for a local server a dummy `apiKey` is enough.

## Before assigning a local model

1. Confirm the model accepts tools. A model without tool support cannot drive the orchestrator.
2. Confirm the context window. The orchestrator prompt plus skills and memory can be large;
   prefer at least 32k tokens.
3. Run a small smoke test: ask for one tool call and verify it executes.
4. If the model is weak at long instructions, use a hosted model for the main session.

## Assigning the model

```sh
memo-ai configure --model ollama/qwen3:8b
```

This writes `defaultProvider` and `defaultModel` in `~/.pi/agent/settings.json` and leaves every
other setting alone. You can also pick a model with `/model` and press `Ctrl+S` to save it as the
default.

## Declaring a local server

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

Save it as `~/.pi/agent/models.json`. Pi reloads the file when `/model` opens.

## Failure modes

- **Model not listed:** the provider is missing from `models.json`, or the runtime is not running
  or not reachable from the machine that runs Pi.
- **Tools silently ignored:** the model does not support tool calls. Pick a different model or
  use a hosted one.
- **Context overflow:** reduce enabled skills, lower the memory you inject, or move to a
  larger-context model.
