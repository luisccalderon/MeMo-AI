import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

/**
 * MeMo-AI extension for oh-my-pi.
 *
 * The orchestration rules live in APPEND_SYSTEM.md, agents, skills and commands.
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
	subagents?: string[] | null;
	subagent_model?: string;
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
	const subagents = state.subagents?.length ? state.subagents.join(", ") : "disabled (work inline)";
	return [
		"MeMo-AI: your coding agent, bringing your changes, tasks and engineering workflow together.",
		`Orchestrator prompt: ${state.orchestrator ? "installed" : "not installed"}`,
		`Model set by MeMo-AI: ${state.primary_model || "none (omp default)"}`,
		`Subagents: ${subagents}`,
		`Subagent model: ${state.subagent_model || "inherited"}`,
		`Engram memory: ${state.engram_mcp ? `configured (${state.engram_path})` : "not configured"}`,
		"Commit mode: supervised unless the user declares an autonomous run",
		`Last configured: ${state.updated_at || "unknown"}`,
		"Commands: /memo /memo-changes /memo-tasks /memo-recall /memo-remember /memo-handoff /memo-review /memo-agents",
	].join("\n");
}

export default function memoAI(pi: ExtensionAPI) {
	const z = pi.zod;

	pi.setLabel("MeMo-AI");

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
			"Return the MeMo-AI configuration: orchestrator prompt, model, enabled subagents, " +
			"Engram memory and commit mode. Read-only. Call it before delegating to a memo-* agent " +
			"or when asked how MeMo-AI is set up.",
		parameters: z.object({}),
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
