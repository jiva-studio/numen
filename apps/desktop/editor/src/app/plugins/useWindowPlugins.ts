import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginServices } from './pluginServices'
import { registerTabKindIcon } from '@/entities/note'
import { getPluginTexts } from '@/entities/plugin'
import {
  DEFAULT_PLUGIN_ENABLED,
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  createPluginNotices,
  createPluginTasks,
  openExternalUrl,
  type CommandDeclaration,
  type PluginHostDeps,
  type PluginLoader,
} from '@numen/plugin-loader'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { NoticeQueue } from '@numen/workbench'
import type { useSettings } from '../window/useSettings'
import type { WindowPlatform } from '@numen/workbench'

export interface PluginSupportDeps {
  readonly windowTabs: WindowTabs
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
  readonly getPlatform?: () => WindowPlatform
  readonly readNote?: (
    path: string,
  ) => Promise<{ body?: string; error?: string } | string | undefined>
  readonly reportError?: (pluginId: string, error: unknown) => void
  readonly reportLoadError?: (pluginId: string, error: unknown) => void
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  readonly writeNote?: (path: string, body: string) => Promise<void>
  readonly services?: PluginServices
  readonly notices?: NoticeQueue
  readonly getCommandDeclarations?: () => readonly CommandDeclaration[] | undefined
  readonly handlers?: Pick<
    PluginHostDeps,
    'getHandlerDeclarations' | 'listBackendHandlerItems' | 'runBackendHandler'
  >
  readonly reportCommandIssue?: (pluginId: string, message: string) => void
  readonly reportViewIssue?: (pluginId: string, message: string) => void
  readonly slots?: Pick<
    PluginHostDeps,
    | 'getSlotEntries'
    | 'findSlotCommand'
    | 'reportSlotIssue'
    | 'getSlotWhenValues'
    | 'getWindowValues'
    | 'getContentTypes'
  >
}

function createPluginSettingsAccessor(
  settingsHolder: { current?: ReturnType<typeof useSettings> },
  reportError: (pluginId: string, error: unknown) => void,
) {
  const getSetting = (pluginId: string, key: string): unknown => {
    const rest = settingsHolder.current?.rest
    if (!rest) return undefined
    // Stryker disable next-line StringLiteral: rest reads under the plugin id.
    const value = rest.getCommittedSetting([pluginId, key])
    if (value !== undefined) return value
    return key === 'enabled' ? DEFAULT_PLUGIN_ENABLED : undefined
  }

  const onSettingChange = (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void | Promise<void>,
  ): (() => void) =>
    watch(
      () => getSetting(pluginId, key),
      (newVal) => {
        runInBackground(Promise.resolve(handler(newVal)), (error) => reportError(pluginId, error))
      },
    )

  return { getSetting, onSettingChange }
}

function createServiceAccess(services: PluginServices | undefined) {
  /* Stryker disable OptionalChaining,ArrowFunction,ObjectLiteral: host wiring forwards optional capabilities */
  return {
    getWorkspace: () => services?.workspace,
    getNotes: () => services?.notes,
    getRelations: () => services?.relations,
    getFiles: () => services?.files,
    getAgent: () => services?.agent,
    getCommands: () => services?.commands,
    getVaults: () => services?.vaults,
    getWindow: () => services?.window,
    getBuffers: () => services?.buffers,
    getNoteBuffer: () => services?.noteBuffer,
    getTasks: (pluginId: string) => {
      const list = services?.taskList
      return list ? createPluginTasks(list, pluginId) : undefined
    },
    getBackgroundTasks: () => services?.backgroundTasks,
  }
  /* Stryker restore OptionalChaining,ArrowFunction,ObjectLiteral */
}

function createNoticeAccess(queue: NoticeQueue) {
  return {
    getNotices: (pluginId: string) => createPluginNotices(queue, pluginId),
    clearNotices: (pluginId: string) => queue.clear(pluginId),
  }
}

function getHostDeclarations(deps: PluginSupportDeps) {
  return {
    ...deps.slots,
    ...deps.handlers,
    ...(deps.getCommandDeclarations !== undefined
      ? { getCommandDeclarations: deps.getCommandDeclarations }
      : {}),
    ...(deps.reportViewIssue !== undefined ? { reportViewIssue: deps.reportViewIssue } : {}),
    ...(deps.reportCommandIssue !== undefined
      ? { reportCommandIssue: deps.reportCommandIssue }
      : {}),
  }
}

