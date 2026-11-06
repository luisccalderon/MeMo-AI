# MeMo-AI

**Orquestación con memoria persistente para [OpenCode](https://opencode.ai).**

MeMo-AI instala y configura un stack pequeño y opinionado: un agente primario
orquestador, subagentes opcionales, un servidor de memoria persistente
([Engram](https://github.com/Gentleman-Programming/engram)), comandos de barra,
skills y un plugin TUI que dibuja el wordmark de MeMo-AI.

Es un único binario Go sin dependencias en tiempo de ejecución. Solo edita los
archivos de OpenCode que le pertenecen, guarda una copia de tu configuración
antes de cambiarla y puede eliminar todo lo que instaló.

> [Read in English](README.md)

---

## Por qué

Los agentes de código olvidan. Vuelven a preguntar decisiones, releen los
mismos archivos y pierden el hilo después de una compactación. MeMo-AI hace que
la memoria y la evidencia sean parte del flujo de trabajo, no un añadido:

- **Recuerda antes de preguntar** — recupera contexto de Engram (`mem_context`, `mem_search`).
- **Orquesta a propósito** — trabaja en línea por defecto; delega solo por una razón concreta.
- **Demuestra antes de declarar terminado** — cada cierre incluye un comando y su resultado.
- **Persiste lo importante** — decisiones, arreglos y handoffs duraderos se guardan en Engram.

## Requisitos

- **OpenCode v2** en `PATH` (o `OPENCODE_BIN`).
- **Go 1.23+** solo si compilas desde el código.
- **Engram** es opcional. MeMo-AI puede detectarlo o descargar la última versión verificada por ti.
- Un runtime de modelos locales (Ollama, LM Studio, vLLM) es opcional. Cualquier `provider/model` que OpenCode acepte funciona.

## Instalación

### Desde el código

```sh
git clone https://github.com/luis/memo-ai.git
cd memo-ai
make build
./bin/memo-ai install
```

### Con el script de instalación

```sh
./scripts/install.sh          # compila este checkout e instala en ~/.local/bin
```

### Con Go

```sh
go install github.com/luis/memo-ai/cmd/memo-ai@latest
memo-ai install
```

## Inicio rápido

```sh
memo-ai                 # asistente interactivo (español por defecto)
memo-ai --lang en       # asistente interactivo en inglés
memo-ai install --yes   # acepta los valores recomendados
memo-ai doctor          # verifica la instalación
memo-ai agents          # lista agentes, modelos y configuración actual
```

Ejemplos no interactivos:

```sh
# Usa un modelo local de Ollama y activa el stack completo de subagentes
memo-ai install --yes --model ollama/qwen3:8b

# Mantenlo en línea: sin subagentes ni plugin TUI
memo-ai install --yes --no-subagents --no-plugin

# Usa el agente build integrado como primario, solo con memoria
memo-ai install --yes --agent build --model anthropic/claude-sonnet-4-5 --no-subagents
```

Después abre OpenCode y ejecuta `/memo`.

## Qué se instala

Todo vive bajo el directorio global de configuración de OpenCode
(`${XDG_CONFIG_HOME:-~/.config}/opencode`):

| Ruta | Propósito |
| --- | --- |
| `agents/memo-*.md` | Orquestador más subagentes explorer, writer, reviewer y curator |
| `skills/memo-*/SKILL.md` | Procedimientos de memoria, delegación, handoff y modelos locales |
| `commands/memo*.md` | `/memo`, `/memo-recall`, `/memo-remember`, `/memo-handoff`, `/memo-review`, `/memo-agents` |
| `plugins/memo-ai/` | Wordmark TUI, badge `MEMO`, comando `MeMo-AI · About` y herramienta `memo_status` |
| `memo-ai/state.json` | Manifiesto de archivos instalados y sus hashes SHA-256 |
| `memo-ai/backups/` | Copias de configuración y archivos creadas antes de escribir |

`opencode.json` (u `opencode.jsonc`) se fusiona, no se reemplaza:

- `default_agent` y `model` apuntan a tus elecciones.
- `agents.<primario>.permissions` recibe una política ordenada de subagentes.
- `mcp.servers.engram` se añade cuando Engram está activo.
- `experimental.subagent_depth` se fija en `1` cuando hay subagentes.

Las claves existentes, los comentarios fuera de los límites de JSONC y la
configuración no relacionada se conservan. Los comentarios JSONC se pierden al
reescribir el archivo y siempre se mantiene una copia de seguridad.

## Agentes

| Agente | Modo | Rol |
| --- | --- | --- |
| `memo-orchestrator` | primario | Por defecto con memoria primero: recupera, enruta, verifica, persiste. |
| `memo-explorer` | subagente | Mapa de código de solo lectura con evidencia `path:line`. |
| `memo-writer` | subagente | Una unidad de implementación acotada con superficie de edición disjunta. |
| `memo-reviewer` | subagente | Revisión independiente de solo lectura, ordenada por severidad. |
| `memo-curator` | subagente | Consolida hallazgos duraderos en Engram. |

Los subagentes son opcionales y están limitados a profundidad 1. Cuando se
desactivan, el agente primario recibe `deny subagent *` y hace todo el trabajo
en línea.

## Comandos y skills

| Comando | Qué hace |
| --- | --- |
| `/memo` | Muestra la configuración activa de memoria y orquestación. |
| `/memo-recall <tema>` | Busca en la memoria del proyecto y responde con evidencia. |
| `/memo-remember` | Guarda una decisión, arreglo o descubrimiento duradero. |
| `/memo-handoff` | Escribe un resumen reanudable antes de pausar o compactar. |
| `/memo-review` | Ejecuta una revisión independiente de solo lectura. |
| `/memo-agents` | Explica los agentes instalados y el enrutamiento. |

OpenCode invoca las skills automáticamente cuando su descripción coincide con
la tarea: `memo-memory`, `memo-delegation`, `memo-handoff`, `memo-local-models`.

## Plugin TUI

El plugin incluido se instala en `plugins/memo-ai/` y se descubre
automáticamente. Proporciona:

- El wordmark de MeMo-AI en `home.footer` (paleta Vesper).
- Un badge de estado `MEMO`.
- Un comando de paleta `MeMo-AI · About` y el comando `/memo-about`.
- Una herramienta `memo_status` del lado del servidor para los agentes.

Desactívalo con `--no-plugin`. Reconfigura con `memo-ai configure` en cualquier
momento.

## Memoria con Engram

MeMo-AI configura Engram como servidor MCP local:

```json
{
  "mcp": {
    "servers": {
      "engram": {
        "type": "local",
        "command": ["/usr/bin/engram", "mcp", "--tools=agent"]
      }
    }
  }
}
```

Si falta Engram:

```sh
memo-ai engram status
memo-ai engram install   # descarga la última release y verifica su checksum
```

Consulta [docs/engram.md](docs/engram.md) para el protocolo de memoria y la
solución de problemas.

## Modelos locales

MeMo-AI detecta Ollama (`:11434`), LM Studio (`:1234`) y vLLM (`:8000`) y ofrece
sus modelos durante el asistente. También puedes escribir manualmente cualquier
`provider/model` que OpenCode soporte.

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lmstudio/qwen2.5-coder-7b
```

Consulta [docs/local-models.md](docs/local-models.md) para capacidad de
herramientas y ventanas de contexto.

## Seguridad

- Solo se eliminan los archivos registrados en `memo-ai/state.json`.
- Los archivos que modificaste se omiten salvo que uses `--force`.
- La configuración se restaura desde la copia previa al instalar al desinstalar.
- `--dry-run` muestra cada escritura planeada sin tocar el disco.
- Ningún dato sale de tu máquina salvo la descarga opcional de la release de Engram.

```sh
memo-ai uninstall          # restaura la configuración y elimina lo instalado
memo-ai uninstall --force  # también elimina archivos que modificaste
```

## Referencia de comandos

```text
memo-ai [comando] [flags]

Comandos
  install       Instala o reconfigura el stack (interactivo)
  configure     Reconfigura agentes, modelo y subagentes
  doctor        Diagnóstico de solo lectura (--json soportado)
  agents        Lista agentes, modelos locales y configuración actual
  engram        Estado o instalación de Engram
  uninstall     Elimina solo lo que instaló MeMo-AI
  version       Muestra la versión

Flags globales
  --lang es|en
  --yes
  --dry-run
  --agent <id>
  --model <provider/model>
  --subagent-model <provider/model>
  --no-subagents
  --no-engram
  --no-plugin
  --install-engram
  --force
  --restore
  --json
```

## Desarrollo

```sh
make test     # go test ./...
make vet      # go vet ./...
make fmt      # gofmt -w
make build    # bin/memo-ai
make install  # instala en PREFIX/bin
```

Estructura del proyecto:

```text
cmd/memo-ai/            punto de entrada
internal/cli/           parsing de flags, asistente, comandos
internal/install/       escritura de assets, fusión de config, política de subagentes
internal/uninstall/     eliminación segura y restauración de config
internal/config/        carga/guardado JSONC y helpers anidados
internal/detect/        detección de OpenCode, Engram y modelos locales
internal/doctor/        diagnósticos
internal/engram/        descarga, checksum e instalación de releases
internal/assets/files/  agentes, skills, comandos y plugin embebidos
packages/opencode-memo/ paquete fuente del plugin
```

Consulta [docs/architecture.md](docs/architecture.md) para el ciclo de
instalación y el modelo de datos.

## Licencia

[MIT](LICENSE)
