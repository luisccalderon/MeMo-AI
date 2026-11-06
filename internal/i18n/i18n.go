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
		"app.tagline":        "Memoria, orquestación y evidencia para OpenCode.",
		"common.yes":         "Sí",
		"common.no":          "No",
		"common.none":        "Ninguno",
		"common.inherit":     "Heredar del agente principal",
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

		"help.usage":             "Uso: memo-ai [comando] [flags]",
		"help.intro":             "Configura un stack de orquestación con memoria persistente para OpenCode.",
		"help.commands":          "Comandos",
		"help.flags":             "Flags globales",
		"help.examples":          "Ejemplos",
		"help.cmd.install":       "Instala o reconfigura el stack MeMo-AI (asistente interactivo).",
		"help.cmd.configure":     "Reconfigura agentes, modelo principal y subagentes.",
		"help.cmd.doctor":        "Diagnóstico de solo lectura del entorno y la instalación.",
		"help.cmd.agents":        "Lista agentes, modelos locales y configuración detectada.",
		"help.cmd.engram":        "Gestiona la memoria Engram (estado o instalación).",
		"help.cmd.uninstall":     "Elimina solo los archivos que instaló MeMo-AI.",
		"help.cmd.version":       "Muestra la versión.",
		"help.flag.lang":         "Idioma del CLI: es | en.",
		"help.flag.yes":          "Acepta los valores recomendados sin preguntar.",
		"help.flag.dry-run":      "Muestra los cambios sin escribir archivos.",
		"help.flag.agent":        "Agente principal (memo-orchestrator, build, plan...).",
		"help.flag.model":        "Modelo principal en formato provider/model.",
		"help.flag.no-subagents": "Desactiva la delegación a subagentes.",
		"help.flag.no-engram":    "No configura el servidor MCP de Engram.",
		"help.flag.no-plugin":    "No instala el plugin TUI con el logo.",

		"wizard.detecting":        "Detectando el entorno...",
		"wizard.detected":         "Entorno detectado",
		"wizard.opencode":         "OpenCode",
		"wizard.engram":           "Engram",
		"wizard.local_models":     "Modelos locales",
		"wizard.no_local_models":  "No hay runtimes locales respondiendo en Ollama, LM Studio o vLLM.",
		"wizard.language":         "Idioma del CLI",
		"wizard.primary_agent":    "Agente principal",
		"wizard.primary_model":    "Modelo principal",
		"wizard.model_hint":       "Elige un modelo local detectado o escribe provider/model.",
		"wizard.model_manual":     "provider/model (ej. ollama/qwen3:8b)",
		"wizard.subagents":        "¿Habilitar subagentes?",
		"wizard.subagents_choose": "Subagentes habilitados",
		"wizard.subagent_model":   "Modelo para subagentes",
		"wizard.engram_enable":    "¿Configurar Engram como memoria persistente por MCP?",
		"wizard.engram_missing":   "Engram no está instalado.",
		"wizard.engram_install":   "¿Descargar e instalar Engram ahora?",
		"wizard.plugin":           "¿Instalar el plugin TUI de MeMo-AI (logo y comando /memo)?",
		"wizard.confirm":          "Resumen de la instalación",
		"wizard.installing":       "Aplicando cambios...",
		"wizard.done":             "MeMo-AI quedó configurado.",
		"wizard.next":             "Abre OpenCode y ejecuta /memo. Usa /memo-recall antes de reexplicar contexto.",
		"wizard.cancelled":        "Instalación cancelada.",

		"install.config_backup":  "Respaldo de configuración",
		"install.files_written":  "Archivos escritos",
		"install.config_updated": "Configuración actualizada",
		"install.engram_ready":   "Engram configurado",
		"install.engram_skipped": "Engram omitido",
		"install.plugin_ready":   "Plugin TUI instalado",
		"install.subagents_off":  "Subagentes desactivados",
		"install.dry_run":        "Modo simulación: no se escribió nada.",

		"doctor.title":         "Diagnóstico de MeMo-AI",
		"doctor.pass":          "OK",
		"doctor.warn":          "AVISO",
		"doctor.fail":          "FALLA",
		"doctor.summary":       "Resumen",
		"doctor.no_state":      "No hay instalación registrada. Ejecuta `memo-ai install`.",
		"doctor.config":        "Configuración de OpenCode",
		"doctor.agents":        "Agentes MeMo-AI",
		"doctor.skills":        "Skills MeMo-AI",
		"doctor.commands":      "Comandos MeMo-AI",
		"doctor.plugin":        "Plugin TUI",
		"doctor.engram":        "Engram",
		"doctor.mcp":           "Servidor MCP de Engram",
		"doctor.default_agent": "Agente principal",
		"doctor.model":         "Modelo principal",
		"doctor.local_models":  "Runtimes locales",
		"doctor.state":         "Estado de instalación",

		"uninstall.title":    "Desinstalación de MeMo-AI",
		"uninstall.confirm":  "¿Eliminar los archivos instalados por MeMo-AI?",
		"uninstall.restore":  "¿Restaurar la configuración previa de OpenCode?",
		"uninstall.removed":  "Eliminado",
		"uninstall.skipped":  "Omitido (modificado por ti)",
		"uninstall.restored": "Configuración restaurada",
		"uninstall.done":     "Desinstalación completada.",
		"uninstall.force":    "Elimina también archivos modificados por el usuario.",

		"agents.title":     "Agentes y modelos detectados",
		"agents.primary":   "Agente principal configurado",
		"agents.available": "Agentes disponibles",
		"agents.subagents": "Subagentes MeMo-AI",
		"agents.local":     "Modelos locales detectados",
		"agents.no_config": "Sin configuración de MeMo-AI todavía.",

		"engram.status":       "Estado de Engram",
		"engram.binary":       "Binario",
		"engram.version":      "Versión",
		"engram.installed":    "Instalado en",
		"engram.missing":      "No encontrado",
		"engram.installing":   "Descargando Engram...",
		"engram.installed_ok": "Engram instalado correctamente.",
		"engram.configure":    "Configura Engram con `memo-ai install` o `engram setup opencode`.",

		"error.opencode_missing": "OpenCode no está en PATH. Instálalo antes de continuar.",
		"error.config_parse":     "No se pudo leer la configuración de OpenCode: %s",
		"error.write":            "No se pudo escribir %s: %s",
		"error.unknown_command":  "Comando desconocido: %s",
		"error.no_tty":           "El asistente necesita una terminal interactiva. Usa flags o --yes.",
	},
	English: {
		"app.name":           "MeMo-AI",
		"app.tagline":        "Memory, orchestration and evidence for OpenCode.",
		"common.yes":         "Yes",
		"common.no":          "No",
		"common.none":        "None",
		"common.inherit":     "Inherit from the primary agent",
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

		"help.usage":             "Usage: memo-ai [command] [flags]",
		"help.intro":             "Configure a memory-first orchestration stack for OpenCode.",
		"help.commands":          "Commands",
		"help.flags":             "Global flags",
		"help.examples":          "Examples",
		"help.cmd.install":       "Install or reconfigure the MeMo-AI stack (interactive wizard).",
		"help.cmd.configure":     "Reconfigure agents, primary model and subagents.",
		"help.cmd.doctor":        "Read-only health check for the environment and installation.",
		"help.cmd.agents":        "List agents, local models and detected configuration.",
		"help.cmd.engram":        "Manage Engram memory (status or install).",
		"help.cmd.uninstall":     "Remove only the files installed by MeMo-AI.",
		"help.cmd.version":       "Print the version.",
		"help.flag.lang":         "CLI language: es | en.",
		"help.flag.yes":          "Accept recommended values without prompting.",
		"help.flag.dry-run":      "Show changes without writing files.",
		"help.flag.agent":        "Primary agent (memo-orchestrator, build, plan...).",
		"help.flag.model":        "Primary model as provider/model.",
		"help.flag.no-subagents": "Disable delegation to subagents.",
		"help.flag.no-engram":    "Do not configure the Engram MCP server.",
		"help.flag.no-plugin":    "Do not install the TUI plugin with the logo.",

		"wizard.detecting":        "Detecting environment...",
		"wizard.detected":         "Detected environment",
		"wizard.opencode":         "OpenCode",
		"wizard.engram":           "Engram",
		"wizard.local_models":     "Local models",
		"wizard.no_local_models":  "No local runtime answered on Ollama, LM Studio or vLLM.",
		"wizard.language":         "CLI language",
		"wizard.primary_agent":    "Primary agent",
		"wizard.primary_model":    "Primary model",
		"wizard.model_hint":       "Pick a detected local model or type provider/model.",
		"wizard.model_manual":     "provider/model (e.g. ollama/qwen3:8b)",
		"wizard.subagents":        "Enable subagents?",
		"wizard.subagents_choose": "Enabled subagents",
		"wizard.subagent_model":   "Model for subagents",
		"wizard.engram_enable":    "Configure Engram as persistent memory over MCP?",
		"wizard.engram_missing":   "Engram is not installed.",
		"wizard.engram_install":   "Download and install Engram now?",
		"wizard.plugin":           "Install the MeMo-AI TUI plugin (logo and /memo command)?",
		"wizard.confirm":          "Installation summary",
		"wizard.installing":       "Applying changes...",
		"wizard.done":             "MeMo-AI is configured.",
		"wizard.next":             "Open OpenCode and run /memo. Use /memo-recall before re-explaining context.",
		"wizard.cancelled":        "Installation cancelled.",

		"install.config_backup":  "Configuration backup",
		"install.files_written":  "Files written",
		"install.config_updated": "Configuration updated",
		"install.engram_ready":   "Engram configured",
		"install.engram_skipped": "Engram skipped",
		"install.plugin_ready":   "TUI plugin installed",
		"install.subagents_off":  "Subagents disabled",
		"install.dry_run":        "Dry run: nothing was written.",

		"doctor.title":         "MeMo-AI health check",
		"doctor.pass":          "OK",
		"doctor.warn":          "WARN",
		"doctor.fail":          "FAIL",
		"doctor.summary":       "Summary",
		"doctor.no_state":      "No installation recorded. Run `memo-ai install`.",
		"doctor.config":        "OpenCode configuration",
		"doctor.agents":        "MeMo-AI agents",
		"doctor.skills":        "MeMo-AI skills",
		"doctor.commands":      "MeMo-AI commands",
		"doctor.plugin":        "TUI plugin",
		"doctor.engram":        "Engram",
		"doctor.mcp":           "Engram MCP server",
		"doctor.default_agent": "Primary agent",
		"doctor.model":         "Primary model",
		"doctor.local_models":  "Local runtimes",
		"doctor.state":         "Installation state",

		"uninstall.title":    "MeMo-AI uninstall",
		"uninstall.confirm":  "Remove the files installed by MeMo-AI?",
		"uninstall.restore":  "Restore the previous OpenCode configuration?",
		"uninstall.removed":  "Removed",
		"uninstall.skipped":  "Skipped (modified by you)",
		"uninstall.restored": "Configuration restored",
		"uninstall.done":     "Uninstall complete.",
		"uninstall.force":    "Also remove files modified by the user.",

		"agents.title":     "Detected agents and models",
		"agents.primary":   "Configured primary agent",
		"agents.available": "Available agents",
		"agents.subagents": "MeMo-AI subagents",
		"agents.local":     "Detected local models",
		"agents.no_config": "No MeMo-AI configuration yet.",

		"engram.status":       "Engram status",
		"engram.binary":       "Binary",
		"engram.version":      "Version",
		"engram.installed":    "Installed at",
		"engram.missing":      "Not found",
		"engram.installing":   "Downloading Engram...",
		"engram.installed_ok": "Engram installed successfully.",
		"engram.configure":    "Configure Engram with `memo-ai install` or `engram setup opencode`.",

		"error.opencode_missing": "OpenCode is not on PATH. Install it before continuing.",
		"error.config_parse":     "Could not read the OpenCode configuration: %s",
		"error.write":            "Could not write %s: %s",
		"error.unknown_command":  "Unknown command: %s",
		"error.no_tty":           "The wizard needs an interactive terminal. Use flags or --yes.",
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
