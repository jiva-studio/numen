import {
  createAppInfo,
  createPluginHost,
  NO_COMMANDS,
  NO_KINDS,
  NO_PLUGINS,
  NO_SLOTS,
  NO_TABS,
  NO_VAULT,
  NO_WINDOW,
  NO_WORK,
  createPluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { WindowPlatform } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import { TARGET } from '../lib/window'
import type { HostParts, TabActionParts, TabKindParts } from './createHostParts'
import type { createPluginCatalog } from './createPluginCatalog'
import { createPluginSettingsAccessor } from './pluginSettings'

/** What the plugin host and the plugin loader are made from. */
export interface PluginSupportDeps {
  readonly rest: SettingsStore
  readonly catalog: ReturnType<typeof createPluginCatalog>
  readonly status: PluginStatus
  readonly kinds: TabKindParts
  readonly tabs: TabActionParts
  readonly parts: readonly HostParts[]
  readonly getPlatform: () => WindowPlatform
  readonly reports: {
    readonly reportError: (pluginId: string, error: unknown) => void
    readonly reportLoadError: (pluginId: string, error: unknown) => void
  }
}

const mergeHostParts = (parts: readonly HostParts[]): HostParts =>
  parts.reduce<HostParts>(
    (all, one) => ({
      window: { ...all.window, ...one.window },
      plugins: { ...all.plugins, ...one.plugins },
      commands: { ...all.commands, ...one.commands },
      slots: { ...all.slots, ...one.slots },
      kinds: { ...all.kinds, ...one.kinds },
      tabs: { ...all.tabs, ...one.tabs },
      vault: { ...all.vault, ...one.vault },
      work: { ...all.work, ...one.work },
    }),
    {},
  )

export function createPluginSupport(deps: PluginSupportDeps) {
  const { rest, catalog, status } = deps
  const { getSetting, watchSetting, getEditorSettings, watchEditorSettings } =
    createPluginSettingsAccessor(rest, deps.reports.reportError)
  const parts = mergeHostParts(deps.parts)
  const pluginHost = createPluginHost({
    window: {
      ...NO_WINDOW,
      ...parts.window,
      target: TARGET,
      app: createAppInfo({
        version: import.meta.env.VITE_NUMEN_VERSION,
        build: import.meta.env.VITE_NUMEN_BUILD,
        getPlatform: deps.getPlatform,
      }),
      onError: deps.reports.reportError,
    },
    plugins: {
      ...NO_PLUGINS,
      ...parts.plugins,
      hasPluginFrontend: catalog.hasFrontend,
      hasPluginBackend: catalog.hasBackend,
      isPluginFailed: (pluginId) => status.getPluginFailure(pluginId) !== undefined,
      startPlugin: status.startPlugin,
      startPluginsOf: status.startPluginsOf,
      findPluginsOf: status.findPluginsOf,
      getSetting,
      watchSetting,
      getEditorSettings,
      watchEditorSettings,
    },
    commands: { ...NO_COMMANDS, ...parts.commands },
    slots: { ...NO_SLOTS, ...parts.slots },
    kinds: {
      ...NO_KINDS,
      ...parts.kinds,
      ...deps.kinds,
      getReaderExtensions: catalog.getReaderExtensions,
    },
    tabs: { ...NO_TABS, ...parts.tabs, ...deps.tabs },
    vault: { ...NO_VAULT, ...parts.vault },
    work: { ...NO_WORK, ...parts.work },
  })
  const pluginLoader = createPluginLoader({
    host: pluginHost.lifecycle,
    status,
    styles: pluginHost.slotViews.styles,
    onError: deps.reports.reportLoadError,
  })
  return { pluginHost, pluginLoader }
}
