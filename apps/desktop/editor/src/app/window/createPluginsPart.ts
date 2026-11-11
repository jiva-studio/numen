/**
 * The plugins of the window: the host and the loader, the services a plugin is given, and the words
 * a plugin's errors are reported in.
 */
import { createWindowHost, SLOT, type PluginHost, type PluginStatus } from '@numen/plugin-loader'
import { formatError } from '@/shared/api/errors'
import { createNoticeQueue } from '@numen/workbench'
import { getFileKinds } from '@/entities/file'
import { listDeclaredItemLists, runDeclaredHandler } from '@/entities/plugin'
import { readNote, resolveNameLink, writeNote } from '@/entities/note'
import type { SettingsStore } from '@/entities/settings'
import {
  createBackendHandlers,
  createCommandItemLists,
  createDeclaredState,
  createHostPort,
  createPluginCatalog,
  createPluginNotes,
  createPluginReports,
  createPluginSupport,
  createWindowCommands,
  createWindowVaults,
  createNoteKindReader,
  getAppCommandIds,
  useDeclaredCommands,
} from '@/features/plugin-commands'
import { getWindowConfig } from '@/features/settings-commands'
import { createVaultOpener } from '@/features/command-palette'
import { createWindowSlots } from '../plugins/windowSlots'
import type { createEditorCommands, createWindowKinds } from './createEditorCommands'
import type { WindowServices } from './createWindowServices'
import type { PluginsCore, TabsCore, WindowCore } from './windowCore'

interface PluginsCoreDeps {
  readonly log: WindowCore['log']
  readonly writeMessage: WindowCore['writeMessage']
  readonly frame: WindowCore['frame']
  readonly windowTabs: TabsCore['windowTabs']
  readonly tabOpeners: TabsCore['tabOpeners']
  readonly rest: SettingsStore
  readonly status: PluginStatus
  readonly registry: WindowCore['registry']
}

export function createPluginsCore(deps: PluginsCoreDeps): PluginsCore {
  const { log, writeMessage, frame, windowTabs, tabOpeners, rest, status, registry } = deps
  const catalog = createPluginCatalog(rest)
  const reports = createPluginReports(log, catalog.getName)
  const hostPort = createHostPort()
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: readNote, write: writeNote })
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: getFileKinds,
  })
  const declared = createDeclaredState({ getConfig: () => getWindowConfig(rest) })
  const slots = createWindowSlots({
    declaredCommands: declared,
    facts: hostPort,
    writeIssue: reports.writeIssue,
  })
  const { pluginHost, pluginLoader } = createPluginSupport({
    slots: slots.hostDeps,
    getCommandDeclarations: () => declared.declarations.value,
    handlers: createBackendHandlers({
      getHandlers: () => declared.contributions.handlers.value,
      list: itemLists.listItems,
      run: runDeclaredHandler,
      writeMessage,
      openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
    }),
    windowTabs,
    tabOpeners,
    rest,
    status,
    getPlatform: () => frame.platform.value,
    getServices: hostPort.getServices,
    notices: pluginNotices,
    reportViewIssue: reports.reportViewIssue,
    reportCommandIssue: reports.reportCommandIssue,
    reportError: reports.reportError,
    reportLoadError: reports.reportLoadError,
    ...pluginNotes,
    resolveLink: resolveNameLink,
  })
  const declaredCommands = useDeclaredCommands({
    state: declared,
    registry,
    isListed: (id) => pluginHost.commands.getPaletteCommands().some((one) => one.id === id),
    hasBackend: catalog.hasBackend,
    listItems: itemLists.listItems,
  })
  return {
    declaredCommands,
    itemLists,
    host: { pluginHost, pluginLoader },
    hostPort,
    pluginNotices,
    pluginNotes,
  }
}

export function createWindowFrameServices(
  base: WindowCore,
  services: Pick<WindowServices, 'vaults'>,
  commands: Pick<
    ReturnType<typeof createEditorCommands>,
    'getTarget' | 'featureCommands' | 'commands' | 'facts'
  >,
  kinds: Pick<ReturnType<typeof createWindowKinds>, 'services'>,
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
  base.plugins.hostPort.open({
    services: {
      ...kinds.services,
      commands: windowCommands,
      vaults: createWindowVaults({
        vaultList: services.vaults.vaultList,
        openVault: createVaultOpener(commands.commands.commandDeps, commands.getTarget),
      }),
      window: createWindowHost(frame, (error) => base.writeMessage(formatError(error), 'error')),
    },
    facts: commands.facts,
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
