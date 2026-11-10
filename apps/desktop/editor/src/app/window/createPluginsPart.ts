/**
 * The plugins of the window: the host and the loader, the services a plugin is given, and the words
 * a plugin's errors are reported in.
 */
import { formatError } from '@/shared/api/errors'
import { createWindowHost, SLOT, type PluginHost } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { listDeclaredItemLists, runDeclaredHandler } from '@/entities/plugin'
import type { fileOpeners } from '@/entities/tab'
import type { Command } from '@/entities/command'
import { getFileKinds } from '@/entities/file'
import { readNote, resolveLinks, writeNote } from '@/entities/note'
import type { createMessageLog } from '@/shared/notices/messages'
import { WINDOW_WORDS } from './words'
import { createBackendHandlers } from '../plugins/commands/backendHandlers'
import { createCommandItemLists, type CommandItemLists } from '../plugins/commands/commandItemLists'
import { useDeclaredCommands } from '../plugins/commands/declaredCommands'
import { createPluginNotes } from '../plugins/pluginNotes'
import type { PluginServices } from '../plugins/pluginServices'
import { createPluginSupport } from '../plugins/useWindowPlugins'
import type { initWindowKindsAndCommands } from './createCommandsPart'
import {
  getCommands,
  getVaults,
  type PluginsCore,
  type TabsCore,
  type WindowContext,
  type WindowCore,
} from './createDisplayPart'
import { getSettingsConfig } from '@/features/settings-commands'
import type { useSettings } from './useSettings'
import { createWindowCommands } from '../plugins/windowCommands'
import { createWindowSlots } from '../plugins/windowSlots'
import { createEditorVaults } from '../plugins/windowVaults'

const getDeclaredCommandsDeps = (
  hostHolder: { current?: PluginHost },
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) => ({
  isListed: (id: string): boolean =>
    hostHolder.current?.getPaletteCommands().some((one) => one.id === id) ?? false,
  hasBackend: (pluginId: string): boolean =>
    settingsHolder.current?.rest.state.plugins.value.find((one) => one.id === pluginId)
      ?.hasBackend ?? true,
  getConfig: () => getWindowConfig(settingsHolder.current),
})

const getWindowConfig = (settings: ReturnType<typeof useSettings> | undefined) =>
  settings
    ? getSettingsConfig([
        { id: 'app', name: '', settings: settings.rest.state.app.value },
        ...settings.rest.state.plugins.value.map((one) => ({
          id: one.id,
          name: one.name,
          settings: one.settings,
        })),
      ])
    : {}

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
        WINDOW_WORDS.pluginError(getPluginName(pluginId), formatError(error)),
        'error',
      ),
    reportLoadError: (pluginId: string, error: unknown) =>
      write(
        pluginId,
        WINDOW_WORDS.pluginStopped(
          getPluginName(pluginId),
          `${WINDOW_WORDS.pluginLoadFailed}: ${formatError(error)}`,
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

const createDeclaredParts = (
  hostHolder: { current?: PluginHost },
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) => {
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: getFileKinds,
  })
  const declaredCommands = useDeclaredCommands({
    ...getDeclaredCommandsDeps(hostHolder, settingsHolder),
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
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
}

export function createPluginsCore(deps: PluginsCoreDeps): PluginsCore {
  const { log, writeMessage, frame, windowTabs, tabLinks, tabOpeners, settingsHolder } = deps
  /* Stryker disable OptionalChaining,ObjectLiteral,StringLiteral,Regex: window core wiring delegates to mocked collaborators in unit tests */
  const getPluginName = (pluginId: string): string =>
    settingsHolder.current?.rest.state.plugins.value.find((one) => one.id === pluginId)?.name ??
    pluginId
  const pluginServices: PluginServices = {}
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: readNote, write: writeNote })
  const hostHolder: { current?: PluginHost } = {}
  const { itemLists, declaredCommands } = createDeclaredParts(hostHolder, settingsHolder)
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
    settingsHolder,
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
  ctx: WindowContext,
  commands: Pick<ReturnType<typeof initWindowKindsAndCommands>, 'getTarget'>,
) {
  const { frame } = base
  let tableIds: readonly string[] | undefined
  const { items, runCommandToEnd } = getCommands(ctx).pluginRuns
  const windowCommands = createWindowCommands({
    getAppIds: () =>
      (tableIds ??= [
        ...new Set((ctx.featureCommands?.commands ?? []).map((one) => one.command ?? one.id)),
      ]),
    getPluginIds: () => base.plugins.host.pluginHost.getPaletteCommands().map((one) => one.id),
    runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget: commands.getTarget,
  })
  Object.assign(base.plugins.pluginServices, {
    commands: windowCommands,
    vaults: createEditorVaults({
      vaultList: getVaults(ctx).vaultList,
      getTarget: commands.getTarget,
      commandDeps: getCommands(ctx).commandDeps,
    }),
    window: createWindowHost(frame, (error) => base.writeMessage(formatError(error), 'error')),
  })
  return { frame, slots: base.plugins.host.pluginHost.slots, windowCommands }
}

export const getPluginViews = (pluginHost: PluginHost) => ({
  titleBarView: pluginHost.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.getSlotView(SLOT.statusBar),
})
