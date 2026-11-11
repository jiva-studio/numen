/**
 * The services a plugin is given by the window, made from the parts of the window: the workspace,
 * the notes and files of the vault, and the agent.
 */
import { getCurrentDate } from '@numen/ui'
import { relations } from '@/entities/note'
import { agentPort } from '@/entities/plugin'
import { getNoteInFocus, type WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import { formatLocalDate } from '../../lib/localDate'
import { createAgentAsker } from './createAgentAsker'
import { createVaultFiles, type VaultFilesDeps } from './fileServices'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceHost,
  type AgentHostDeps,
  type PluginServices,
  type VaultNotesDeps,
  type WorkspaceHostDeps,
} from './pluginServices'

/** The services the commands, the vaults, the window and the tasks of the window make instead. */
type WindowOwnedServices =
  'commands' | 'vaults' | 'window' | 'taskList' | 'backgroundTasks' | 'buffers' | 'noteBuffer'

/** What the services of the plugins are made from. */
export interface PluginServicesDeps {
  readonly log: Pick<MessageLog, 'getWriter'>
  readonly windowTabs: WindowTabs
  readonly search: VaultNotesDeps['search']
  readonly making: VaultNotesDeps['making']
  readonly pluginNotes: VaultNotesDeps['pluginNotes']
  readonly tabOpeners: WorkspaceHostDeps['tabOpeners']
  readonly tabSync: NonNullable<WorkspaceHostDeps['tabSync']>
  readonly window: WorkspaceHostDeps['window'] & AgentHostDeps['window']
  readonly settings: AgentHostDeps['settings']
  readonly getTarget: VaultFilesDeps['getTarget']
  readonly runInvocation: VaultFilesDeps['runCommand']
  readonly getPluginCommandIcon: NonNullable<WorkspaceHostDeps['getPluginCommandIcon']>
}

export function createPluginServices(
  deps: PluginServicesDeps,
): Omit<PluginServices, WindowOwnedServices> {
  const { log, windowTabs } = deps
  const writeMessage = log.getWriter('command')
  return {
    workspace: createWorkspaceHost({
      ...deps,
      dragged: windowTabs.dragged,
      getNoteInFocus: () => getNoteInFocus(windowTabs),
      ask: createAgentAsker(windowTabs, writeMessage),
      writeMessage,
      getMessageWriter: (name) => log.getWriter(name),
      today: () => formatLocalDate(getCurrentDate()),
    }),
    notes: createVaultNotes({
      search: deps.search,
      making: deps.making,
      pluginNotes: deps.pluginNotes,
    }),
    relations,
    files: createVaultFiles({
      making: deps.making,
      getTarget: deps.getTarget,
      runCommand: deps.runInvocation,
    }),
    agent: createAgentHost({ port: agentPort, window: deps.window, settings: deps.settings }),
  }
}
