import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginServices } from './pluginServices'
import { registerNoteIcon } from '@/entities/note'
import {
  DEFAULT_PLUGIN_ENABLED,
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  createPluginNotices,
  createPluginTasks,
  createPluginTexts,
  openExternalUrl,
  type CommandDeclaration,
  type PluginLoader,
} from '@numen/plugin-host'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { NoticeQueue } from '@numen/shell'
import type { useSettings } from '../window/useSettings'
import { texts } from '@/shared/clients'

export interface PluginSupportDeps {
  readonly windowTabs: ReturnType<typeof useWindowTabs>
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
  readonly readNote?: (
    path: string,
  ) => Promise<{ body?: string; error?: string } | string | undefined>
  readonly reportError?: (pluginId: string, error: unknown) => void
  /** Reports a plugin whose frontend failed to load or activate; falls back to `reportError`. */
  readonly reportLoadError?: (pluginId: string, error: unknown) => void
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  readonly writeNote?: (path: string, body: string) => Promise<void>
  /** The window and the notes a plugin's tabs are given, filled once the window has made them. */
  readonly services?: PluginServices
  /** The window's one list of notices, which each plugin shows its own in. */
  readonly notices?: NoticeQueue
  /** The commands the enabled plugins declare in their manifests. */
  readonly getCommandDeclarations?: () => readonly CommandDeclaration[] | undefined
  /** Reports a command whose declaration and handler do not match. */
  readonly reportCommandIssue?: (pluginId: string, message: string) => void
}

function createPluginSettingsAccessor(
  settingsHolder: { current?: ReturnType<typeof useSettings> },
  reportError: (pluginId: string, error: unknown) => void,
) {
  const getSetting = (pluginId: string, key: string): unknown => {
    const rest = settingsHolder.current?.rest
    if (!rest) return undefined
    const read = rest.getCommittedSetting ?? rest.getSetting
    // Stryker disable next-line OptionalChaining,StringLiteral: rest always defines getSetting and reads under the plugin id.
    const value = read?.([pluginId, key])
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

function createNoticeAccess(queue: NoticeQueue) {
  return {
    getNotices: (pluginId: string) => createPluginNotices(queue, pluginId),
    clearNotices: (pluginId: string) => queue.clear(pluginId),
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
    ...(deps.getCommandDeclarations !== undefined
      ? { getCommandDeclarations: deps.getCommandDeclarations }
      : {}),
    ...(deps.reportCommandIssue !== undefined
      ? { reportCommandIssue: deps.reportCommandIssue }
      : {}),
    hasPluginFrontend: (pluginId) =>
      settingsHolder.current?.rest?.plugins?.value.find((plugin) => plugin.id === pluginId)
        ?.hasFrontend ?? true,
    hasPluginBackend: (pluginId) =>
      settingsHolder.current?.rest?.plugins?.value.find((plugin) => plugin.id === pluginId)
        ?.hasBackend ?? true,
    isPluginFailed: (pluginId) => loaderHolder.current?.getPluginFailure(pluginId) !== undefined,
    waitForPlugin: (pluginId) => loaderHolder.current?.settlePlugin(pluginId) ?? Promise.resolve(),
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      userAgent: navigator.userAgent,
    }),
    openExternal: (url) => openExternalUrl(url),
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    registerKind: (kind) => {
      const unregisterKind = windowTabs.registerKind(kind)
      const unregisterIcon = kind.icon ? registerNoteIcon(kind.kind, kind.icon) : () => undefined // No icon was registered.
      return () => {
        unregisterKind()
        unregisterIcon()
      }
    },
    registerExtensionReader: tabOpeners.registerExtensionReader,
    getReaderExtensions: (pluginId, readerId) =>
      settingsHolder.current?.rest?.plugins?.value
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
    getWorkspace: () => deps.services?.workspace,
    getNotes: () => deps.services?.notes,
    getRelations: () => deps.services?.relations,
    getFiles: () => deps.services?.files,
    getAgent: () => deps.services?.agent,
    getBuffers: () => deps.services?.buffers,
    getNoteBuffer: () => deps.services?.noteBuffer,
    ...(deps.resolveLink !== undefined ? { resolveLink: deps.resolveLink } : {}),
    getTasks: (pluginId) => {
      const list = deps.services?.taskList
      return list ? createPluginTasks(list, pluginId) : undefined
    },
    getTexts: (pluginId: string) => createPluginTexts(texts, pluginId),
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
    // Stryker disable next-line OptionalChaining: settings.rest or plugins may be absent in unit test mocks
    () => settings.rest?.plugins?.value ?? [],
    (plugins) => {
      runInBackground(pluginLoader.trackPlugins(getFrontendPluginIds(plugins)), reportError)
    },
  )
}
