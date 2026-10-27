/**
 * What one agent tab asks of the window, answered by the host's workspace and agent.
 */
import type { AgentHost, WorkspaceHost } from '@numen/plugin-sdk'
import type { AgentTabDeps } from '../types'

/** The host services an agent tab is made from. */
export interface AgentChatHost {
  readonly workspace: WorkspaceHost
  readonly agent: AgentHost
}

/** What an agent tab asks of the window. */
export function createAgentTabDeps({ workspace, agent }: AgentChatHost): AgentTabDeps {
  return {
    openFileAt: (path, ...spans) => workspace.openFileAt(path, spans),
    openFileBeside: (path) => workspace.openFile(path, { placement: 'split' }),
    resolveLinks: (text) => agent.resolveLinks(text),
    unreachable: () => agent.getUnreachable(),
    writeSetting: (path, value) => agent.writeSetting(path, value),
    getSetting: (path) => agent.getSetting(path),
    getModelsAt: (path) => agent.getModelsAt(path),
  }
}
