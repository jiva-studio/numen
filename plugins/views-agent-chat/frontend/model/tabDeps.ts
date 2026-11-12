/**
 * What one agent tab requires of the window, supplied by the host's workspace and agent.
 */
import {
  formatErrorMessage,
  type AgentHost,
  type CommandsHost,
  type WorkspaceHost,
  type Caught,
} from '@numen/plugin-sdk'
import type { AgentTabDeps } from '../types'

/** The command of the window that opens the settings tab. */
const OPEN_SETTINGS = 'app:settings.open'

/** The host services an agent tab is made from. */
export interface AgentChatHost {
  readonly workspace: WorkspaceHost
  readonly agent: AgentHost
  readonly commands?: CommandsHost
  readonly getReasoning: () => string
}

/** What an agent tab requires of the window. */
export function createAgentTabDeps({
  workspace,
  agent,
  commands,
  getReasoning,
}: AgentChatHost): AgentTabDeps {
  return {
    openFileAt: (path, ...spans) => workspace.files.openAt(path, spans),
    openFileAdjacent: (path) => workspace.files.open(path, { placement: 'split' }),
    resolveLinks: (text) => agent.resolveLinks(text),
    unreachable: () => agent.getUnreachable(),
    settings: {
      writeSetting: (path, value) => agent.writeSetting(path, value),
      getSetting: (path) => agent.getSetting(path),
      getReasoning,
      getModelsAt: (path) => agent.getModelsAt(path),
      openSettings: () => {
        commands
          ?.run(OPEN_SETTINGS)
          .catch((error: Caught) => workspace.messages.showError(formatErrorMessage(error)))
      },
    },
  }
}
