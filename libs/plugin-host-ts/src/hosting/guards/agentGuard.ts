import type { AgentHost } from '@numen/plugin-sdk'
import { checkPermission, type PermissionTarget } from './permission'

const isAgentPath = (path: readonly string[]): boolean => path[0] === 'app' && path[1] === 'agent'

export function guardAgent(
  agent: AgentHost,
  requirePerm: (permission: string, operation: string) => void,
): AgentHost {
  return {
    ...agent,
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
    ask: (query) => {
      requirePerm('vault:read', 'agent.ask')
      return agent.ask(query)
    },
    finish: (conversation) => {
      requirePerm('vault:read', 'agent.finish')
      return agent.finish(conversation)
    },
    resolveLinks: (text, vault) => {
      requirePerm('vault:read', 'agent.resolveLinks')
      return agent.resolveLinks(text, vault)
    },
  }
}

/** The agent of a plugin, whose calls need the permissions the manifest of the plugin declares. */
export function createPluginAgent(agent: AgentHost, target: PermissionTarget): AgentHost {
  return guardAgent(agent, (permission, operation) =>
    checkPermission(target, permission, operation),
  )
}
