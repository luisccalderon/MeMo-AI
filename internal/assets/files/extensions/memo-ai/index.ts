import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

/**
 * MeMo-AI extension for Pi.
 *
 * The orchestration rules live in APPEND_SYSTEM.md, skills and prompt templates.
 * This extension only surfaces state: a status badge, a `memo_status` tool the
 * agent can call instead of guessing its configuration, a `/memo-about` command
 * and a reminder to hand off before the context is compacted.
 */

// Installed at <agent dir>/extensions/memo-ai/index.ts; state lives two levels up.
const STATE_FILE = fileURLToPath(new URL("../../memo-ai/state.json", import.meta.url));

type MemoState = {
	language?: string;
	updated_at?: string;
	orchestrator?: boolean;
	primary_model?: string;
	engram_mcp?: boolean;
	engram_path?: string;
};

function readState(): MemoState | undefined {
	try {
		return JSON.parse(readFileSync(STATE_FILE, "utf8")) as MemoState;
	} catch {
		return undefined;
	}
}

function describe(state: MemoState | undefined): string {
	if (!state) {
		return "MeMo-AI is not configured. Run `memo-ai install`.";
	}
	return [
		"MeMo-AI: your coding agent, bringing your changes, tasks and engineering workflow together.",
		`Orchestrator prompt: ${state.orchestrator ? "installed" : "not installed"}`,
		`Model set by MeMo-AI: ${state.primary_model || "none (Pi default)"}`,
		"Execution: inline (Pi has no subagents yet)",
		`Engram memory: ${state.engram_mcp ? `configured (${state.engram_path})` : "not configured"}`,
		"Commit mode: supervised unless the user declares an autonomous run",
		`Last configured: ${state.updated_at || "unknown"}`,
		"Commands: /memo /memo-changes /memo-tasks /memo-recall /memo-remember /memo-handoff /memo-review",
	].join("\n");
}

export default function memoAI(pi: ExtensionAPI) {
	pi.on("session_start", async (_event, ctx) => {
		const state = readState();
		ctx.ui.setStatus("memo-ai", state?.engram_mcp ? "MEMO · memory" : "MEMO");
	});

	pi.on("session_before_compact", async (_event, ctx) => {
		ctx.ui.notify("MeMo-AI: context is being compacted. Run /memo-handoff to keep a resumable record.", "info");
	});

	pi.registerTool({
		name: "memo_status",
		label: "MeMo Status",
		description:
			"Return the MeMo-AI configuration: orchestrator prompt, model, Engram memory and commit mode. " +
			"Read-only. Call it when asked how MeMo-AI is set up instead of guessing.",
		parameters: Type.Object({}),
		async execute() {
			const state = readState();
			return {
				content: [{ type: "text", text: describe(state) }],
				details: { configured: state !== undefined, state: state ?? null },
			};
		},
	});

	pi.registerCommand("memo-about", {
		description: "Show the MeMo-AI configuration",
		handler: async (_args, ctx) => {
			ctx.ui.notify(describe(readState()), "info");
		},
	});
}
