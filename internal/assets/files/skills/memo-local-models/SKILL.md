---
name: MeMo Local Models
description: Use Ollama, LM Studio or vLLM models safely, including tool-capability and context-limit checks.
metadata:
  opencode/autoinvoke: true
---

# MeMo Local Models

OpenCode discovers these local runtimes automatically:

| Runtime | Provider ID | Default endpoint |
| --- | --- | --- |
| Ollama | `ollama` | `http://127.0.0.1:11434` |
| LM Studio | `lmstudio` | `http://127.0.0.1:1234/v1` |
| vLLM | `vllm` | `http://127.0.0.1:8000/v1` |

Model references use `provider/model`:

```text
ollama/qwen3:8b
lmstudio/qwen2.5-coder-7b-instruct
vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

## Before assigning a local model to an agent

1. Confirm the model accepts tools. A model without tool support cannot drive the
   orchestrator or its subagents.
2. Confirm the context window. The orchestrator prompt plus skills and memory can be large;
   prefer at least 32k tokens for the primary agent.
3. Run a small smoke test: ask for one tool call and verify it executes.
4. If the model is weak at long instructions, keep the primary agent on a hosted model and use
   the local model for bounded writer or explorer work.

## Remote runtimes

Point OpenCode at another host with provider settings:

```jsonc
{
  "providers": {
    "ollama": {
      "settings": { "baseURL": "http://gpu-host:11434/v1" }
    }
  }
}
```

For LM Studio and vLLM, replace the provider ID and base URL accordingly.

## Failure modes

- **No models listed:** the runtime is not running, or `/api/tags` / `/v1/models` is not
  reachable from the OpenCode server process.
- **Tools silently ignored:** the model metadata does not advertise tool support. Pick a
  different model or use a hosted model for the orchestrator.
- **Context overflow:** lower the amount of memory injected, reduce enabled skills, or move
  the primary agent to a larger-context model.
