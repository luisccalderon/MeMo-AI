// Package i18n provides the small bilingual message catalogue used by the CLI.
package i18n

import (
	"fmt"
	"strings"
)

// Supported languages.
const (
	Spanish = "es"
	English = "en"
)

// Normalize maps user input to a supported language code.
func Normalize(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "en-us", "en-gb", "english", "inglés", "ingles":
		return English
	case "es", "es-es", "es-mx", "spanish", "español", "espanol":
		return Spanish
	default:
		return Spanish
	}
}

var messages = map[string]map[string]string{
	Spanish: {
		"app.name":           "MeMo-AI",
		"app.tagline":        "Tu agente de código: tus cambios, tareas y flujo de ingeniería en un solo lugar.",
		"common.yes":         "Sí",
		"common.no":          "No",
		"common.none":        "Ninguno",
		"common.inherit":     "Heredar (modelo por defecto de Pi)",
		"common.current":     "Actual",
		"common.recommended": "recomendado",
		"common.default":     "por defecto",
		"common.cancel":      "Cancelar",
		"common.back":        "Volver",
		"common.done":        "Listo",
		"common.warning":     "Aviso",
		"common.error":       "Error",
		"common.ok":          "OK",
		"common.skip":        "Omitir",
		"common.manual":      "Escribir manualmente",

		"help.usage":                "Uso: memo-ai [comando] [flags]",
		"help.intro":                "Instala MeMo-AI sobre Pi (pi.dev): orquestación, memoria persistente y evidencia.",
		"help.commands":             "Comandos",
		"help.flags":                "Flags globales",
		"help.examples":             "Ejemplos",
		"help.cmd.install":          "Instala o reconfigura el stack MeMo-AI (asistente interactivo).",
		"help.cmd.configure":        "Reconfigura orquestador, modelo y extensión.",
		"help.cmd.doctor":           "Diagnóstico de solo lectura del entorno y la instalación.",
		"help.cmd.agents":           "Muestra la configuración de MeMo-AI y los modelos locales detectados.",
		"help.cmd.engram":           "Gestiona la memoria Engram (estado o instalación).",
		"help.cmd.uninstall":        "Elimina solo los archivos que instaló MeMo-AI.",
		"help.cmd.version":          "Muestra la versión.",
		"help.flag.lang":            "Idioma del CLI: es | en.",
		"help.flag.yes":             "Acepta los valores recomendados sin preguntar.",
		"help.flag.dry-run":         "Muestra los cambios sin escribir archivos.",
		"help.flag.no-orchestrator": "No instala el prompt del orquestador (solo memoria, skills y comandos).",
		"help.flag.model":           "Modelo por defecto como provider/model (defaultProvider/defaultModel de Pi).",
		"help.flag.no-subagents":    "Sin efecto: Pi aún no soporta subagentes (se acepta por compatibilidad).",
		"help.flag.no-engram":       "No configura el servidor MCP de Engram.",
		"help.flag.no-extension":    "No instala la extensión de Pi (insignia de estado y memo_status).",
		"help.flag.free-models":     "Instala pi-free y opencode-pi con `pi install` (proveedores de modelos gratis).",
		"help.flag.no-free-models":  "No instala ni pregunta por los proveedores de modelos gratis.",

		"wizard.detecting":            "Detectando el entorno...",
		"wizard.detected":             "Entorno detectado",
		"wizard.pi":                   "Pi",
		"wizard.engram":               "Engram",
		"wizard.local_models":         "Modelos locales",
		"wizard.no_local_models":      "No hay runtimes locales respondiendo en Ollama, LM Studio o vLLM.",
		"wizard.language":             "Idioma del CLI",
		"wizard.orchestrator":         "¿Instalar el orquestador MeMo en el prompt de Pi (APPEND_SYSTEM.md)?",
		"wizard.primary_model":        "Modelo principal",
		"wizard.model_hint":           "Elige un modelo local detectado o escribe provider/model.",
		"wizard.model_manual":         "provider/model (ej. ollama/qwen3:8b)",
		"wizard.engram_enable":        "¿Configurar Engram como memoria persistente por MCP?",
		"wizard.engram_missing":       "Engram no está instalado.",
		"wizard.engram_install":       "¿Descargar e instalar Engram ahora?",
		"wizard.extension":            "¿Instalar la extensión de MeMo-AI para Pi (insignia de estado, memo_status y /memo-about)?",
		"wizard.free_models":          "¿Instalar los proveedores de modelos gratis (pi-free y opencode-pi) con `pi install`?",
		"wizard.free_models_hint":     "Paquetes de la comunidad: agregan modelos free al catálogo de Pi.",
		"wizard.opencode_bin_hint":    "opencode-pi no puede iniciar los lanzadores .cmd/.ps1 de OpenCode; necesita la ruta del ejecutable real.",
		"wizard.opencode_bin":         "¿Fijar esa variable de entorno en tu usuario para que los modelos de OpenCode funcionen?",
		"wizard.opencode_bin_missing": "opencode-pi no podrá iniciar OpenCode: no se encontró opencode.exe. Define OPENCODE_PI_BIN con su ruta completa.",
		"wizard.confirm":              "Resumen de la instalación",
		"wizard.installing":           "Aplicando cambios...",
		"wizard.done":                 "MeMo-AI quedó configurado.",
		"wizard.next":                 "Abre pi y ejecuta /memo. Usa /memo-changes y /memo-tasks para ver cambios y tareas.",
		"wizard.cancelled":            "Instalación cancelada.",

		"install.config_backup":          "Respaldo de configuración",
		"install.files_written":          "Archivos escritos",
		"install.config_updated":         "Configuración actualizada",
		"install.engram_ready":           "Engram configurado",
		"install.engram_skipped":         "Engram omitido",
		"install.extension_ready":        "Extensión de Pi instalada",
		"install.free_models_installing": "Instalando proveedor de modelos gratis",
		"install.free_models_present":    "Proveedor de modelos gratis ya instalado",
		"install.free_models_ready":      "Proveedor de modelos gratis instalado",
		"install.free_models_failed":     "No se pudo instalar el proveedor de modelos gratis",
		"install.opencode_bin_ready":     "Variable de entorno de usuario fijada",
		"install.opencode_bin_failed":    "No se pudo fijar la variable de entorno",
		"install.opencode_bin_restart":   "Abre una terminal nueva para que Pi la vea.",
		"install.subagents_off":          "Subagentes no soportados en Pi todavía: todo corre en línea",
		"install.prompt_ready":           "Prompt del orquestador instalado",
		"install.prompt_skipped":         "Orquestador omitido: solo memoria, skills y comandos",
		"install.dry_run":                "Modo simulación: no se escribió nada.",

		"doctor.title":         "Diagnóstico de MeMo-AI",
		"doctor.pass":          "OK",
		"doctor.warn":          "AVISO",
		"doctor.fail":          "FALLA",
		"doctor.summary":       "Resumen",
		"doctor.no_state":      "No hay instalación registrada. Ejecuta `memo-ai install`.",
		"doctor.config":        "Configuración MCP de Pi",
		"doctor.agents":        "Agentes MeMo-AI",
		"doctor.skills":        "Skills MeMo-AI",
		"doctor.commands":      "Comandos MeMo-AI",
		"doctor.plugin":        "Extensión de Pi",
		"doctor.engram":        "Engram",
		"doctor.mcp":           "Servidor MCP de Engram",
		"doctor.default_agent": "Prompt del orquestador",
		"doctor.model":         "Modelo principal",
		"doctor.local_models":  "Runtimes locales",
		"doctor.state":         "Estado de instalación",

		"uninstall.title":    "Desinstalación de MeMo-AI",
		"uninstall.confirm":  "¿Eliminar los archivos instalados por MeMo-AI?",
		"uninstall.restore":  "¿Restaurar el mcp.json previo a la instalación?",
		"uninstall.removed":  "Eliminado",
		"uninstall.skipped":  "Omitido (modificado por ti)",
		"uninstall.restored": "Configuración restaurada",
		"uninstall.done":     "Desinstalación completada.",
		"uninstall.force":    "Elimina también archivos modificados por el usuario.",

		"agents.title":                 "Agentes y modelos detectados",
		"agents.primary":               "Orquestador MeMo",
		"agents.subagents":             "Subagentes MeMo-AI",
		"agents.subagents_unsupported": "no soportados en Pi todavía (todo corre en línea)",
		"wizard.subagents_ignored":     "--subagent-model se ignora: Pi aún no soporta subagentes.",
		"agents.local":                 "Modelos locales detectados",
		"agents.no_config":             "Sin configuración de MeMo-AI todavía.",

		"engram.status":       "Estado de Engram",
		"engram.binary":       "Binario",
		"engram.version":      "Versión",
		"engram.installed":    "Instalado en",
		"engram.missing":      "No encontrado",
		"engram.installing":   "Descargando Engram...",
		"engram.installed_ok": "Engram instalado correctamente.",
		"engram.configure":    "Configura Engram con `memo-ai install`.",

		"error.pi_missing":      "pi no está en PATH. Instálalo (https://pi.dev) antes de continuar.",
		"error.config_parse":    "No se pudo leer mcp.json de Pi: %s",
		"error.write":           "No se pudo escribir %s: %s",
		"error.unknown_command": "Comando desconocido: %s",
		"error.no_tty":          "El asistente necesita una terminal interactiva. Usa flags o --yes.",
	},
	English: {
		"app.name":           "MeMo-AI",
		"app.tagline":        "Your coding agent, bringing your changes, tasks, and engineering workflow together.",
		"common.yes":         "Yes",
		"common.no":          "No",
		"common.none":        "None",
		"common.inherit":     "Inherit (Pi default model)",
		"common.current":     "Current",
		"common.recommended": "recommended",
		"common.default":     "default",
		"common.cancel":      "Cancel",
		"common.back":        "Back",
		"common.done":        "Done",
		"common.warning":     "Warning",
		"common.error":       "Error",
		"common.ok":          "OK",
		"common.skip":        "Skip",
		"common.manual":      "Enter manually",

		"help.usage":                "Usage: memo-ai [command] [flags]",
		"help.intro":                "Install MeMo-AI on top of Pi (pi.dev): orchestration, persistent memory and evidence.",
		"help.commands":             "Commands",
		"help.flags":                "Global flags",
		"help.examples":             "Examples",
		"help.cmd.install":          "Install or reconfigure the MeMo-AI stack (interactive wizard).",
		"help.cmd.configure":        "Reconfigure the orchestrator, model and extension.",
		"help.cmd.doctor":           "Read-only health check for the environment and installation.",
		"help.cmd.agents":           "Show the MeMo-AI configuration and detected local models.",
		"help.cmd.engram":           "Manage Engram memory (status or install).",
		"help.cmd.uninstall":        "Remove only the files installed by MeMo-AI.",
		"help.cmd.version":          "Print the version.",
		"help.flag.lang":            "CLI language: es | en.",
		"help.flag.yes":             "Accept recommended values without prompting.",
		"help.flag.dry-run":         "Show changes without writing files.",
		"help.flag.no-orchestrator": "Do not install the orchestrator prompt (memory, skills and commands only).",
		"help.flag.model":           "Default model as provider/model (Pi defaultProvider/defaultModel).",
		"help.flag.no-subagents":    "No effect: Pi does not support subagents yet (accepted for compatibility).",
		"help.flag.no-engram":       "Do not configure the Engram MCP server.",
		"help.flag.no-extension":    "Do not install the Pi extension (status badge and memo_status).",
		"help.flag.free-models":     "Install pi-free and opencode-pi with `pi install` (free model providers).",
		"help.flag.no-free-models":  "Do not install or ask about the free model providers.",

		"wizard.detecting":            "Detecting environment...",
		"wizard.detected":             "Detected environment",
		"wizard.pi":                   "Pi",
		"wizard.engram":               "Engram",
		"wizard.local_models":         "Local models",
		"wizard.no_local_models":      "No local runtime answered on Ollama, LM Studio or vLLM.",
		"wizard.language":             "CLI language",
		"wizard.orchestrator":         "Install the MeMo orchestrator into the Pi prompt (APPEND_SYSTEM.md)?",
		"wizard.primary_model":        "Primary model",
		"wizard.model_hint":           "Pick a detected local model or type provider/model.",
		"wizard.model_manual":         "provider/model (e.g. ollama/qwen3:8b)",
		"wizard.engram_enable":        "Configure Engram as persistent memory over MCP?",
		"wizard.engram_missing":       "Engram is not installed.",
		"wizard.engram_install":       "Download and install Engram now?",
		"wizard.extension":            "Install the MeMo-AI Pi extension (status badge, memo_status and /memo-about)?",
		"wizard.free_models":          "Install the free model providers (pi-free and opencode-pi) with `pi install`?",
		"wizard.free_models_hint":     "Community packages: they add free models to Pi's catalog.",
		"wizard.opencode_bin_hint":    "opencode-pi cannot start OpenCode's .cmd/.ps1 launchers; it needs the path of the real executable.",
		"wizard.opencode_bin":         "Set that environment variable for your user so the OpenCode models work?",
		"wizard.opencode_bin_missing": "opencode-pi will not be able to start OpenCode: opencode.exe was not found. Set OPENCODE_PI_BIN to its full path.",
		"wizard.confirm":              "Installation summary",
		"wizard.installing":           "Applying changes...",
		"wizard.done":                 "MeMo-AI is configured.",
		"wizard.next":                 "Open pi and run /memo. Use /memo-changes and /memo-tasks to see your changes and tasks.",
		"wizard.cancelled":            "Installation cancelled.",

		"install.config_backup":          "Configuration backup",
		"install.files_written":          "Files written",
		"install.config_updated":         "Configuration updated",
		"install.engram_ready":           "Engram configured",
		"install.engram_skipped":         "Engram skipped",
		"install.extension_ready":        "Pi extension installed",
		"install.free_models_installing": "Installing free model provider",
		"install.free_models_present":    "Free model provider already installed",
		"install.free_models_ready":      "Free model provider installed",
		"install.free_models_failed":     "Could not install the free model provider",
		"install.opencode_bin_ready":     "User environment variable set",
		"install.opencode_bin_failed":    "Could not set the environment variable",
		"install.opencode_bin_restart":   "Open a new terminal so Pi sees it.",
		"install.subagents_off":          "Subagents are not supported on Pi yet: everything runs inline",
		"install.prompt_ready":           "Orchestrator prompt installed",
		"install.prompt_skipped":         "Orchestrator skipped: memory, skills and commands only",
		"install.dry_run":                "Dry run: nothing was written.",

		"doctor.title":         "MeMo-AI health check",
		"doctor.pass":          "OK",
		"doctor.warn":          "WARN",
		"doctor.fail":          "FAIL",
		"doctor.summary":       "Summary",
		"doctor.no_state":      "No installation recorded. Run `memo-ai install`.",
		"doctor.config":        "Pi MCP configuration",
		"doctor.agents":        "MeMo-AI agents",
		"doctor.skills":        "MeMo-AI skills",
		"doctor.commands":      "MeMo-AI commands",
		"doctor.plugin":        "Pi extension",
		"doctor.engram":        "Engram",
		"doctor.mcp":           "Engram MCP server",
		"doctor.default_agent": "Orchestrator prompt",
		"doctor.model":         "Primary model",
		"doctor.local_models":  "Local runtimes",
		"doctor.state":         "Installation state",

		"uninstall.title":    "MeMo-AI uninstall",
		"uninstall.confirm":  "Remove the files installed by MeMo-AI?",
		"uninstall.restore":  "Restore the mcp.json from before the install?",
		"uninstall.removed":  "Removed",
		"uninstall.skipped":  "Skipped (modified by you)",
		"uninstall.restored": "Configuration restored",
		"uninstall.done":     "Uninstall complete.",
		"uninstall.force":    "Also remove files modified by the user.",

		"agents.title":                 "Detected agents and models",
		"agents.primary":               "MeMo orchestrator",
		"agents.subagents":             "MeMo-AI subagents",
		"agents.subagents_unsupported": "not supported on Pi yet (everything runs inline)",
		"wizard.subagents_ignored":     "--subagent-model is ignored: Pi does not support subagents yet.",
		"agents.local":                 "Detected local models",
		"agents.no_config":             "No MeMo-AI configuration yet.",

		"engram.status":       "Engram status",
		"engram.binary":       "Binary",
		"engram.version":      "Version",
		"engram.installed":    "Installed at",
		"engram.missing":      "Not found",
		"engram.installing":   "Downloading Engram...",
		"engram.installed_ok": "Engram installed successfully.",
		"engram.configure":    "Configure Engram with `memo-ai install`.",

		"error.pi_missing":      "pi is not on PATH. Install it (https://pi.dev) before continuing.",
		"error.config_parse":    "Could not read the Pi mcp.json: %s",
		"error.write":           "Could not write %s: %s",
		"error.unknown_command": "Unknown command: %s",
		"error.no_tty":          "The wizard needs an interactive terminal. Use flags or --yes.",
	},
}

// T returns a translated message, falling back to Spanish and then the key.
func T(lang, key string) string {
	lang = Normalize(lang)
	if m, ok := messages[lang][key]; ok {
		return m
	}
	if m, ok := messages[Spanish][key]; ok {
		return m
	}
	return key
}

// Tf is T with fmt.Sprintf formatting.
func Tf(lang, key string, args ...any) string {
	return fmt.Sprintf(T(lang, key), args...)
}