export function createPluginSupport(deps: PluginSupportDeps) {
  const { windowTabs, tabOpeners, settingsHolder } = deps
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(
    settingsHolder,
    // Stryker disable next-line OptionalChaining: reportError is optional on PluginSupportDeps
    (pluginId, error) => deps.reportError?.(pluginId, error),
  )

  /** The tabs of a plugin that stops, kept as placeholders that hold their saved state. */
  const closeTabsForPlugin = (pluginId: string) => {
    if (pluginId) windowTabs.closeTabsOfPlugin(pluginId)
  }

  /* Stryker disable OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction: host and loader wiring forwards optional capabilities */
  const loaderHolder: { current?: PluginLoader } = {}
  const pluginHost = createPluginHost({
    ...getHostDeclarations(deps),
    hasPluginFrontend: (pluginId) =>
      settingsHolder.current?.rest.plugins.value.find((plugin) => plugin.id === pluginId)
        ?.hasFrontend ?? true,
    hasPluginBackend: (pluginId) =>
      settingsHolder.current?.rest.plugins.value.find((plugin) => plugin.id === pluginId)
        ?.hasBackend ?? true,
    isPluginFailed: (pluginId) => loaderHolder.current?.getPluginFailure(pluginId) !== undefined,
    waitForPlugin: (pluginId) => loaderHolder.current?.settlePlugin(pluginId) ?? Promise.resolve(),
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      getPlatform: deps.getPlatform ?? (() => 'windows'),
    }),
    openExternal: (url) => openExternalUrl(url),
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    registerKind: (kind) => {
      const unregisterKind = windowTabs.registerKind(kind)
      const unregisterIcon = kind.icon ? registerTabKindIcon(kind.kind, kind.icon) : () => undefined
      return () => {
        unregisterKind()
        unregisterIcon()
      }
    },
    registerExtensionReader: tabOpeners.registerExtensionReader,
    getReaderExtensions: (pluginId, readerId) =>
      settingsHolder.current?.rest.plugins.value
        .find((plugin) => plugin.id === pluginId)
        ?.readers?.find((reader) => reader.id === readerId)?.extensions ?? [],
    registerEditor: tabOpeners.registerEditor,
    openTab: windowTabs.openTabOfKind,
    openTabAdjacent: windowTabs.openTabAdjacentOfKind,
    navigateTab: windowTabs.navigateTab,
    closeTab: windowTabs.finishClose,
    getTabState: (id: string) => windowTabs.getTab(id)?.state,
    getSetting,
    onSettingChange,
    closeTabsForPlugin,
    ...(deps.readNote !== undefined ? { readNote: deps.readNote } : {}),
    ...(deps.writeNote !== undefined ? { writeNote: deps.writeNote } : {}),
    ...createServiceAccess(deps.services),
    ...(deps.resolveLink !== undefined ? { resolveLink: deps.resolveLink } : {}),
    getTexts: (pluginId: string) => getPluginTexts(pluginId),
    ...(deps.notices !== undefined ? createNoticeAccess(deps.notices) : {}),
  })
  const reportLoadError = deps.reportLoadError ?? deps.reportError
  const pluginLoader = createPluginLoader({
    host: pluginHost,
    ...(reportLoadError !== undefined ? { onError: reportLoadError } : {}),
  })
  loaderHolder.current = pluginLoader
  /* Stryker restore OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction */

  return { pluginHost, pluginLoader }
}

/** The plugins whose frontend the window loads: a plugin that declares none has nothing to fetch. */
export const getFrontendPluginIds = (
  plugins: readonly { id: string; hasFrontend: boolean }[],
): string[] => plugins.filter((plugin) => plugin.hasFrontend).map((plugin) => plugin.id)

export function watchPluginSettings(
  settings: ReturnType<typeof useSettings>,
  pluginLoader: PluginLoader,
  reportError: (error: unknown) => void,
) {
  watch(
    () => settings.rest.plugins.value,
    (plugins) => {
      runInBackground(pluginLoader.trackPlugins(getFrontendPluginIds(plugins)), reportError)
    },
  )
}
