/**
 * What one agent tab requires of the window, supplied by the host's workspace and agent.
 */
import type { AgentHost, WorkspaceHost } from '@numen/plugin-sdk'
import type { AgentTabDeps } from '../types'

/** The command of the window that opens the settings tab. */
const OPEN_SETTINGS = 'app:settings.open'

/** The host services an agent tab is made from. */
export interface AgentChatHost {
  readonly workspace: WorkspaceHost
  readonly agent: AgentHost
  /** The depth of reasoning the plugin setting `reasoning` keeps. */
  readonly getReasoning: () => string
}

/** What an agent tab requires of the window. */
export function createAgentTabDeps({
  workspace,
  agent,
  getReasoning,
}: AgentChatHost): AgentTabDeps {
  return {
    openFileAt: (path, ...spans) => workspace.openFileAt(path, spans),
    openFileAdjacent: (path) => workspace.openFile(path, { placement: 'split' }),
    resolveLinks: (text) => agent.resolveLinks(text),
    unreachable: () => agent.getUnreachable(),
    writeSetting: (path, value) => agent.writeSetting(path, value),
    getSetting: (path) => agent.getSetting(path),
    getReasoning,
    getModelsAt: (path) => agent.getModelsAt(path),
    openSettings: () => workspace.runCommand(OPEN_SETTINGS, workspace.getNoteInFocus()),
  }
}
