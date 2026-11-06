import { Plugin } from "@opencode/plugin"

/**
 * Server-side part of MeMo-AI.
 *
 * The orchestration rules live in agents, skills and commands. This plugin only
 * exposes a small status tool so the model can inspect what is installed without
 * shelling out.
 */
export default Plugin.define({
  id: "memo-ai",
  async setup(ctx) {
    const registration = await ctx.tool.transform((editor) => {
      editor.namespace({
        name: "memo",
        description: "MeMo-AI orchestration and memory helpers",
      })

      editor.add({
        name: "status",
        description:
          "Return MeMo-AI status: OpenCode version, active location, primary agent, " +
          "enabled subagents and whether Engram memory is configured.",
        input: {
          type: "object",
          properties: {},
          additionalProperties: false,
        },
        options: { namespace: "memo", codemode: true },
        execute: async () => {
          const agents = await ctx.agent.list()
          const servers = await ctx.mcp.list()

          const agentList = Array.isArray(agents)
            ? agents
            : Array.isArray((agents as { data?: unknown[] })?.data)
              ? ((agents as { data: unknown[] }).data)
              : []
          const serverList = Array.isArray(servers)
            ? servers
            : Array.isArray((servers as { data?: unknown[] })?.data)
              ? ((servers as { data: unknown[] }).data)
              : []

          const names = agentList
            .map((agent) => {
              const record = agent as { id?: string; name?: string; mode?: string }
              const id = record.id ?? record.name ?? "unknown"
              return record.mode ? `${id} (${record.mode})` : id
            })
            .sort()

          const memoNames = names.filter((name) => name.startsWith("memo"))
          const engram = serverList.some((server) => {
            const record = server as { name?: string; id?: string }
            return (record.name ?? record.id ?? "").toLowerCase().includes("engram")
          })

          return {
            content: [
              {
                type: "text" as const,
                text: [
                  "MeMo-AI is active.",
                  `OpenCode: ${ctx.app.version} (${ctx.app.channel})`,
                  `Location: ${ctx.location?.directory ?? "unknown"}`,
                  `MeMo agents: ${memoNames.length ? memoNames.join(", ") : "not installed"}`,
                  `Engram MCP: ${engram ? "configured" : "not configured"}`,
                  "Commands: /memo, /memo-recall, /memo-remember, /memo-handoff, /memo-review, /memo-agents",
                ].join("\n"),
              },
            ],
          }
        },
      })
    })

    return () => {
      void registration.dispose()
    }
  },
})
