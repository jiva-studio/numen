import { watch } from 'vue'
import { openExternalUrl, runInBackground } from '@numen/ui'
import type { PluginServices } from './pluginServices'
import { registerTabKindIcon } from '@/entities/note'
import { getPluginTexts } from '@/entities/plugin'
import {
  DEFAULT_PLUGIN_ENABLED,
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  createPluginStatus,
  isPluginEnabledValue,
  createPluginNotices,
  createPluginTasks,
  type CommandDeclaration,
  type PluginHostDeps,
  type PluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { NoticeQueue } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import type { WindowPlatform } from '@numen/workbench'

export interface PluginSupportDeps {
  readonly windowTabs: WindowTabs
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly rest: SettingsStore
  readonly getPlatform?: () => WindowPlatform
  readonly readNote?: (
    path: string,
  ) => Promise<{ body?: string; error?: string } | string | undefined>
  readonly reportError?: (pluginId: string, error: unknown) => void
  readonly reportLoadError?: (pluginId: string, error: unknown) => void
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  readonly writeNote?: (path: string, body: string) => Promise<void>
  readonly status?: PluginStatus
  readonly getServices?: () => PluginServices
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
    | 'createSlotCommandFinder'
    | 'reportSlotIssue'
    | 'getSlotWhenValues'
    | 'getWindowValues'
    | 'getContentTypes'
  >
}

const getPluginSetting = (rest: SettingsStore, pluginId: string, key: string): unknown => {
  // Stryker disable next-line StringLiteral: rest reads under the plugin id.
  const value = rest.read.getCommittedSetting([pluginId, key])
  if (value !== undefined) return value
  return key === 'enabled' ? DEFAULT_PLUGIN_ENABLED : undefined
}

/** Where the plugins stand, as the settings say which are enabled. */
export const createWindowPluginStatus = (rest: SettingsStore): PluginStatus =>
  createPluginStatus({
    isEnabled: (id) => isPluginEnabledValue(getPluginSetting(rest, id, 'enabled')),
  })

function createPluginSettingsAccessor(
  rest: SettingsStore,
  reportError: (pluginId: string, error: unknown) => void,
) {
  const getSetting = (pluginId: string, key: string): unknown =>
    getPluginSetting(rest, pluginId, key)

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

function createVaultServiceAccess(getServices: (() => PluginServices) | undefined) {
  /* Stryker disable OptionalChaining,ArrowFunction,ObjectLiteral: host wiring forwards optional capabilities */
  return {
    getWorkspace: () => getServices?.().workspace,
    getNotes: () => getServices?.().notes,
    getRelations: () => getServices?.().relations,
    getFiles: () => getServices?.().files,
    getAgent: () => getServices?.().agent,
    getCommands: () => getServices?.().commands,
    getVaults: () => getServices?.().vaults,
    getWindow: () => getServices?.().window,
  }
  /* Stryker restore OptionalChaining,ArrowFunction,ObjectLiteral */
}

function createWindowServiceAccess(getServices: (() => PluginServices) | undefined) {
  /* Stryker disable OptionalChaining,ArrowFunction,ObjectLiteral: host wiring forwards optional capabilities */
  return {
    getBuffers: () => getServices?.().buffers,
    getNoteBuffer: () => getServices?.().noteBuffer,
    getTasks: (pluginId: string) => {
      const list = getServices?.().taskList
      return list ? createPluginTasks(list, pluginId) : undefined
    },
    getBackgroundTasks: () => getServices?.().backgroundTasks,
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
  const { windowTabs, tabOpeners, rest } = deps
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(
    rest,
    // Stryker disable next-line OptionalChaining: reportError is optional on PluginSupportDeps
    (pluginId, error) => deps.reportError?.(pluginId, error),
  )

  /** The tabs of a plugin that stops, kept as placeholders that hold their saved state. */
  const closeTabsForPlugin = (pluginId: string) => {
    if (pluginId) windowTabs.groups.closeTabsOfPlugin(pluginId)
  }

  /* Stryker disable OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction: host and loader wiring forwards optional capabilities */
  const status = deps.status ?? createWindowPluginStatus(rest)
  const pluginHost = createPluginHost({
    ...getHostDeclarations(deps),
    hasPluginFrontend: (pluginId) =>
      rest.state.plugins.value.find((plugin) => plugin.id === pluginId)?.hasFrontend ?? true,
    hasPluginBackend: (pluginId) =>
      rest.state.plugins.value.find((plugin) => plugin.id === pluginId)?.hasBackend ?? true,
    isPluginFailed: (pluginId) => status.getPluginFailure(pluginId) !== undefined,
    waitForPlugin: (pluginId) => status.settlePlugin(pluginId),
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      getPlatform: deps.getPlatform ?? (() => 'windows'),
    }),
    openExternal: (url) => openExternalUrl(url),
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    registerKind: (kind) => {
      const unregisterKind = windowTabs.groups.registerKind(kind)
      const unregisterIcon = kind.icon ? registerTabKindIcon(kind.kind, kind.icon) : () => undefined
      return () => {
        unregisterKind()
        unregisterIcon()
      }
    },
    registerExtensionReader: tabOpeners.registerExtensionReader,
    getReaderExtensions: (pluginId, readerId) =>
      rest.state.plugins.value
        .find((plugin) => plugin.id === pluginId)
        ?.readers?.find((reader) => reader.id === readerId)?.extensions ?? [],
    registerEditor: tabOpeners.registerEditor,
    openTab: windowTabs.tabs.openTabOfKind,
    openTabAdjacent: windowTabs.tabs.openTabAdjacentOfKind,
    navigateTab: windowTabs.tabs.navigateTab,
    closeTab: windowTabs.panes.finishClose,
    getTabState: (id: string) => windowTabs.tabs.getTab(id)?.state,
    getSetting,
    onSettingChange,
    closeTabsForPlugin,
    ...(deps.readNote !== undefined ? { readNote: deps.readNote } : {}),
    ...(deps.writeNote !== undefined ? { writeNote: deps.writeNote } : {}),
    ...createVaultServiceAccess(deps.getServices),
    ...createWindowServiceAccess(deps.getServices),
    ...(deps.resolveLink !== undefined ? { resolveLink: deps.resolveLink } : {}),
    getTexts: (pluginId: string) => getPluginTexts(pluginId),
    ...(deps.notices !== undefined ? createNoticeAccess(deps.notices) : {}),
  })
  const reportLoadError = deps.reportLoadError ?? deps.reportError
  const pluginLoader = createPluginLoader({
    host: pluginHost,
    status,
    ...(reportLoadError !== undefined ? { onError: reportLoadError } : {}),
  })
  /* Stryker restore OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction */

  return { pluginHost, pluginLoader }
}

/** The plugins whose frontend the window loads: a plugin that declares none has nothing to fetch. */
export const getFrontendPluginIds = (
  plugins: readonly { id: string; hasFrontend: boolean }[],
): string[] => plugins.filter((plugin) => plugin.hasFrontend).map((plugin) => plugin.id)

export function watchPluginSettings(
  rest: SettingsStore,
  pluginLoader: PluginLoader,
  reportError: (error: unknown) => void,
) {
  watch(
    () => rest.state.plugins.value,
    (plugins) => {
      runInBackground(pluginLoader.trackPlugins(getFrontendPluginIds(plugins)), reportError)
    },
  )
}
