# MeMo-AI

**Tu agente de código: reúne tus cambios, tus tareas y tu flujo de ingeniería.**

MeMo-AI convierte [Pi](https://pi.dev) (`pi`) en un agente de ingeniería con memoria. Instala un
prompt orquestador, un servidor de memoria persistente
([Engram](https://github.com/Gentleman-Programming/engram)), comandos de barra, skills y una
extensión pequeña.

Es un único binario Go sin dependencias en tiempo de ejecución. Solo edita lo que le pertenece,
guarda una copia de los archivos compartidos antes de cambiarlos y puede eliminar todo lo que
instaló.

> [Read in English](README.md)

---

## Instalación rápida

Necesitas [Pi](https://pi.dev) y [Go 1.23+](https://go.dev/dl/). Engram es opcional: el
asistente puede descargarlo por ti.

### Linux y macOS

```sh
go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest
memo-ai install      # asistente interactivo
memo-ai doctor       # verificación
```

Si la terminal responde `memo-ai: command not found`, añade el directorio bin de Go al `PATH`
(pon la línea en `~/.bashrc` o `~/.zshrc` para que se mantenga):

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Windows (PowerShell)

```powershell
winget install --id GoLang.Go -e     # omítelo si ya tienes Go
# Abre una terminal nueva para que el PATH incluya Go y luego:
go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest
memo-ai install      # asistente interactivo
memo-ai doctor       # verificación
```

El instalador de Go añade `%USERPROFILE%\go\bin` al `PATH`. Si no encuentra `memo-ai`, abre una
terminal nueva o ejecútalo como `& "$env:USERPROFILE\go\bin\memo-ai.exe" install`.

Después abre `pi` y ejecuta `/memo`. Para actualizar más adelante, repite el mismo `go install` y
`memo-ai install`.

## Por qué

Los agentes de código olvidan. Vuelven a preguntar decisiones, releen los mismos archivos y
pierden el hilo después de una compactación. MeMo-AI mantiene tres cosas en un mismo ciclo:

| | Qué hace MeMo-AI |
| --- | --- |
| **Cambios** | Sabe qué hay en el árbol de trabajo; cada cambio de código cierra con evidencia y una línea `Risk:`. `/memo-changes` lo resume. |
| **Tareas** | El trabajo pequeño vive en la conversación; el grande tiene un documento de feature. `/memo-tasks` informa en qué punto está. |
| **Flujo** | Un ciclo fijo (autorizar, recordar, explorar, clasificar, implementar, verificar, persistir), ejecutado en línea en una sola sesión. |

La memoria lo sostiene: decisiones, arreglos y handoffs se guardan en Engram y se recuperan
antes de que el agente vuelva a preguntarte.

## Requisitos

- **Pi** (`pi`) en `PATH`, en su directorio de instalación gestionada (`~/.pi/agent/bin`), o
  `PI_BIN` apuntando al binario. Instálalo desde [pi.dev](https://pi.dev).
- **Go 1.23+** solo si compilas desde el código.
- **Engram** es opcional. MeMo-AI puede detectarlo o descargar la última versión verificada.
- Un runtime de modelos locales (Ollama, LM Studio, vLLM, llama.cpp) es opcional. Cualquier
  `provider/model` que Pi acepte funciona.

## Instalación

| Método | Comando |
| --- | --- |
| Go (cualquier SO) | `go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest && memo-ai install` |
| Desde un checkout (Linux/macOS) | `make build && ./bin/memo-ai install` |
| Script de instalación (Linux/macOS) | `./scripts/install.sh` (compila este checkout en `~/.local/bin`) |
| Desde un checkout (Windows) | `go install ./cmd/memo-ai` y luego `memo-ai install` |

### Windows

El `Makefile` y `scripts/install.sh` son solo POSIX: dependen de `find` e `install`, así que no
funcionan en PowerShell nativo. Usa `go install` como en
[Instalación rápida](#windows-powershell).

`go install` omite los flags del linker, así que `memo-ai version` reporta
`0.1.0-dev`; si necesitas la versión real, compílala desde un checkout:

```powershell
$v = (git describe --tags --always --dirty)
go build -trimpath -ldflags "-s -w -X github.com/luisccalderon/MeMo-AI/internal/version.Version=$v" -o "$env:USERPROFILE\go\bin\memo-ai.exe" ./cmd/memo-ai
```

El instalador pregunta por TTY, así que ejecútalo en Windows Terminal, PowerShell o Git Bash, nunca
con la salida redirigida. Para correr el checkout sin instalar nada:

```powershell
go run ./cmd/memo-ai install
```

Ejemplos no interactivos:

```sh
# Modelo local de Ollama, stack completo
memo-ai install --yes --model ollama/qwen3:8b

# Sin extensión
memo-ai install --yes --no-extension

# Solo memoria, skills y comandos; sin tocar el prompt de sistema de Pi
memo-ai install --yes --no-orchestrator

# Ver cada escritura planificada sin tocar el disco
memo-ai install --yes --dry-run
```

## Qué se instala

Todo vive en el directorio de agente de Pi: `~/.pi/agent` por defecto. Se respeta
`PI_CODING_AGENT_DIR` igual que lo hace Pi.

| Ruta | Propósito | Propiedad |
| --- | --- | --- |
| `APPEND_SYSTEM.md` | El orquestador MeMo, añadido al prompt de sistema de Pi | Solo el bloque entre los marcadores `memo-ai:begin` / `memo-ai:end` |
| `skills/memo-*/SKILL.md` | Procedimientos de memoria, ejecución en línea, handoff y modelos locales | Archivo completo |
| `prompts/memo*.md` | Los comandos `/memo*` (plantillas de prompt de Pi) | Archivo completo |
| `extensions/memo-ai/index.ts` | Insignia de estado, herramienta `memo_status`, `/memo-about`, aviso de compactación | Archivo completo |
| `mcp.json` | Engram como servidor MCP stdio | Solo `mcpServers.engram` |
| `settings.json` | Modelo por defecto, solo si pasas `--model` o eliges uno en el asistente | Solo `defaultProvider` y `defaultModel` |
| `memo-ai/state.json` | Manifiesto de archivos instalados y sus hashes SHA-256 | Archivo completo |
| `memo-ai/backups/` | Copias tomadas antes de cambiar un archivo compartido | Directorio completo |

Se conservan tu texto propio en `APPEND_SYSTEM.md`, tus otros servidores MCP y el resto de
ajustes de Pi. Un `settings.json` que no se pueda leer se deja intacto y se avisa.

## Subagentes

Pi no trae subagentes, así que MeMo-AI no instala ninguno. El orquestador trabaja en línea en una
sola sesión: acota su evidencia, lanza lecturas en paralelo y, en cambios de riesgo alto, termina
con una autorrevisión de solo lectura del diff final. El soporte de subagentes está previsto para
una versión posterior; las definiciones de agentes se conservan en el repositorio para ese
trabajo.

`--no-subagents` y `--subagent-model` se siguen aceptando para no romper scripts existentes, pero
no tienen efecto.

## Cómo funciona el orquestador

El orquestador ejecuta un ciclo fijo en cada petición:

1. **Autorizar** — solo lectura salvo que la petición autorice un cambio.
2. **Recordar** — recupera contexto del proyecto en Engram antes de preguntar.
3. **Explorar** — lee primero el código existente.
4. **Clasificar** — las tareas pequeñas van en línea; las grandes tienen un documento de feature.
5. **Implementar** — una unidad verificable a la vez, primero el test cuando aplica.
6. **Verificar** — informa `comando → resultado`.
7. **Persistir** — guarda el conocimiento duradero y cierra las sesiones sustanciales.

### Política de commits

- **Supervisado (por defecto):** sin commit, push, publicación, merge ni release salvo que lo pidas.
- **Autónomo:** cuando declaras explícitamente una ejecución automática, headless o sin
  supervisión, el orquestador puede crear commits por unidad de trabajo en una rama de feature con
  mensajes Conventional Commit. Push, pull requests, merge, release y operaciones git destructivas
  siguen requiriendo una petición explícita.

## Comandos y skills

| Comando | Qué hace |
| --- | --- |
| `/memo` | Muestra la memoria activa y la configuración de orquestación. |
| `/memo-changes [ref]` | Resume los cambios del árbol de trabajo con su evidencia y riesgo. |
| `/memo-tasks [feature]` | Muestra las tareas registradas, su estado y el siguiente paso. |
| `/memo-recall <tema>` | Busca en la memoria del proyecto y responde con evidencia. |
| `/memo-remember <hecho>` | Guarda una decisión, arreglo o descubrimiento duradero. |
| `/memo-handoff` | Escribe un resumen reanudable antes de pausar o compactar. |
| `/memo-review [foco]` | Ejecuta una revisión de solo lectura de los cambios actuales. |
| `/memo-about` | Muestra la configuración sin gastar un turno del modelo (extensión). |

Las skills aparecen en el prompt de sistema de Pi y se cargan cuando su descripción coincide con
la tarea: `memo-memory`, `memo-delegation` (ejecución en línea), `memo-handoff`,
`memo-local-models`.

## Extensión

Pi descubre `extensions/memo-ai/index.ts` automáticamente. Aporta:

- El logo de MeMo, impreso una vez al inicio de la sesión.
- Una entrada de estado `MEMO` (`MEMO · memory` cuando Engram está configurado).
- Un indicador animado mientras el agente trabaja, con el color de acento del tema activo.
- Una herramienta `memo_status` de solo lectura, para que el agente lea su configuración en vez de
  adivinarla.
- El comando `/memo-about`.
- Un aviso para ejecutar `/memo-handoff` cuando el contexto está por compactarse.

Las extensiones se ejecutan con todos tus permisos. Revisa `index.ts` si quieres saber
exactamente qué hace. Desactívala con `--no-extension`. Reconfigura con `memo-ai configure` cuando
quieras. Ejecuta `/reload` en Pi después de cambiar archivos instalados.

## Memoria con Engram

MeMo-AI añade Engram a `~/.pi/agent/mcp.json`:

```json
{
  "mcpServers": {
    "engram": {
      "type": "stdio",
      "command": "/usr/bin/engram",
      "args": ["mcp", "--tools=agent"],
      "exposure": "direct"
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

MeMo-AI detecta Ollama (`:11434`), LM Studio (`:1234`) y vLLM (`:8000`) y ofrece sus modelos en el
asistente. Pi solo lista un modelo cuyo proveedor esté declarado, así que Ollama, LM Studio y vLLM
necesitan una entrada en `~/.pi/agent/models.json`. llama.cpp viene integrado (`/llama`).

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
```

Consulta [docs/local-models.md](docs/local-models.md) para la entrada de `models.json` y la guía
de capacidad.

## Arranque silencioso

Por defecto Pi lista al iniciar cada archivo de contexto, skill, prompt y extensión que carga. El
instalador ofrece fijar `quietStartup` en `"header"` dentro de `settings.json`: Pi conserva su
encabezado con la versión, el listado se oculta y el logo de MeMo muestra los prompts en su lugar.
`ctrl+o` o `pi --verbose` siguen mostrando el listado completo.

```sh
memo-ai install --quiet-startup      # lo aplica sin preguntar
memo-ai install --no-quiet-startup   # nunca lo cambia ni pregunta
```

Las ejecuciones no interactivas (`--yes`) no tocan el ajuste salvo que pases `--quiet-startup`.
Un valor de `quietStartup` que ya hayas elegido nunca se reemplaza, y la desinstalación quita solo
el valor que aplicó MeMo-AI.

## Modelos gratis

Dos paquetes de la comunidad añaden proveedores de modelos gratis al catálogo de Pi:

| Paquete | Qué aporta |
| --- | --- |
| `npm:pi-free` | Filtrado de modelos gratis y descarga dinámica del catálogo |
| `npm:opencode-pi` | Conecta los modelos gratis de OpenCode a Pi sin iniciar sesión en OpenCode |

El instalador ofrece ambos en modo interactivo y ejecuta `pi install` por cada uno, así que Pi es
dueño de la entrada `packages` en `settings.json` y puede actualizarlos después con `pi update`.

```sh
memo-ai install --free-models      # los instala sin preguntar
memo-ai install --no-free-models   # nunca los instala ni pregunta
```

Las ejecuciones no interactivas (`--yes`) los omiten salvo que pases `--free-models`, así una
instalación desatendida nunca descarga paquetes que no pediste. La lista de proveedores no viene
incluida en el binario y no ofrece garantías; la cuota gratis es la que permita cada proveedor.
`memo-ai doctor` informa si están presentes.

`opencode-pi` inicia el CLI de OpenCode sin shell. En Windows eso falla con
`spawn opencode ENOENT` cuando `opencode` es solo un lanzador `.cmd`/`.ps1` en el `PATH` (la
instalación global de npm, por ejemplo). El instalador lo detecta, busca el `opencode.exe` real y
ofrece fijar la variable de entorno de usuario `OPENCODE_PI_BIN` con esa ruta; con `--free-models`
en una ejecución no interactiva la fija sin preguntar. Abre una terminal nueva después. Linux y
macOS no necesitan la variable: `memo-ai doctor` solo comprueba que `opencode` esté en el `PATH`. La desinstalación deja los paquetes y la variable, porque Pi los sigue usando.

## Barra de estado y sidebar

`pi-atelier` es un paquete de la comunidad que añade una barra de estado y una sidebar de
actividad en vivo: modelo, contexto, costo, estado de Git, TODOs y actividad de herramientas.
Ocupa el footer y el editor propios de Pi, por eso se instala solo si lo pides y nunca viene
dentro del binario.

```sh
memo-ai install --atelier            # lo instala sin preguntar
memo-ai install --no-atelier         # nunca lo instala ni pregunta
memo-ai install --atelier-config ./mi-pi-atelier.json   # copia tu propia configuración
```

Para llevar un look concreto a otra máquina, pasa tu archivo con `--atelier-config`. Se aplica
tal cual, implica `--atelier` y rechaza cualquier cosa que no sea un objeto JSON, para que un
archivo roto nunca llegue al paquete. Si ya existe un `pi-atelier.json`, se respalda antes y se
informa la ruta, así no se pierde nada.

Sin ese flag, cuando el paquete se instala y `pi-atelier.json` aún no existe, el instalador
escribe un preset mínimo:

```json
{
  "nerdFont": false
}
```

Las ejecuciones no interactivas (`--yes`) omiten el paquete salvo que pases `--atelier` o
`--atelier-config`. `pi install` es dueño de la entrada `packages`, así que `pi update` lo
mantiene al día.

El texto plano funciona con cualquier fuente monoespaciada. El resto de los valores queda en manos
del paquete, para que sus defaults sigan evolucionando. Sin `--atelier-config`, un
`pi-atelier.json` existente nunca se toca. La desinstalación solo borra un archivo que MeMo-AI
escribió y tú no modificaste; un archivo copiado que editaste se conserva salvo que uses
`--force`.

MeMo-AI no reimplementa esa UI: su logo sigue siendo una entrada del chat y su estado
`MEMO · memory` sigue en la línea de estado, así que ambos conviven. `memo-ai doctor` informa si
está presente. La desinstalación deja el paquete, porque Pi lo sigue usando; quítalo con
`pi uninstall npm:pi-atelier`.

## Seguridad

- Solo se eliminan los archivos registrados en `memo-ai/state.json`.
- Los archivos que modificaste se omiten salvo que uses `--force`.
- `APPEND_SYSTEM.md`, `mcp.json` y `settings.json` se editan en su sitio: solo se tocan el bloque
  MeMo, la entrada de Engram y las dos claves del modelo por defecto, y antes se toma una copia.
- `--dry-run` muestra cada escritura planificada sin tocar el disco.
- El orquestador solo hace commits en una ejecución autónoma declarada; push, pull requests, merge
  y release siempre requieren una petición explícita.
- Ningún dato sale de tu máquina salvo la descarga opcional de Engram.

```sh
memo-ai uninstall            # quita los archivos instalados, el bloque del prompt y la entrada de Engram
memo-ai uninstall --force    # también quita los archivos que modificaste
```

La desinstalación deja el modelo por defecto de Pi como está y te indica cómo restablecerlo.

## Referencia de comandos

```text
memo-ai [comando] [flags]

Comandos
  install       Instala o reconfigura el stack (interactivo)
  configure     Reconfigura el orquestador, el modelo y la extensión
  doctor        Chequeo de salud de solo lectura (admite --json)
  agents        Muestra la configuración de MeMo-AI y los modelos locales detectados
  engram        Estado o instalación de Engram
  uninstall     Quita solo lo que MeMo-AI instaló
  version       Imprime la versión

Flags globales
  --lang es|en
  --yes
  --dry-run
  --model <provider/model>
  --no-orchestrator
  --no-engram
  --no-extension
  --free-models
  --no-free-models
  --atelier
  --no-atelier
  --atelier-config <archivo>
  --quiet-startup
  --no-quiet-startup
  --install-engram
  --force
  --restore
  --json
  --no-subagents                      (se acepta, sin efecto)
  --subagent-model <provider/model>   (se acepta, sin efecto)
```

## Si vienes de la versión de oh-my-pi

La versión anterior apuntaba a oh-my-pi (`omp`). Esa instalación vive en `~/.omp/agent` y esta
versión no la toca. Quítala con el binario anterior (`memo-ai uninstall`) antes de actualizar y
luego ejecuta `memo-ai install` de nuevo.

| Antes (oh-my-pi) | Ahora (Pi) |
| --- | --- |
| `~/.omp/agent` | `~/.pi/agent` |
| `OMP_BIN`, `OMP_PROFILE`, `PI_CONFIG_DIR` | `PI_BIN`, `PI_CODING_AGENT_DIR` |
| `commands/memo*.md` | `prompts/memo*.md` |
| `omp config set modelRoles.default` | `defaultProvider` / `defaultModel` en `settings.json` |
| Agentes de tarea `agents/memo-*.md` | No se instalan: Pi aún no tiene subagentes |
| `/memo-agents` | Eliminado |
| `@oh-my-pi/pi-coding-agent`, `pi.zod` | `@earendil-works/pi-coding-agent`, `typebox` |

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
internal/cli/           parseo de flags, asistente, comandos
internal/install/       escritura de assets, bloque del prompt, fusión de mcp.json, modelo por defecto
internal/uninstall/     eliminación segura y limpieza
internal/promptblock/   bloque gestionado dentro de APPEND_SYSTEM.md
internal/pi/            resolución del directorio de agente de Pi
internal/config/        carga/guardado JSON/JSONC y helpers anidados
internal/detect/        detección de Pi, Engram y modelos locales
internal/doctor/        chequeos de salud
internal/engram/        descarga, checksum e instalación de la release
internal/assets/files/  prompt, skills, plantillas de prompt, extensión y agentes (no se instalan) embebidos
```

Consulta [docs/architecture.md](docs/architecture.md) para el ciclo de instalación y el modelo de
datos.

## Licencia

[MIT](LICENSE)
