import { watch } from 'vue'
import type { PluginServices } from './pluginServices'
import { createPluginHost } from '@/entities/plugin'
import {
  DEFAULT_PLUGIN_ENABLED,
  createPluginLoader,
  type PluginLoader,
} from '@/features/plugin-loader'
import { pluginCatalog } from './pluginCatalog'
import { cards } from '@/entities/deck'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { useSettings } from './useSettings'

export interface PluginSupportDeps {
  readonly held: ReturnType<typeof useWindowTabs>
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
  readonly readNote?: (
    path: string,
  ) => Promise<{ body?: string; error?: string } | string | undefined>
  readonly reportError?: (pluginId: string, error: unknown) => void
  readonly writeNote?: (path: string, body: string) => Promise<void>
  /** The window and the notes a plugin's tabs are given, filled once the window has made them. */
  readonly services?: PluginServices
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

function writeCustomCard(at: string, values: readonly { field: string; text: string }[]) {
  void cards.writeDeck(
    at,
    {
      preamble: '',
      cards: [{ values }] as unknown as Parameters<typeof cards.writeDeck>[1]['cards'],
      sections: [],
      tail: '',
    },
    null,
  )
}

function extractCardValues(args: unknown[]): readonly { field: string; text: string }[] {
  if (Array.isArray(args[1])) {
    return args[1] as readonly { field: string; text: string }[]
  }
  if (Array.isArray(args[0])) {
    return args[0] as readonly { field: string; text: string }[]
  }
  return []
}

export function createPluginSupport(deps: PluginSupportDeps) {
  const { held, tabLinks, tabOpeners, settingsHolder } = deps
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(settingsHolder)

  const closeTabsForPlugin = (pluginId: string) => {
    if (!pluginId) return
    for (const tab of held.tabs.value) {
      const windowTab = held.getTab(tab.id)
      if (windowTab?.kind.pluginId === pluginId) {
        tabLinks.removeClosedTab(tab.id)
        held.requestClose(tab.id)
      }
    }
  }

  const pluginHost = createPluginHost({
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    registerKind: held.registerKind,
    registerExtensionReader: tabOpeners.registerExtensionReader,
    registerEditor: tabOpeners.registerEditor,
    openTab: held.openTabOfKind,
    openTabBeside: held.openTabBesideOfKind,
    closeTab: held.finishClose,
    getTabState: (id: string) => held.getTab(id)?.state,
    getSetting,
    onSettingChange,
    closeTabsForPlugin,
    tabActionHandlers: {
      addCard: (at: string, ...args: unknown[]) => {
        if (!Array.isArray(args[1]) && !Array.isArray(args[0])) return
        writeCustomCard(at, extractCardValues(args))
      },
    },
    onCustomCardAdded: writeCustomCard,
    ...(deps.readNote !== undefined ? { readNote: deps.readNote } : {}),
    ...(deps.writeNote !== undefined ? { writeNote: deps.writeNote } : {}),
    getWorkspace: () => deps.services?.workspace,
    getNotes: () => deps.services?.notes,
    getFiles: () => deps.services?.files,
    getAgent: () => deps.services?.agent,
  })
  const pluginLoader = createPluginLoader({
    host: pluginHost,
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    loaders: pluginCatalog.loaders,
    manifests: pluginCatalog.manifests,
  })

  return { pluginHost, pluginLoader }
}

export function watchPluginSettings(
  settings: ReturnType<typeof useSettings>,
  pluginLoader: PluginLoader,
) {
  watch(
    () => settings.rest?.plugins?.value ?? [],
    (plugins) => {
      void pluginLoader.trackPlugins(plugins.map((plugin) => plugin.id))
    },
  )
}
