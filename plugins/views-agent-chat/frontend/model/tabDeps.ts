/**
 * What one agent tab requires of the window, supplied by the host's workspace and the links of the
 * vault.
 */
import type { VaultRelations, WorkspaceHost } from '@numen/plugin-sdk'
import type { AgentTabDeps } from '../types'

/** The host services an agent tab is made from. */
export interface AgentChatHost {
  readonly workspace: WorkspaceHost
  readonly relations: Pick<VaultRelations, 'resolveLinks'>
  readonly openSettings: () => void
}

/** What an agent tab requires of the window. */
export function createAgentTabDeps({
  workspace,
  relations,
  openSettings,
}: AgentChatHost): AgentTabDeps {
  return {
    openFile: (path) => workspace.files.open(path),
    openFileAt: (path, ...spans) => workspace.files.openAt(path, spans),
    openFileAdjacent: (path) => workspace.files.open(path, { placement: 'split' }),
    resolveLinks: (text) => relations.resolveLinks(text),
    openSettings,
  }
}
