/**
 * The services a plugin is given by the window, made from the parts of the window: the workspace,
 * the notes and files of the vault, and the agent.
 */
import type { Component } from 'vue'
import { getCurrentDate } from '@numen/ui'
import { createWindowHost, type WindowControls, type WindowHost } from '@numen/plugin-loader'
import { relations } from '@/entities/note'
import { agentPort } from '@/entities/plugin'
import { getNoteInFocus, type WindowTabs } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { formatLocalDate } from '../../lib/localDate'
import { createAgentAsker } from './createAgentAsker'
import { createIconHost, createVaultFiles, type VaultFilesDeps } from './fileServices'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceFiles,
  createWorkspaceHost,
  createWorkspaceMessages,
  type AgentHostDeps,
  type PluginServices,
  type VaultNotesDeps,
  type WorkspaceFilesDeps,
  type WorkspaceHostDeps,
} from './pluginServices'

/** What the workspace a plugin's tabs are given is made from. */
export interface WorkspaceServiceDeps {
  readonly log: Pick<MessageLog, 'getWriter'>
  readonly windowTabs: WindowTabs
  readonly tabOpeners: WorkspaceFilesDeps['tabOpeners'] & {
    readonly findFileIcon: (path: string) => Component | undefined
  }
  readonly tabSync: NonNullable<WorkspaceFilesDeps['tabSync']>
  readonly window: WorkspaceHostDeps['window']
  readonly getPluginCommandIcon: (id: string) => Component | undefined
}

/** The workspace of the window: its tabs, files, icons and messages. */
export function createWorkspaceService(deps: WorkspaceServiceDeps): PluginServices['workspace'] {
  const { log, windowTabs } = deps
  const writeMessage = log.getWriter('command')
  return createWorkspaceHost({
    window: deps.window,
    dragged: windowTabs.dragged,
    files: createWorkspaceFiles({
      tabOpeners: deps.tabOpeners,
      tabSync: deps.tabSync,
      writeMessage,
    }),
    icons: createIconHost({
      getPluginCommandIcon: deps.getPluginCommandIcon,
      findFileIcon: deps.tabOpeners.findFileIcon,
    }),
    messages: createWorkspaceMessages({
      writeMessage,
      getMessageWriter: (name) => log.getWriter(name),
    }),
    getNoteInFocus: () => getNoteInFocus(windowTabs),
    ask: createAgentAsker(windowTabs, writeMessage),
    today: () => formatLocalDate(getCurrentDate()),
  })
}

/** What the notes, files and agent a plugin's tabs are given are made from. */
export interface VaultAccessDeps {
  readonly search: VaultNotesDeps['search']
  readonly making: VaultNotesDeps['making']
  readonly pluginNotes: VaultNotesDeps['pluginNotes']
  readonly window: AgentHostDeps['window']
  readonly settings: AgentHostDeps['settings']
  readonly getTarget: VaultFilesDeps['getTarget']
  readonly runInvocation: VaultFilesDeps['runCommand']
}

/** The notes, the links, the files and the agent of the vault. */
export function createVaultAccess(
  deps: VaultAccessDeps,
): Pick<PluginServices, 'notes' | 'relations' | 'files' | 'agent'> {
  return {
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

/** What the window a plugin is given is made from. */
export interface WindowServiceDeps {
  readonly frame: WindowControls
  readonly writeMessage: MessageWriter
}

/** The window of the application as a plugin reaches it; a failure of the frame is written as a message. */
export function createWindowService(deps: WindowServiceDeps): WindowHost {
  return createWindowHost(deps.frame, (error) => deps.writeMessage(formatError(error), 'error'))
}
