/** @jsxImportSource @opentui/solid */
import { Plugin } from "@opencode/plugin/tui"

// Paleta Vesper
const C = {
  white: "#FFFFFF",
  cream: "#FFE0C8",
  peach: "#FFC799",
  mint: "#99FFE4",
  muted: "#A0A0A0",
  rule: "#505050",
}

const letters = [
  "█▄ ▄█  █▀▀▀  █▄ ▄█  ",
  "█ ▀ █  █▀▀   █ ▀ █  ",
  "█   █  █▄▄▄  █   █  ",
]
const ring = ["█▀▀▀█", "█   █", "█▄▄▄█"]
const shade = [C.white, C.cream, C.peach] // blanco → durazno, de arriba a abajo
const WIDTH = 28

const Logo = () => (
  <box flexDirection="column" alignItems="center">
    <text> </text>
    {letters.map((l, i) => (
      <box flexDirection="row">
        <text fg={shade[i]}>{l}</text>
        <text fg={C.peach}>{ring[i]}</text>
        <text fg={C.mint}>{"  ▌"}</text>
      </box>
    ))}
    <text fg={C.rule}>{"─".repeat(WIDTH)}</text>
    <box flexDirection="row">
      <text fg={C.muted}>{"M - M   "}</text>
      <text fg={C.peach}>{"A I"}</text>
    </box>
  </box>
)

/**
 * TUI part of MeMo-AI.
 *
 * The logo lives in the home footer. The palette palette command is intentionally
 * side-effect free: it only shows a short orientation toast.
 */
export default Plugin.define({
  id: "memo-ai.tui",
  setup(context) {
    const disposeLogo = context.ui.slot({
      prepend: "home.footer",
      render: () => <Logo />,
    })

    const disposeBadge = context.ui.slot({
      append: "home.footer.status",
      render: () => <text fg={C.mint}>MEMO</text>,
    })

    context.keymap.layer(() => ({
      mode: "global",
      priority: 10,
      commands: [
        {
          id: "memo-ai.about",
          title: "MeMo-AI · About",
          group: "MeMo-AI",
          palette: true,
          slash: { name: "memo-about" },
          run: () =>
            context.ui.toast.show({
              title: "MeMo-AI",
              message: "Memory · Orchestration · Evidence. Try /memo, /memo-recall or /memo-handoff.",
              variant: "info",
              duration: 6000,
            }),
        },
      ],
    }))

    return () => {
      if (typeof disposeLogo === "function") disposeLogo()
      if (typeof disposeBadge === "function") disposeBadge()
    }
  },
})
