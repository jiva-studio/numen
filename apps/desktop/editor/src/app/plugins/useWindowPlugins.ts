import { watch } from 'vue'
import type { PluginServices } from './pluginServices'
import { createPluginHost } from '@/entities/plugin'
import { registerNoteIcon } from '@/entities/note'
import { DEFAULT_PLUGIN_ENABLED, createPluginLoader, type PluginLoader } from '@numen/plugin-host'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { NoticeQueue } from '@numen/shell'
import type { useSettings } from '../window/useSettings'
import { createPluginNotices } from './pluginNotices'
import { createPluginTasks } from './pluginTasks'
import { createPluginTexts } from './pluginTexts'
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
  readonly writeNote?: (path: string, body: string) => Promise<void>
  /** The window and the notes a plugin's tabs are given, filled once the window has made them. */
  readonly services?: PluginServices
  /** The window's one list of notices, which each plugin shows its own in. */
  readonly notices?: NoticeQueue
}

function createPluginSettingsAccessor(settingsHolder: {
  current?: ReturnType<typeof useSettings>
}) {
  const getSetting = (pluginId: string, key: string): unknown => {
    const rest = settingsHolder.current?.rest
    if (!rest) return undefined
    const value = rest.getSetting?.(['plugins', pluginId, key])
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
      (newVal, oldVal) => {
        if (newVal !== oldVal) void handler(newVal)
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
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(settingsHolder)

  /** The tabs of a plugin that stops, kept as placeholders that hold their saved state. */
  const closeTabsForPlugin = (pluginId: string) => {
    if (pluginId) windowTabs.closeTabsOfPlugin(pluginId)
  }

  const pluginHost = createPluginHost({
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    registerKind: (kind) => {
      const unregisterKind = windowTabs.registerKind(kind)
      const unregisterIcon = kind.icon ? registerNoteIcon(kind.kind, kind.icon) : () => {}
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
    openTabBeside: windowTabs.openTabBesideOfKind,
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
    getFiles: () => deps.services?.files,
    getAgent: () => deps.services?.agent,
    getBuffers: () => deps.services?.buffers,
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

  return { pluginHost, pluginLoader }
}

/** The plugins whose frontend the window loads: a plugin that declares none has nothing to fetch. */
export const getFrontendPluginIds = (
  plugins: readonly { id: string; hasFrontend: boolean }[],
): string[] => plugins.filter((plugin) => plugin.hasFrontend).map((plugin) => plugin.id)

export function watchPluginSettings(
  settings: ReturnType<typeof useSettings>,
  pluginLoader: PluginLoader,
) {
  watch(
    () => settings.rest?.plugins?.value ?? [],
    (plugins) => {
      void pluginLoader.trackPlugins(getFrontendPluginIds(plugins))
    },
  )
}
