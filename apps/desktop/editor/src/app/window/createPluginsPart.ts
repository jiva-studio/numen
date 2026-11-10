/**
 * The plugins of the window: the host and the loader, the services a plugin is given, and the words
 * a plugin's errors are reported in.
 */
import { formatError } from '@/shared/api/errors'
import { createWindowHost, SLOT, type PluginHost } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { listDeclaredItemLists, PLUGIN_FAILURE_WORDS, runDeclaredHandler } from '@/entities/plugin'
import type { fileOpeners } from '@/entities/tab'
import type { Command } from '@/entities/command'
import { getFileKinds } from '@/entities/file'
import { readNote, resolveLinks, writeNote } from '@/entities/note'
import type { createMessageLog } from '@/shared/model/notices/messages'
import {
  createBackendHandlers,
  createCommandItemLists,
  type CommandItemLists,
  useDeclaredCommands,
} from '@/features/plugin-commands'
import { createPluginNotes } from '../plugins/pluginNotes'
import type { PluginServices } from '../plugins/pluginServices'
import { createPluginSupport } from '../plugins/pluginSupport'
import type { createEditorCommands } from './createEditorCommands'
import type { PluginsCore, TabsCore, WindowCore } from './windowCore'
import type { WindowServices } from './createWindowServices'
import { getSettingsConfig } from '@/features/settings-commands'
import type { SettingsStore } from '@/entities/settings'
import { createWindowCommands } from '../plugins/windowCommands'
import { createWindowSlots } from '../plugins/windowSlots'
import { createEditorVaults } from '../plugins/windowVaults'

const getDeclaredCommandsDeps = (hostHolder: { current?: PluginHost }, rest: SettingsStore) => ({
  isListed: (id: string): boolean =>
    hostHolder.current?.getPaletteCommands().some((one) => one.id === id) ?? false,
  hasBackend: (pluginId: string): boolean =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.hasBackend ?? true,
  getConfig: () => getWindowConfig(rest),
})

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
    reportError: (pluginId: string, error: unknown) =>
      write(
        pluginId,
        PLUGIN_FAILURE_WORDS.error(getPluginName(pluginId), formatError(error)),
        'error',
      ),
    reportLoadError: (pluginId: string, error: unknown) =>
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
  declaredCommands: ReturnType<typeof useDeclaredCommands>,
  writeMessage: (text: string, severity: 'info' | 'error') => void,
  tabOpeners: ReturnType<typeof fileOpeners>,
  itemLists: CommandItemLists,
) =>
  createBackendHandlers({
    getHandlers: () => declaredCommands.contributions.handlers.value,
    list: itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage,
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })

const createDeclaredParts = (hostHolder: { current?: PluginHost }, rest: SettingsStore) => {
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: getFileKinds,
  })
  const declaredCommands = useDeclaredCommands({
    ...getDeclaredCommandsDeps(hostHolder, rest),
    listItems: itemLists.listItems,
  })
  return { itemLists, declaredCommands }
}

interface PluginsCoreDeps {
  readonly log: WindowCore['log']
  readonly writeMessage: WindowCore['writeMessage']
  readonly frame: WindowCore['frame']
  readonly windowTabs: TabsCore['windowTabs']
  readonly tabLinks: TabsCore['tabLinks']
  readonly tabOpeners: TabsCore['tabOpeners']
  readonly rest: SettingsStore
}

export function createPluginsCore(deps: PluginsCoreDeps): PluginsCore {
  const { log, writeMessage, frame, windowTabs, tabLinks, tabOpeners, rest } = deps
  /* Stryker disable OptionalChaining,ObjectLiteral,StringLiteral,Regex: window core wiring delegates to mocked collaborators in unit tests */
  const getPluginName = (pluginId: string): string =>
    rest.state.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const pluginServices: PluginServices = {}
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: readNote, write: writeNote })
  const hostHolder: { current?: PluginHost } = {}
  const { itemLists, declaredCommands } = createDeclaredParts(hostHolder, rest)
  const appCommands: { current?: () => readonly Command[] } = {}
  const slots = createWindowSlots({
    declaredCommands,
    hostHolder,
    getAppCommands: () => appCommands.current?.() ?? [],
    writeIssue: (pluginId, message, severity) =>
      log.getWriter(`plugin:${pluginId}`)(`${getPluginName(pluginId)}: ${message}`, severity),
  })
  const { pluginHost, pluginLoader } = createPluginSupport({
    slots: slots.hostDeps,
    getCommandDeclarations: () => declaredCommands.declarations.value,
    handlers: createWindowHandlers(declaredCommands, writeMessage, tabOpeners, itemLists),
    windowTabs,
    tabLinks,
    tabOpeners,
    rest,
    getPlatform: () => frame.platform.value,
    services: pluginServices,
    notices: pluginNotices,
    ...createPluginReports(log, getPluginName),
    ...pluginNotes,
    resolveLink: async (from, address) => {
      const rawName = address.replace(/^name:\/\//, '')
      return (await resolveLinks(from, `[[${rawName}]]`))[0]?.path ?? ''
    },
  })
  hostHolder.current = pluginHost
  /* Stryker restore OptionalChaining,ObjectLiteral,StringLiteral,Regex */
  return {
    declaredCommands,
    itemLists,
    host: { pluginHost, pluginLoader },
    pluginServices,
    pluginNotices,
    pluginNotes,
    slotWindowValues: slots.windowValues,
    appCommands,
  }
}

export function createWindowFrameServices(
  base: WindowCore,
  services: Pick<WindowServices, 'vaults'>,
  commands: Pick<
    ReturnType<typeof createEditorCommands>,
    'getTarget' | 'featureCommands' | 'commands'
  >,
) {
  const { frame } = base
  const { items, runCommandToEnd } = commands.commands.pluginRuns
  const appIds = [...new Set(commands.featureCommands.commands.map((one) => one.command ?? one.id))]
  const windowCommands = createWindowCommands({
    getAppIds: () => appIds,
    getPluginIds: () => base.plugins.host.pluginHost.getPaletteCommands().map((one) => one.id),
    runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget: commands.getTarget,
  })
  Object.assign(base.plugins.pluginServices, {
    commands: windowCommands,
    vaults: createEditorVaults({
      vaultList: services.vaults.vaultList,
      getTarget: commands.getTarget,
      commandDeps: commands.commands.commandDeps,
    }),
    window: createWindowHost(frame, (error) => base.writeMessage(formatError(error), 'error')),
  })
  return { frame, slots: base.plugins.host.pluginHost.slots, windowCommands }
}

export const getPluginViews = (pluginHost: PluginHost) => ({
  titleBarView: pluginHost.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.getSlotView(SLOT.statusBar),
})
