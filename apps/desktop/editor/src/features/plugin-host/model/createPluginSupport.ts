import {
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import { openExternalUrl, type WindowPlatform } from '@numen/workbench'
import { getPluginConnections, getPluginTexts } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { Caught } from '@/shared/api/errors'
import { TARGET } from '../lib/window'
import type { HostParts, TabActionParts, TabKindParts } from './createHostParts'
import type { createPluginCatalog } from './createPluginCatalog'
import { createPluginSettingsAccessor, createRowChangeSubscriber } from './pluginSettings'

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
    readonly reportError: (pluginId: string, error: Caught) => void
    readonly reportLoadError: (pluginId: string, error: Caught) => void
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
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(
    rest,
    deps.reports.reportError,
  )
  const subscribeRowChanges = createRowChangeSubscriber(rest)
  const parts = mergeHostParts(deps.parts)
  const pluginHost = createPluginHost({
    window: {
      ...parts.window,
      target: TARGET,
      app: createAppInfo({
        version: import.meta.env.VITE_NUMEN_VERSION,
        build: import.meta.env.VITE_NUMEN_BUILD,
        getPlatform: deps.getPlatform,
      }),
      openExternal: (url) => openExternalUrl(url),
      onError: deps.reports.reportError,
    },
    plugins: {
      ...parts.plugins,
      hasPluginFrontend: catalog.hasFrontend,
      hasPluginBackend: catalog.hasBackend,
      isPluginFailed: (pluginId) => status.getPluginFailure(pluginId) !== undefined,
      startPlugin: status.startPlugin,
      startPluginsOf: status.startPluginsOf,
      findPluginsOf: status.findPluginsOf,
      getSetting,
      onSettingChange,
    },
    commands: { ...parts.commands },
    slots: { ...parts.slots },
    kinds: { ...parts.kinds, ...deps.kinds, getReaderExtensions: catalog.getReaderExtensions },
    tabs: { ...parts.tabs, ...deps.tabs },
    vault: { ...parts.vault, getTexts: (pluginId) => getPluginTexts(pluginId) },
    work: {
      ...parts.work,
      getConnections: (pluginId) => getPluginConnections(pluginId, subscribeRowChanges),
    },
  })
  const pluginLoader = createPluginLoader({
    host: pluginHost.lifecycle,
    status,
    onError: deps.reports.reportLoadError,
  })
  return { pluginHost, pluginLoader }
}
