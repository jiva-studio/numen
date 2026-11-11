/**
 * The plugins of the window: what the host reads of the window, the services a plugin is given
 * that the window owns, and the background tasks a plugin lists.
 */
import type { Component, Ref } from 'vue'
import type { PluginStatus, WindowControls } from '@numen/plugin-loader'
import type { NoticeQueue, WindowPlatform } from '@numen/workbench'
import type { Command } from '@/entities/command'
import type { CommandItemLists } from '@/features/plugin-commands'
import type { CommandTarget } from '@/shared/model/command'
import { resolveNameLink } from '@/entities/note'
import { runDeclaredHandler } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { FileOpeners, WindowTabs } from '@/entities/tab'
import { createWindowBackgroundTasks } from '@/features/background-tasks'
import { searchCore } from '@/features/command-palette'
import { createBufferHost } from '@/features/file-conflict'
import {
  createBackendHandlers,
  createDeclaredState,
  createPluginCatalog,
  createNoticeParts,
  createPluginReports,
  createPluginSupport,
  createVaultAccess,
  createWorkspaceService,
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
  type PluginNotes,
  type PluginServices,
  createWindowFacts,
  createWindowService,
  createWindowSlots,
  createWindowVaults,
  type HostPort,
  type HostParts,
} from '@/features/plugin-commands'
import { getWindowConfig } from '@/features/settings-commands'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import type { WindowFoundation } from './createWindowFoundation'
import type { WindowVault } from './createWindowVault'
import type { createVaultServices } from './createVaultServices'

type VaultServices = ReturnType<typeof createVaultServices>

/** What the plugins read of the window before the host exists. */
export interface PluginDeclarationsDeps {
  readonly rest: SettingsStore
  readonly log: MessageLog
  readonly writeMessage: MessageWriter
  readonly hostPort: Pick<HostPort, 'isPaletteOpen'>
  readonly getTarget: () => CommandTarget
  readonly appCommands: readonly Command[]
  readonly itemLists: CommandItemLists
  readonly tabOpeners: Pick<FileOpeners, 'openFile'>
}

/**
 * The catalog and the reports of the plugins, the commands they declare, and the parts of the host
 * that read them: the slots and the handlers of the backends.
 */
export function createPluginDeclarations(deps: PluginDeclarationsDeps) {
  const { rest, log, itemLists, tabOpeners } = deps
  const catalog = createPluginCatalog(rest)
  const reports = createPluginReports(log, catalog.getName)
  const declared = createDeclaredState({ getConfig: () => getWindowConfig(rest) })
  const facts = createWindowFacts({
    appCommands: deps.appCommands,
    getTarget: deps.getTarget,
    isPaletteOpen: deps.hostPort.isPaletteOpen,
  })
  const slots = createWindowSlots({
    declaredCommands: declared,
    facts,
    writeIssue: reports.writeIssue,
  })
  const handlers = createBackendHandlers({
    getHandlers: () => declared.contributions.handlers.value,
    list: itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage: deps.writeMessage,
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })
  const hostParts: readonly HostParts[] = [
    slots.hostDeps,
    handlers,
    {
      getCommandDeclarations: () => declared.declarations.value,
      reportCommandIssue: reports.reportCommandIssue,
      reportViewIssue: reports.reportViewIssue,
    },
  ]
  return { catalog, reports, declared, hostParts }
}

/** What the background tasks of the plugins read of the window. */
export interface PluginBackgroundTasksDeps {
  readonly log: MessageLog
  readonly notices: NoticeQueue
  readonly window: Pick<VaultServices['window'], 'health' | 'vault' | 'tasks'>
  readonly windowSettings: { readonly appearance: { readonly state: { readonly lost: Readonly<Ref<string>> } } }
  readonly writeMessage: MessageWriter
  readonly openSettings: (pluginId?: string) => void
}

/** The tasks of the window and the failures of the plugins, as the background tasks a plugin lists. */
export function createPluginBackgroundTasks(deps: PluginBackgroundTasksDeps) {
  const { window } = deps
  return createWindowBackgroundTasks({
    log: deps.log,
    plugins: deps.notices,
    window: {
      lost: window.health.lost,
      unwatched: window.vault.unwatched,
      error: window.vault.error,
      tasks: window.tasks,
    },
    settings: deps.windowSettings,
    writeMessage: deps.writeMessage,
    openSettings: deps.openSettings,
  })
}

