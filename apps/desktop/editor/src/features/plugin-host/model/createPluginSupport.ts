import {
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import { openExternalUrl, type WindowPlatform } from '@numen/workbench'
import type { PluginHostApi } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { Caught } from '@/shared/api/errors'
import { loadFrameRuntime } from '@/shared/api/runtime'
import { TARGET } from '../lib/window'
import type { HostParts, TabActionParts, TabKindParts } from './createHostParts'
import type { createPluginCatalog } from './createPluginCatalog'
import { createPluginSettingsAccessor, createRowChangeSubscriber } from './pluginSettings'

/** What the plugin host and the plugin loader are made from. */
export interface PluginSupportDeps {
  readonly api: PluginHostApi
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
  parts.reduce<HostParts>((all, one) => ({ ...all, ...one }), {})

export function createPluginSupport(deps: PluginSupportDeps) {
  const { rest, catalog, status } = deps
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(
    rest,
    deps.reports.reportError,
  )
  const subscribeRowChanges = createRowChangeSubscriber(rest)
  const pluginHost = createPluginHost({
    target: TARGET,
    ...mergeHostParts(deps.parts),
    ...deps.kinds,
    ...deps.tabs,
    hasPluginFrontend: catalog.hasFrontend,
    hasPluginBackend: catalog.hasBackend,
    isPluginFailed: (pluginId) => status.getPluginFailure(pluginId) !== undefined,
    startPlugin: status.startPlugin,
    startPluginsOf: status.startPluginsOf,
    findPluginsOf: status.findPluginsOf,
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      getPlatform: deps.getPlatform,
    }),
    openExternal: (url) => openExternalUrl(url, loadFrameRuntime),
    onError: deps.reports.reportError,
    getReaderExtensions: catalog.getReaderExtensions,
    getSetting,
    onSettingChange,
    getTexts: (pluginId) => deps.api.getPluginTexts(pluginId),
    getConnections: (pluginId) => deps.api.getPluginConnections(pluginId, subscribeRowChanges),
  })
  const pluginLoader = createPluginLoader({
    host: pluginHost.lifecycle,
    status,
    onError: deps.reports.reportLoadError,
  })
  return { pluginHost, pluginLoader }
}
