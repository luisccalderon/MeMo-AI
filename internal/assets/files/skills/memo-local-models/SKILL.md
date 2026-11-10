---
name: memo-local-models
description: Use Ollama, LM Studio or vLLM models safely with oh-my-pi, including tool-capability and context-limit checks.
---

# MeMo Local Models

oh-my-pi (omp) discovers Ollama and LM Studio without configuration. vLLM and any other
OpenAI-compatible server need a provider entry in `~/.omp/agent/models.yml`.

| Runtime | Provider ID | Default endpoint | Override |
| --- | --- | --- | --- |
| Ollama | `ollama` | `http://127.0.0.1:11434` | `OLLAMA_BASE_URL`, then `OLLAMA_HOST` |
| LM Studio | `lm-studio` | `http://127.0.0.1:1234/v1` | `LM_STUDIO_BASE_URL` |
| vLLM | your choice, for example `vllm` | `http://127.0.0.1:8000/v1` | `models.yml` |

Model references use `provider/model`, with an optional thinking suffix such as `:high`:

```text
ollama/qwen3:8b
lm-studio/qwen2.5-coder-7b-instruct
vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct
```

Check what omp can see with `omp models <provider>`.

## Before assigning a local model

1. Confirm the model accepts tools. A model without tool support cannot drive the
   orchestrator or its subagents.
2. Confirm the context window. The orchestrator prompt plus skills and memory can be large;
   prefer at least 32k tokens for the default role.
3. Run a small smoke test: ask for one tool call and verify it executes.
4. If the model is weak at long instructions, keep the default role on a hosted model and pin
   the local model on bounded subagents (`memo-ai configure --subagent-model <provider/model>`).

## Assigning models

```sh
omp config set modelRoles.default ollama/qwen3:8b   # main session
omp config set modelRoles.smol ollama/qwen3:4b      # cheap, fast role
```

`memo-ai install --model <provider/model>` runs the first command for you.

## vLLM or another OpenAI-compatible server

```yaml
# ~/.omp/agent/models.yml
providers:
  vllm:
    baseUrl: http://127.0.0.1:8000/v1
    api: openai-completions
    auth: none
    models:
      - id: Qwen/Qwen3-Coder-30B-A3B-Instruct
        name: Qwen3 Coder 30B
        contextWindow: 65536
        maxTokens: 8192
```

Keep this file valid YAML: omp quarantines an invalid config file and refuses to start.

## Failure modes

- **No models listed:** the runtime is not running, or its endpoint is not reachable from the
  machine that runs omp.
- **Tools silently ignored:** the model does not support tool calls. Pick a different model or
  use a hosted model for the default role.
- **Context overflow:** reduce enabled skills, lower the memory you inject, or move the default
  role to a larger-context model.
