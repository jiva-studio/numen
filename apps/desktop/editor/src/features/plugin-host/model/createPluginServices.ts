/**
 * The services a plugin is given by the window, made from the parts of the window: the workspace,
 * and the notes and files of the vault.
 */
import type { Component } from 'vue'
import { getCurrentDate } from '@numen/ui'
import {
  createWindowHost,
  type VaultRelations,
  type WindowControls,
  type WindowHost,
} from '@numen/plugin-loader'
import type { FileApi } from '@/entities/file'
import { getNoteInFocus, type WindowTabs } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { formatLocalDate } from '../lib/localDate'
import { createIconHost, createVaultFiles, type VaultFilesDeps } from './fileServices'
import {
  createVaultNotes,
  createWorkspaceFiles,
  createWorkspaceHost,
  createWorkspaceMessages,
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
    today: () => formatLocalDate(getCurrentDate()),
  })
}

/** What the notes and files a plugin's tabs are given are made from. */
export interface VaultAccessDeps {
  readonly notes: VaultNotesDeps['notes']
  readonly files: Pick<FileApi, 'getFileKinds' | 'listFiles'>
  readonly relations: VaultRelations
  readonly search: VaultNotesDeps['search']
  readonly making: VaultNotesDeps['making']
  readonly pluginNotes: VaultNotesDeps['pluginNotes']
  readonly getTarget: VaultFilesDeps['getTarget']
  readonly runInvocation: VaultFilesDeps['runCommand']
}

/** The notes, the links and the files of the vault. */
export function createVaultAccess(
  deps: VaultAccessDeps,
): Pick<PluginServices, 'notes' | 'relations' | 'files'> {
  return {
    notes: createVaultNotes({
      notes: deps.notes,
      getFileKinds: deps.files.getFileKinds,
      search: deps.search,
      making: deps.making,
      pluginNotes: deps.pluginNotes,
    }),
    relations: deps.relations,
    files: createVaultFiles({
      listFiles: deps.files.listFiles,
      making: deps.making,
      getTarget: deps.getTarget,
      runCommand: deps.runInvocation,
    }),
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
