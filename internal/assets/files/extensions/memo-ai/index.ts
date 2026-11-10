import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Box, Text } from "@earendil-works/pi-tui";
import { Type } from "typebox";

/**
 * MeMo-AI extension for Pi.
 *
 * The orchestration rules live in APPEND_SYSTEM.md, skills and prompt templates.
 * This extension only surfaces state: the startup mark, a working spinner, a
 * status badge, a `memo_status` tool the agent can call instead of guessing its
 * configuration, a `/memo-about` command and a reminder to hand off before the
 * context is compacted.
 */

// Installed at <agent dir>/extensions/memo-ai/index.ts; state lives two levels up.
const STATE_FILE = fileURLToPath(new URL("../../memo-ai/state.json", import.meta.url));

const LOGO_ENTRY = "memo-logo";
/** The mark is exactly this wide; every line below is padded to it. */
const LOGO_WIDTH = 28;
/** Braille spinner; Pi paints the frames with the active theme accent. */
const SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

type MemoState = {
	language?: string;
	updated_at?: string;
	orchestrator?: boolean;
	primary_model?: string;
	engram_mcp?: boolean;
	engram_path?: string;
};

type Rgb = { kind: "rgb"; r: number; g: number; b: number };

const rgb = (r: number, g: number, b: number): Rgb => ({ kind: "rgb", r, g, b });

/** MeMo palette (Vesper). Brand colors, independent of the active Pi theme. */
const WHITE = rgb(255, 255, 255);
const CREAM = rgb(255, 224, 200);
const PEACH = rgb(255, 199, 153);
const MINT = rgb(153, 255, 228);
const RULE = rgb(80, 80, 80);
const MUTED = rgb(160, 160, 160);

// 20 columns of letters + 5 of ring + 3 of accent = 28.
const LOGO_ROWS: ReadonlyArray<ReadonlyArray<{ text: string; color: Rgb }>> = [
	[
		{ text: "█▄ ▄█  █▀▀▀  █▄ ▄█  ", color: WHITE },
		{ text: "█▀▀▀█", color: PEACH },
		{ text: "  ▌", color: MINT },
	],
	[
		{ text: "█ ▀ █  █▀▀   █ ▀ █  ", color: CREAM },
		{ text: "█   █", color: PEACH },
		{ text: "  ▌", color: MINT },
	],
	[
		{ text: "█   █  █▄▄▄  █   █  ", color: PEACH },
		{ text: "█▄▄▄█", color: PEACH },
		{ text: "  ▌", color: MINT },
	],
];

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
	pi.registerEntryRenderer(LOGO_ENTRY, (_entry, _options, theme) => {
		const box = new Box(1, 0);
		for (const row of LOGO_ROWS) {
			const line = row.map((segment) => theme.style(segment.text, { fg: segment.color })).join("");
			box.addChild(new Text(line, 0, 0));
		}
		box.addChild(new Text(theme.style("─".repeat(LOGO_WIDTH), { fg: RULE }), 0, 0));
		const caption =
			theme.style(" M - M   ", { fg: MUTED }) +
			theme.style("A I", { fg: PEACH }) +
			theme.style(" · ORCHESTRATOR ", { fg: MUTED });
		box.addChild(new Text(caption, 0, 0));
		return box;
	});

	pi.on("session_start", async (_event, ctx) => {
		const state = readState();
		ctx.ui.setStatus("memo-ai", state?.engram_mcp ? "MEMO · memory" : "MEMO · no memory");
		ctx.ui.setWorkingIndicator({ frames: SPINNER_FRAMES, intervalMs: 80 });
		// Only a fresh session gets the mark; resumed ones already carry it.
		if (ctx.sessionManager.getEntries().length === 0) {
			pi.appendEntry(LOGO_ENTRY, {});
		}
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
