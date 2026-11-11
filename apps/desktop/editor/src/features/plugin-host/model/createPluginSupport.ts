import { openExternalUrl } from '@/shared/api/wailsRuntime'
import {
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { WindowPlatform } from '@numen/workbench'
import { getPluginTexts } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { Caught } from '@/shared/api/errors'
import type { HostParts, TabActionParts, TabKindParts } from './createHostParts'
import { createPluginCatalog } from './createPluginCatalog'
import { createPluginSettingsAccessor, createWindowPluginStatus } from './pluginSettings'

/** What the plugin host and the plugin loader are made from. */
export interface PluginSupportDeps {
  readonly rest: SettingsStore
  readonly kinds: TabKindParts
  readonly tabs: TabActionParts
  readonly parts: readonly HostParts[]
  readonly status?: PluginStatus
  readonly getPlatform?: () => WindowPlatform
  readonly reportError?: (pluginId: string, error: Caught) => void
  readonly reportLoadError?: (pluginId: string, error: Caught) => void
}

const mergeHostParts = (parts: readonly HostParts[]): HostParts =>
  parts.reduce<HostParts>((all, one) => ({ ...all, ...one }), {})

export function createPluginSupport(deps: PluginSupportDeps) {
  const { rest } = deps
  const catalog = createPluginCatalog(rest)
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(
    rest,
    // Stryker disable next-line OptionalChaining: reportError is optional on PluginSupportDeps
    (pluginId, error) => deps.reportError?.(pluginId, error),
  )

  /* Stryker disable OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction: host and loader wiring forwards optional capabilities */
  const status = deps.status ?? createWindowPluginStatus(rest)
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
      getPlatform: deps.getPlatform ?? (() => 'windows'),
    }),
    openExternal: (url) => openExternalUrl(url),
    ...(deps.reportError !== undefined ? { onError: deps.reportError } : {}),
    getReaderExtensions: catalog.getReaderExtensions,
    getSetting,
    onSettingChange,
    getTexts: (pluginId) => getPluginTexts(pluginId),
  })
  const reportLoadError = deps.reportLoadError ?? deps.reportError
  const pluginLoader = createPluginLoader({
    host: pluginHost.lifecycle,
    status,
    ...(reportLoadError !== undefined ? { onError: reportLoadError } : {}),
  })
  /* Stryker restore OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction */

  return { pluginHost, pluginLoader }
}
