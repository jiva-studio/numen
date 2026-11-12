import {
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import { openExternalUrl, type WindowPlatform } from '@numen/workbench'
import { getPluginModels, getPluginTexts } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { Caught } from '@/shared/api/errors'
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
  const pluginHost = createPluginHost({
    ...mergeHostParts(deps.parts),
    ...deps.kinds,
    ...deps.tabs,
    hasPluginFrontend: catalog.hasFrontend,
    hasPluginBackend: catalog.hasBackend,
    isPluginFailed: (pluginId) => status.getPluginFailure(pluginId) !== undefined,
    waitForPlugin: (pluginId) => status.settlePlugin(pluginId),
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      getPlatform: deps.getPlatform,
    }),
    openExternal: (url) => openExternalUrl(url),
    onError: deps.reports.reportError,
    getReaderExtensions: catalog.getReaderExtensions,
    getSetting,
    onSettingChange,
    getTexts: (pluginId) => getPluginTexts(pluginId),
    getModels: (pluginId) => getPluginModels(pluginId),
  })
  const pluginLoader = createPluginLoader({
    host: pluginHost.lifecycle,
    status,
    onError: deps.reports.reportLoadError,
  })
  return { pluginHost, pluginLoader }
}
