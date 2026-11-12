import type { AgentHost } from '@numen/plugin-sdk'

const isAgentPath = (path: readonly string[]): boolean => path[0] === 'app' && path[1] === 'agent'

/** The agent with the settings it reads, writes and lists limited to its own. */
export function limitAgentSettings(agent: AgentHost): AgentHost {
  return {
    ask: (query) => agent.ask(query),
    finish: (conversation) => agent.finish(conversation),
    resolveLinks: (text, vault) => agent.resolveLinks(text, vault),
    getUnreachable: () => agent.getUnreachable(),
    getSetting: (path) => (isAgentPath(path) ? agent.getSetting(path) : undefined),
    writeSetting: async (path, value) => {
      if (!isAgentPath(path)) {
        throw new Error(
          `agent.writeSetting is limited to the agent settings, not "${path.join('.')}"`,
        )
      }
      await agent.writeSetting(path, value)
    },
    getModelsAt: (path) => (isAgentPath(path) ? agent.getModelsAt(path) : []),
  }
}
