import { openExternalUrl } from '@numen/ui'
import { registerTabKindIcon } from '@/entities/note'
import { getPluginTexts } from '@/entities/plugin'
import type { PluginServices } from './pluginServices'
import { createPluginCatalog } from './createPluginCatalog'
import { createPluginSettingsAccessor, createWindowPluginStatus } from './pluginSettings'
import {
  createAppInfo,
  createPluginHost,
  createPluginLoader,
  createPluginNotices,
  createPluginTasks,
  type CommandDeclaration,
  type PluginHostDeps,
  type PluginStatus,
} from '@numen/plugin-loader'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { NoticeQueue } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import type { WindowPlatform } from '@numen/workbench'
import type { Caught } from '@/shared/api/errors'

export interface PluginSupportDeps {
  readonly windowTabs: WindowTabs
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly rest: SettingsStore
  readonly getPlatform?: () => WindowPlatform
  readonly readNote?: (
    path: string,
  ) => Promise<{ body?: string; error?: string } | string | undefined>
  readonly reportError?: (pluginId: string, error: Caught) => void
  readonly reportLoadError?: (pluginId: string, error: Caught) => void
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  readonly writeNote?: (path: string, body: string) => Promise<void>
  readonly status?: PluginStatus
  readonly getServices?: () => Partial<PluginServices>
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

function createVaultServiceAccess(getServices: (() => Partial<PluginServices>) | undefined) {
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

function createWindowServiceAccess(getServices: (() => Partial<PluginServices>) | undefined) {
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
  const catalog = createPluginCatalog(rest)
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
    registerKind: (kind) => {
      const unregisterKind = windowTabs.groups.registerKind(kind)
      const icon = kind.display?.icon
      const unregisterIcon = icon ? registerTabKindIcon(kind.kind, icon) : () => undefined
      return () => {
        unregisterKind()
        unregisterIcon()
      }
    },
    registerExtensionReader: tabOpeners.registerExtensionReader,
    getReaderExtensions: catalog.getReaderExtensions,
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
    host: pluginHost.lifecycle,
    status,
    ...(reportLoadError !== undefined ? { onError: reportLoadError } : {}),
  })
  /* Stryker restore OptionalChaining,ConditionalExpression,EqualityOperator,LogicalOperator,ObjectLiteral,ArrowFunction */

  return { pluginHost, pluginLoader }
}
