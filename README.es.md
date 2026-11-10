# MeMo-AI

**Tu agente de código: reúne tus cambios, tus tareas y tu flujo de ingeniería.**

MeMo-AI convierte [oh-my-pi](https://github.com/can1357/oh-my-pi) (`omp`) en un agente de
ingeniería con memoria. Instala un prompt orquestador, subagentes opcionales, un servidor de
memoria persistente ([Engram](https://github.com/Gentleman-Programming/engram)), comandos de
barra, skills y una extensión pequeña.

Es un único binario Go sin dependencias en tiempo de ejecución. Solo edita lo que le pertenece,
guarda una copia de los archivos compartidos antes de cambiarlos y puede eliminar todo lo que
instaló.

> [Read in English](README.md)

---

## Camino rápido

```sh
git clone https://github.com/luisccalderon/MeMo-AI.git
cd MeMo-AI
make build
./bin/memo-ai install      # asistente interactivo
./bin/memo-ai doctor       # verificación
```

Después abre `omp` y ejecuta `/memo`.

## Por qué

Los agentes de código olvidan. Vuelven a preguntar decisiones, releen los mismos archivos y
pierden el hilo después de una compactación. MeMo-AI mantiene tres cosas en un mismo ciclo:

| | Qué hace MeMo-AI |
| --- | --- |
| **Cambios** | Sabe qué hay en el árbol de trabajo; cada cambio de código cierra con evidencia y una línea `Risk:`. `/memo-changes` lo resume. |
| **Tareas** | El trabajo pequeño vive en la conversación; el grande tiene un documento de feature. `/memo-tasks` informa en qué punto está. |
| **Flujo** | Un ciclo fijo (autorizar, recordar, explorar, clasificar, implementar, verificar, persistir), con delegación solo por un trigger concreto. |

La memoria lo sostiene: decisiones, arreglos y handoffs se guardan en Engram y se recuperan
antes de que el agente vuelva a preguntarte.

## Requisitos

- **oh-my-pi** (`omp`) en `PATH`, o `OMP_BIN` apuntando al binario.
- **Go 1.23+** solo si compilas desde el código.
- **Engram** es opcional. MeMo-AI puede detectarlo o descargar la última versión verificada.
- Un runtime de modelos locales (Ollama, LM Studio, vLLM) es opcional. Cualquier
  `provider/model` que omp acepte funciona.

## Instalación

| Método | Comando |
| --- | --- |
| Desde el código | `make build && ./bin/memo-ai install` |
| Script de instalación | `./scripts/install.sh` (compila este checkout en `~/.local/bin`) |
| Go | `go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest && memo-ai install` |

Ejemplos no interactivos:

```sh
# Modelo local de Ollama y el stack completo de subagentes
memo-ai install --yes --model ollama/qwen3:8b

# Todo en línea: sin subagentes ni extensión
memo-ai install --yes --no-subagents --no-extension

# Solo memoria, skills y comandos; sin tocar el prompt de sistema de omp
memo-ai install --yes --no-orchestrator --no-subagents

# Ver cada escritura planificada sin tocar el disco
memo-ai install --yes --dry-run
```

El asistente usa español por defecto; `--lang en` lo cambia a inglés.

## Qué se instala

Todo vive en el directorio de agente de omp: `~/.omp/agent` por defecto. Se respetan
`PI_CODING_AGENT_DIR`, `PI_CONFIG_DIR` y `OMP_PROFILE` igual que lo hace omp.

| Ruta | Propósito | Propiedad |
| --- | --- | --- |
| `APPEND_SYSTEM.md` | El orquestador MeMo, añadido al prompt de sistema de omp | Solo el bloque entre los marcadores `memo-ai:begin` / `memo-ai:end` |
| `agents/memo-*.md` | Subagentes explorer, writer, reviewer y curator | Archivo completo |
| `skills/memo-*/SKILL.md` | Procedimientos de memoria, delegación, handoff y modelos locales | Archivo completo |
| `commands/memo*.md` | Los comandos `/memo*` | Archivo completo |
| `extensions/memo-ai/index.ts` | Insignia de estado, herramienta `memo_status`, `/memo-about`, aviso de compactación | Archivo completo |
| `mcp.json` | Engram como servidor MCP stdio | Solo `mcpServers.engram` |
| `memo-ai/state.json` | Manifiesto de archivos instalados y sus hashes SHA-256 | Archivo completo |
| `memo-ai/backups/` | Copias tomadas antes de cambiar un archivo compartido | Directorio completo |

Tu propio texto en `APPEND_SYSTEM.md` y tus otros servidores MCP se conservan.

El modelo es lo único que MeMo-AI no escribe a mano. `--model` ejecuta
`omp config set modelRoles.default <provider/model>`, de modo que omp valida el valor contra su
propio esquema. Si no encuentra `omp`, MeMo-AI te muestra ese comando.

## Agentes

omp tiene una sesión principal y agentes de tarea que puede lanzar. MeMo-AI sigue esa forma:

| Pieza | Tipo | Rol |
| --- | --- | --- |
| Orquestador MeMo | bloque del prompt de sistema | Memoria primero: recuperar, enrutar, verificar, persistir. |
| `memo-explorer` | agente de tarea | Mapa de código de solo lectura con evidencia `path:line`. |
| `memo-writer` | agente de tarea | Una unidad de implementación acotada con superficie de edición propia. |
| `memo-reviewer` | agente de tarea | Revisión independiente de solo lectura, ordenada por severidad. |
| `memo-curator` | agente de tarea | Consolida hallazgos duraderos en Engram. |

Los subagentes son opcionales. Un subagente se activa instalando su archivo y se desactiva
quitándolo; los agentes MeMo declaran `spawns: ""`, así que no pueden lanzar otros agentes. Con
`--no-subagents` el orquestador hace todo en línea. `--subagent-model` fija un modelo en el
frontmatter de los agentes.

## Cómo funciona el orquestador

El orquestador ejecuta un ciclo fijo en cada petición:

1. **Autorizar** — solo lectura salvo que la petición autorice un cambio.
2. **Recordar** — recuperar el contexto del proyecto desde Engram antes de preguntar.
3. **Explorar** — leer primero el código existente.
4. **Clasificar** — las tareas pequeñas se hacen en línea; las grandes tienen un documento de feature.
5. **Implementar** — una unidad verificable a la vez, con test primero cuando aplica.
6. **Verificar** — informar `comando → resultado`.
7. **Persistir** — guardar el conocimiento duradero y cerrar las sesiones sustanciales.

La delegación es opcional y por trigger: presupuesto de evidencia, mapeo, writers en paralelo,
límite de contexto, trabajo por acción, verificación de alto riesgo y curación de memoria. Cada
handoff declara objetivo, superficies de edición permitidas, restricciones, invariantes,
criterio de aceptación y formato de retorno.

### Política de commits

- **Supervisado (por defecto):** sin commit, push, publicación, merge ni release salvo que lo pidas.
- **Autónomo:** cuando declaras explícitamente una ejecución automática, headless o sin
  supervisión, el orquestador puede crear commits por unidad de trabajo en una rama de feature
  con mensajes Conventional Commit. Push, pull requests, merge, release y operaciones
  destructivas de git siguen requiriendo una petición explícita.

## Comandos y skills

| Comando | Qué hace |
| --- | --- |
| `/memo` | Muestra la configuración activa de memoria y orquestación. |
| `/memo-changes [ref]` | Resume los cambios del árbol de trabajo con su evidencia y riesgo. |
| `/memo-tasks [feature]` | Muestra las tareas registradas, su estado y el siguiente paso. |
| `/memo-recall <tema>` | Busca en la memoria del proyecto y responde con evidencia. |
| `/memo-remember <hecho>` | Guarda una decisión, arreglo o descubrimiento duradero. |
| `/memo-handoff` | Escribe un resumen reanudable antes de pausar o compactar. |
| `/memo-review [foco]` | Ejecuta una revisión independiente de solo lectura. |
| `/memo-agents` | Explica los agentes instalados y el enrutamiento. |
| `/memo-about` | Muestra la configuración sin gastar un turno del modelo (extensión). |

Las skills aparecen en el prompt de sistema de omp y se cargan cuando su descripción coincide
con la tarea: `memo-memory`, `memo-delegation`, `memo-handoff`, `memo-local-models`.

## Extensión

omp descubre `extensions/memo-ai/index.ts` automáticamente. Aporta:

- Una entrada de estado `MEMO` (`MEMO · memory` cuando Engram está configurado).
- Una herramienta de solo lectura `memo_status`, para que el agente lea su configuración en vez
  de suponerla.
- El comando `/memo-about`.
- Un aviso para ejecutar `/memo-handoff` cuando el contexto está por compactarse.

Se desactiva con `--no-extension`. Reconfigura cuando quieras con `memo-ai configure`.

## Memoria con Engram

MeMo-AI añade Engram a `~/.omp/agent/mcp.json`:

```json
{
  "mcpServers": {
    "engram": {
      "type": "stdio",
      "command": "/usr/bin/engram",
      "args": ["mcp", "--tools=agent"]
    }
  }
}
```

Si falta Engram:

```sh
memo-ai engram status
memo-ai engram install   # descarga la última versión y verifica su checksum
```

Consulta [docs/engram.md](docs/engram.md) para el protocolo de memoria y la resolución de
problemas.

## Modelos locales

MeMo-AI detecta Ollama (`:11434`), LM Studio (`:1234`) y vLLM (`:8000`) y ofrece sus modelos en
el asistente. omp descubre Ollama y LM Studio por su cuenta; vLLM necesita un proveedor en
`models.yml`.

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
```

Consulta [docs/local-models.md](docs/local-models.md) para la guía de capacidades y ventana de
contexto.

## Seguridad

- Solo se eliminan los archivos registrados en `memo-ai/state.json`.
- Los archivos que modificaste se omiten salvo que uses `--force`.
- `APPEND_SYSTEM.md` y `mcp.json` se editan en el sitio: solo se tocan el bloque MeMo y la
  entrada de Engram, y antes se toma una copia.
- `--dry-run` muestra cada escritura planificada sin tocar el disco.
- El orquestador solo hace commits en una ejecución autónoma declarada; push, pull requests,
  merge y release siempre requieren una petición explícita.
- Ningún dato sale de tu máquina salvo la descarga opcional de Engram.

```sh
memo-ai uninstall            # quita los archivos instalados, el bloque del prompt y la entrada de Engram
memo-ai uninstall --force    # también quita los archivos que modificaste
```

La desinstalación deja el modelo por defecto de omp como está y te indica cómo restablecerlo.

## Referencia de comandos

```text
memo-ai [comando] [flags]

Comandos
  install       Instala o reconfigura el stack (interactivo)
  configure     Reconfigura el orquestador, el modelo y los subagentes
  doctor        Diagnóstico de solo lectura (admite --json)
  agents        Lista agentes, modelos locales y la configuración actual
  engram        Estado o instalación de Engram
  uninstall     Elimina solo lo que MeMo-AI instaló
  version       Muestra la versión

Flags globales
  --lang es|en
  --yes
  --dry-run
  --model <provider/model>
  --subagent-model <provider/model>
  --no-orchestrator
  --no-subagents
  --no-engram
  --no-extension
  --install-engram
  --force
  --restore
  --json
```

## Si vienes de la versión para OpenCode

Las versiones anteriores apuntaban a OpenCode. Esa instalación vive en `~/.config/opencode` y
esta versión no la toca. Elimínala con el binario anterior (`memo-ai uninstall`) antes de
actualizar y luego ejecuta `memo-ai install` de nuevo.

| Antes (OpenCode) | Ahora (oh-my-pi) |
| --- | --- |
| Agente primario `memo-orchestrator`, `--agent` | Bloque en `APPEND_SYSTEM.md`, `--no-orchestrator` |
| Fusión en `opencode.json` | Fusión en `mcp.json` + `omp config set` |
| Reglas de permisos de subagentes | Archivos de agente con `spawns: ""` |
| Plugin TUI, `--no-plugin` | Extensión, `--no-extension` |
| `lmstudio/<modelo>` | `lm-studio/<modelo>` |

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
internal/cli/           flags, asistente y comandos
internal/install/       escritura de assets, bloque del prompt, fusión de mcp.json, rol de modelo
internal/uninstall/     eliminación segura y limpieza
internal/promptblock/   bloque gestionado dentro de APPEND_SYSTEM.md
internal/omp/           resolución del directorio de agente de omp
internal/config/        carga/guardado JSON/JSONC y utilidades anidadas
internal/detect/        detección de omp, Engram y modelos locales
internal/doctor/        diagnósticos
internal/engram/        descarga, checksum e instalación
internal/assets/files/  prompt, agentes, skills, comandos y extensión embebidos
```

Consulta [docs/architecture.md](docs/architecture.md) para el ciclo de instalación y el modelo
de datos.

## Licencia

[MIT](LICENSE)
