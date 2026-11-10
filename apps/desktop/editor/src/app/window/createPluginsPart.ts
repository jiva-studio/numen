/**
 * The plugins of the window: the host and the loader, the services a plugin is given, and the words
 * a plugin's errors are reported in.
 */
import { formatError, type Caught } from '@/shared/api/errors'
import { createWindowHost, SLOT, type PluginHost, type PluginStatus } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { listDeclaredItemLists, PLUGIN_FAILURE_WORDS, runDeclaredHandler } from '@/entities/plugin'
import type { fileOpeners } from '@/entities/tab'
import { getFileKinds } from '@/entities/file'
import { readNote, resolveLinks, writeNote } from '@/entities/note'
import type { createMessageLog } from '@/shared/model/notices/messages'
import {
  createBackendHandlers,
  createHostPort,
  createPluginNotes,
  createWindowCommands,
  createCommandItemLists,
  createDeclaredState,
  type DeclaredState,
  useDeclaredCommands,
} from '@/features/plugin-commands'
import { createPluginSupport } from '../plugins/pluginSupport'
import type { createEditorCommands, createWindowKinds } from './createEditorCommands'
import type { PluginsCore, TabsCore, WindowCore } from './windowCore'
import type { WindowServices } from './createWindowServices'
import { getSettingsConfig } from '@/features/settings-commands'
import type { SettingsStore } from '@/entities/settings'
import { createWindowSlots } from '../plugins/windowSlots'
import { createEditorVaults } from '../plugins/windowVaults'

const hasPluginBackend = (rest: SettingsStore, pluginId: string): boolean =>
  rest.state.plugins.value.find((one) => one.id === pluginId)?.hasBackend ?? true

const getWindowConfig = (rest: SettingsStore) =>
  getSettingsConfig([
    { id: 'app', name: '', settings: rest.state.app.value },
    ...rest.state.plugins.value.map((one) => ({
      id: one.id,
      name: one.name,
      settings: one.settings,
    })),
  ])

function createPluginReports(
  log: ReturnType<typeof createMessageLog>,
  getPluginName: (pluginId: string) => string,
) {
  const write = (pluginId: string, text: string, severity: 'warning' | 'error') =>
    log.getWriter(`plugin:${pluginId}`)(text, severity)
  return {
    reportViewIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'error'),
    reportCommandIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'warning'),
    reportError: (pluginId: string, error: Caught) =>
      write(
        pluginId,
        PLUGIN_FAILURE_WORDS.error(getPluginName(pluginId), formatError(error)),
        'error',
      ),
    reportLoadError: (pluginId: string, error: Caught) =>
      write(
        pluginId,
        PLUGIN_FAILURE_WORDS.stopped(
          getPluginName(pluginId),
          `${PLUGIN_FAILURE_WORDS.loadFailed}: ${formatError(error)}`,
        ),
        'error',
      ),
  }
}

const createWindowHandlers = (
  declared: DeclaredState,
  writeMessage: (text: string, severity: 'info' | 'error') => void,
  tabOpeners: ReturnType<typeof fileOpeners>,
  itemLists: ReturnType<typeof createCommandItemLists>,
) =>
  createBackendHandlers({
    getHandlers: () => declared.contributions.handlers.value,
    list: itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage,
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })

interface PluginsCoreDeps {
  readonly log: WindowCore['log']
  readonly writeMessage: WindowCore['writeMessage']
  readonly frame: WindowCore['frame']
  readonly windowTabs: TabsCore['windowTabs']
  readonly tabLinks: TabsCore['tabLinks']
  readonly tabOpeners: TabsCore['tabOpeners']
  readonly rest: SettingsStore
  readonly status: PluginStatus
  readonly registry: WindowCore['registry']
}

export function createPluginsCore(deps: PluginsCoreDeps): PluginsCore {
  const { log, writeMessage, frame, windowTabs, tabLinks, tabOpeners, rest, status, registry } = deps
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
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
    writeIssue: (pluginId, message, severity) =>
      log.getWriter(`plugin:${pluginId}`)(`${getPluginName(pluginId)}: ${message}`, severity),
  })
  const { pluginHost, pluginLoader } = createPluginSupport({
    slots: slots.hostDeps,
    getCommandDeclarations: () => declared.declarations.value,
    handlers: createWindowHandlers(declared, writeMessage, tabOpeners, itemLists),
    windowTabs,
    tabLinks,
    tabOpeners,
    rest,
    status,
    getPlatform: () => frame.platform.value,
    getServices: hostPort.getServices,
    notices: pluginNotices,
    ...createPluginReports(log, getPluginName),
    ...pluginNotes,
    resolveLink: async (from, address) => {
      const rawName = address.replace(/^name:\/\//, '')
      return (await resolveLinks(from, `[[${rawName}]]`))[0]?.path ?? ''
    },
  })
  const declaredCommands = useDeclaredCommands({
    state: declared,
    registry,
    isListed: (id) => pluginHost.commands.getPaletteCommands().some((one) => one.id === id),
    hasBackend: (pluginId) => hasPluginBackend(rest, pluginId),
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
  const appIds = [...new Set(commands.featureCommands.commands.map((one) => one.command ?? one.id))]
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
      vaults: createEditorVaults({
      vaultList: services.vaults.vaultList,
      getTarget: commands.getTarget,
      commandDeps: commands.commands.commandDeps,
    }),
      window: createWindowHost(frame, (error) => base.writeMessage(formatError(error), 'error')),
    },
    facts: commands.facts,
  })
  return { frame, slots: base.plugins.host.pluginHost.slots, windowCommands }
}

export const getPluginViews = (pluginHost: PluginHost) => ({
  titleBarView: pluginHost.slotViews.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.slotViews.getSlotView(SLOT.statusBar),
})