/** What the services a plugin is given that the window owns are made from. */
export interface WindowOwnedServicesDeps {
  readonly editing: Pick<VaultServices['editing'], 'fileFlush' | 'notes'>
  readonly window: Pick<VaultServices['window'], 'tasks'>
  readonly vaults: Pick<VaultServices['vaults'], 'vaultList'>
  readonly hostPort: Pick<HostPort, 'commands' | 'openVault'>
  readonly backgroundTasks: ReturnType<typeof createPluginBackgroundTasks>
  readonly frame: WindowControls
  readonly writeMessage: MessageWriter
}

/** The buffers, the tasks, the commands, the vaults and the window a plugin is given. */
export function createWindowOwnedServices(deps: WindowOwnedServicesDeps) {
  const { editing, window, vaults, hostPort } = deps
  return {
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
    noteBuffer: editing.notes,
    taskList: window.tasks,
    backgroundTasks: deps.backgroundTasks,
    commands: hostPort.commands,
    vaults: createWindowVaults({ vaultList: vaults.vaultList, openVault: hostPort.openVault }),
    window: createWindowService({ frame: deps.frame, writeMessage: deps.writeMessage }),
  }
}

/** What the services a plugin is given are read through: the notes, the notices and the services. */
export interface PluginServicesAccess {
  readonly getServices: () => PluginServices
  readonly notices: NoticeQueue
  readonly pluginNotes: PluginNotes
}

/** What the plugin host reads of the window. */
export interface PluginHostSupportDeps {
  readonly rest: SettingsStore
  readonly status: PluginStatus
  readonly getPlatform: () => WindowPlatform
  readonly windowTabs: WindowTabs
  readonly tabOpeners: FileOpeners
  readonly declarations: ReturnType<typeof createPluginDeclarations>
  readonly access: PluginServicesAccess
}

/** The plugin host and the loader, over the tabs, the services and the declarations of the window. */
export function createPluginHostSupport(deps: PluginHostSupportDeps) {
  const { windowTabs, tabOpeners, declarations, access } = deps
  const { getServices, pluginNotes } = access
  return createPluginSupport({
    rest: deps.rest,
    status: deps.status,
    getPlatform: deps.getPlatform,
    reportError: declarations.reports.reportError,
    reportLoadError: declarations.reports.reportLoadError,
    kinds: createTabKindParts({ windowTabs, tabOpeners }),
    tabs: createTabActionParts({ windowTabs }),
    parts: [
      createVaultServiceParts({ getServices }),
      createTaskServiceParts({ getServices }),
      createNoticeParts({ queue: access.notices }),
      {
        readNote: pluginNotes.readNote,
        writeNote: pluginNotes.writeNote,
        resolveLink: resolveNameLink,
      },
      ...declarations.hostParts,
    ],
  })
}

/** What the workspace and the vault services a plugin is given are made from. */
export interface VaultFacingServicesDeps {
  readonly foundation: Pick<WindowFoundation, 'log' | 'rest'>
  readonly tabsPart: Pick<WindowVault['tabsPart'], 'windowTabs' | 'tabOpeners'>
  readonly vault: Pick<VaultServices, 'tabSync' | 'editing' | 'window'>
  readonly pluginNotes: PluginNotes
  readonly getTarget: () => CommandTarget
  readonly runInvocation: HostPort['runInvocation']
  readonly getPluginCommandIcon: (id: string) => Component | undefined
}

/** The workspace, the notes, the links, the files and the agent a plugin is given. */
export function createVaultFacingServices(deps: VaultFacingServicesDeps) {
  const { foundation, tabsPart, vault } = deps
  const access = createVaultAccess({
    search: searchCore,
    making: vault.editing.making,
    pluginNotes: deps.pluginNotes,
    window: vault.window,
    settings: { rest: foundation.rest },
    getTarget: deps.getTarget,
    runInvocation: deps.runInvocation,
  })
  return {
    workspace: createWorkspaceService({
      log: foundation.log,
      windowTabs: tabsPart.windowTabs,
      tabOpeners: tabsPart.tabOpeners,
      tabSync: vault.tabSync,
      window: vault.window,
      getPluginCommandIcon: deps.getPluginCommandIcon,
    }),
    notes: access.notes,
    relations: access.relations,
    files: access.files,
    agent: access.agent,
  }
}
