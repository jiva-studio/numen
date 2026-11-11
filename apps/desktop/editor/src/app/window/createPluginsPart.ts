/**
 * The plugins of the window: the host and the loader, the services a plugin is given, and the words
 * a plugin's errors are reported in.
 */
import { createWindowHost, SLOT, type PluginHost } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { formatError } from '@/shared/api/errors'
import type { Command } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'
import { readNote, resolveNameLink, writeNote } from '@/entities/note'
import { runDeclaredHandler } from '@/entities/plugin'
import { createWindowBackgroundTasks } from '@/features/background-tasks'
import { createInvocationRunner, createVaultOpener, searchCore } from '@/features/command-palette'
import { createBufferHost } from '@/features/file-conflict'
import {
  createBackendHandlers,
  createCommandIconFinder,
  createDeclaredState,
  createHostPort,
  createNoteKindReader,
  createPluginCatalog,
  createPluginNotes,
  createPluginReports,
  createPluginServices,
  createPluginSupport,
  createWindowCommands,
  createWindowFacts,
  createWindowVaults,
  getAppCommandIds,
  useDeclaredCommands,
  type CommandItemLists,
  type PluginServices,
} from '@/features/plugin-commands'
import { getWindowConfig } from '@/features/settings-commands'
import { createWindowSlots } from '../plugins/windowSlots'
import type { createEditorCommands } from './createEditorCommands'
import type { VaultServices } from './createWindowServices'
import type { PluginsCore, WindowCore, WindowFoundation } from './windowCore'

interface PluginsCoreDeps {
  readonly foundation: WindowFoundation
  readonly vaultServices: VaultServices
  readonly itemLists: CommandItemLists
  readonly getTarget: () => CommandTarget
  readonly getAppCommands: () => readonly Command[]
}

function createPluginBackgroundTasks(
  deps: PluginsCoreDeps,
  pluginNotices: ReturnType<typeof createNoticeQueue>,
) {
  const { log, writeMessage, settingsRequests } = deps.foundation
  const { window, windowSettings } = deps.vaultServices
  return createWindowBackgroundTasks({
    log,
    plugins: pluginNotices,
    window: {
      lost: window.health.lost,
      unwatched: window.vault.unwatched,
      error: window.vault.error,
      tasks: window.tasks,
    },
    settings: windowSettings,
    writeMessage,
    openSettings: (pluginId) => settingsRequests.emit({ pluginId, filter: undefined }),
  })
}

function createWindowBackendHandlers(
  deps: PluginsCoreDeps,
  declared: ReturnType<typeof createDeclaredState>,
) {
  const { writeMessage, tabs } = deps.foundation
  return createBackendHandlers({
    getHandlers: () => declared.contributions.handlers.value,
    list: deps.itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage,
    openFile: (path) => tabs.tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })
}

export function createPluginsCore(deps: PluginsCoreDeps): PluginsCore {
  const { foundation, vaultServices, itemLists, getTarget, getAppCommands } = deps
  const { log, writeMessage, frame, rest, status, registry } = foundation
  const { windowTabs, tabOpeners } = foundation.tabs
  const { editing, tabSync, window, vaults } = vaultServices
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: readNote, write: writeNote })
  const catalog = createPluginCatalog(rest)
  const reports = createPluginReports(log, catalog.getName)
  const hostPort = createHostPort()
  const declared = createDeclaredState({ getConfig: () => getWindowConfig(rest) })
  const facts = createWindowFacts({ getAppCommands, getTarget, isPaletteOpen: hostPort.isPaletteOpen })
  const slots = createWindowSlots({ declaredCommands: declared, facts, writeIssue: reports.writeIssue })
  const { pluginHost, pluginLoader } = createPluginSupport({
    slots: slots.hostDeps,
    getCommandDeclarations: () => declared.declarations.value,
    handlers: createWindowBackendHandlers(deps, declared),
    windowTabs,
    tabOpeners,
    rest,
    status,
    getPlatform: () => frame.platform.value,
    getServices: () => services,
    notices: pluginNotices,
    ...reports,
    ...pluginNotes,
    resolveLink: resolveNameLink,
  })
  vaultServices.vaultChanges.subscribe((event) => pluginHost.notifyVaultChange(event))
  const declaredCommands = useDeclaredCommands({
    state: declared,
    registry,
    isListed: (id) => pluginHost.commands.getPaletteCommands().some((one) => one.id === id),
    hasBackend: catalog.hasBackend,
    listItems: itemLists.listItems,
  })
  const pluginServices = createPluginServices({
    log,
    windowTabs,
    search: searchCore,
    making: editing.making,
    pluginNotes,
    tabOpeners,
    tabSync,
    window,
    settings: { rest },
    getTarget,
    runInvocation: hostPort.runInvocation,
    getPluginCommandIcon: createCommandIconFinder(pluginHost, declaredCommands),
  })
  const services: PluginServices = {
    ...pluginServices,
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
    noteBuffer: editing.notes,
    taskList: window.tasks,
    backgroundTasks: createPluginBackgroundTasks(deps, pluginNotices),
    commands: hostPort.commands,
    vaults: createWindowVaults({ vaultList: vaults.vaultList, openVault: hostPort.openVault }),
    window: createWindowHost(frame, (error) => writeMessage(formatError(error), 'error')),
  }
  return {
    declaredCommands,
    itemLists,
    host: { pluginHost, pluginLoader },
    hostPort,
    facts,
    services,
    pluginNotices,
    pluginNotes,
  }
}

export function createWindowFrameServices(
  base: WindowCore,
  commands: Pick<
    ReturnType<typeof createEditorCommands>,
    'getTarget' | 'featureCommands' | 'commands'
  >,
) {
  const { frame } = base
  const { items, runCommandToEnd } = commands.commands.pluginRuns
  const appIds = getAppCommandIds(commands.featureCommands.commands)
  const windowCommands = createWindowCommands({
    getAppIds: () => appIds,
    getPluginIds: () =>
      base.plugins.host.pluginHost.commands.getPaletteCommands().map((one) => one.id),
    runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget: commands.getTarget,
  })
  base.plugins.hostPort.connect({
    commands: windowCommands,
    isPaletteOpen: () =>
      commands.commands.palette.open.value || commands.commands.commands.palette.open.value,
    runInvocation: createInvocationRunner(() => commands.commands.commandDeps),
    openVault: createVaultOpener(commands.commands.commandDeps, commands.getTarget),
  })
  return {
    frame,
    slots: base.plugins.host.pluginHost.slots,
    windowCommands,
    views: getPluginViews(base.plugins.host.pluginHost),
    readNoteKind: createNoteKindReader(base.plugins.itemLists),
  }
}

const getPluginViews = (pluginHost: PluginHost) => ({
  titleBarView: pluginHost.slotViews.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.slotViews.getSlotView(SLOT.statusBar),
})
