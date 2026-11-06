/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef } from 'vue'
import type {
  PluginContext,
  SlotHost,
  SettingsViewRegistration,
  VaultChangeEvent,
  DropRegistry,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createAppInfo } from '../appInfo'
import { formatCommandId } from '../commandId'
import { createDropRegistry } from '../drops'
import { createStatusViewRegistry, type StatusViewRegistry } from '../statusViews'
import { createSlotViewRegistry, type SlotViewRegistry } from '../slot-views/slotViews'
import { createViewRegistry, type ViewRegistry } from '../views'
import { createPluginContext } from './context'
import { addListener, notifyListeners } from './events'
import { createCommandApi } from './commandApi'
import { createPluginPaneWrapper } from './customTabKind'
import { registerReaderEntry, registerTabKindEntry } from './entries'
import {
  addEntry,
  disposeAllEntries,
  disposePluginEntry,
  setStarting,
  type HostRegistry,
} from './registry'
import type { PluginContextHost, PluginHostDeps } from './types'

export { createPluginPaneWrapper }

type EventBridge = Pick<
  PluginContextHost,
  'onVaultChange' | 'onFileSelect' | 'emitVaultChange' | 'emitFileSelect'
>

function getSettingsViewKey(pluginId: string, viewId: string): string {
  return `${pluginId}:${viewId}`
}

function createEventBridge(deps: PluginHostDeps, registry: HostRegistry): EventBridge {
  return {
    onVaultChange: (pluginId, handler) =>
      addListener(registry.vaultListeners, { pluginId, handler }),
    onFileSelect: (pluginId, handler) => addListener(registry.fileListeners, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.vaultListeners, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.fileListeners, event, deps),
  }
}

/** The registrations a plugin makes into the host's own registry. */
type RegistrationBridge = Pick<
  PluginContextHost,
  | 'registerReader'
  | 'registerTabKind'
  | 'registerCommand'
  | 'registerSettingsView'
  | 'registerAgentProvider'
>

function createRegistrationBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
): RegistrationBridge {
  return {
    registerReader: (pluginId, registration) =>
      registerReaderEntry(deps, registry.readers, {
        pluginId,
        registration,
        getContext: () => registry.contexts.get(pluginId),
      }),
    registerTabKind: (pluginId, registration) =>
      registerTabKindEntry(deps, registry.tabKinds, {
        pluginId,
        registration,
        getContext: () => registry.contexts.get(pluginId),
      }),
    registerCommand: (pluginId, registration) => {
      const fullId = formatCommandId(pluginId, registration.id)
      if (registry.commands.value.has(fullId)) {
        throw new Error(`Command "${fullId}" is already registered`)
      }
      return addEntry(registry.commands, fullId, { pluginId, reg: registration })
    },
    registerSettingsView: (pluginId, registration) =>
      addEntry(registry.settingsViews, getSettingsViewKey(pluginId, registration.id), {
        pluginId,
        reg: registration,
      }),
    // Stryker disable all: agentProviders map is internal state for future host provider lookup
    registerAgentProvider: (pluginId, registration) => {
      const entry = { pluginId, reg: registration }
      registry.agentProviders.set(registration.id, entry)
      return {
        dispose: () => {
          if (registry.agentProviders.get(registration.id) === entry) {
            registry.agentProviders.delete(registration.id)
          }
        },
      }
    },
    // Stryker restore all
  }
}

/** The settings of a plugin, as the window holds them. */
function createSettingBridge(
  deps: PluginHostDeps,
): Pick<PluginContextHost, 'getSetting' | 'onSettingChange'> {
  return {
    getSetting: <T = unknown>(pluginId: string, key: string) =>
      deps.getSetting ? (deps.getSetting(pluginId, key) as T | undefined) : undefined,
    onSettingChange: <T = unknown>(
      pluginId: string,
      key: string,
      handler: (value: T) => void | Promise<void>,
    ) => {
      // Without a host subscription there is nothing to unsubscribe.
      if (!deps.onSettingChange) return { dispose: () => undefined }
      return {
        dispose: deps.onSettingChange(
          pluginId,
          key,
          handler as (v: unknown) => void | Promise<void>,
        ),
      }
    },
  }
}

type SharedBridge = 'registerStatusView' | 'registerSlotView' | 'registerView' | 'mountView'

/** The views, status views and drops a plugin registers, which the shared registries hold. */
function createSharedBridge(
  shared: SharedRegistries,
): Pick<PluginContextHost, SharedBridge | 'registerDropHandler' | 'drops'> {
  const { views, drops, statusViews, slotViews } = shared
  return {
    registerStatusView: statusViews.registerStatusView,
    registerSlotView: slotViews.registerSlotView,
    registerView: views.registerView,
    mountView: views.mountView,
    registerDropHandler: drops.registerDropHandler,
    drops: { canDrop: drops.canDrop, drop: drops.drop },
  }
}

/** The tabs a plugin opens in the window. */
function createTabBridge(
  deps: PluginHostDeps,
): Pick<PluginContextHost, 'openTab' | 'navigateTab' | 'closeTab'> {
  return {
    openTab: (kind: string, target?: string) => deps.openTab(kind, target),
    navigateTab: (kind: string, target: string) => deps.navigateTab?.(kind, target),
    closeTab: (id: string) => deps.closeTab?.(id),
  }
}

/** The registries other than the host's own, which the context bridge delegates to. */
interface SharedRegistries {
  readonly views: ViewRegistry
  readonly drops: DropRegistry
  readonly statusViews: StatusViewRegistry
  readonly slotViews: SlotViewRegistry
}

function createHostContextBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  shared: SharedRegistries,
): PluginContextHost {
  return {
    app: deps.app ?? createAppInfo({ userAgent: '' }),
    openExternal: (url) =>
      deps.openExternal
        ? deps.openExternal(url)
        : Promise.reject(new Error('the window opens no addresses')),
    ...createRegistrationBridge(deps, registry),
    ...createSharedBridge(shared),
    ...createSettingBridge(deps),
    ...createEventBridge(deps, registry),
    ...createTabBridge(deps),
    ...createServiceBridge(deps),
  }
}

/** The host services a plugin context is given, as the window has made them. */
function createServiceBridge(
  deps: PluginHostDeps,
): Pick<
  PluginContextHost,
  | 'getWorkspace'
  | 'getNotes'
  | 'getRelations'
  | 'getFiles'
  | 'getAgent'
  | 'getCommands'
  | 'getBuffers'
  | 'getNotices'
  | 'getTasks'
  | 'getTexts'
> {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getNotes: () => deps.getNotes?.(),
    getRelations: () => deps.getRelations?.(),
    getFiles: () => deps.getFiles?.(),
    getAgent: () => deps.getAgent?.(),
    getCommands: () => deps.getCommands?.(),
    getBuffers: () => deps.getBuffers?.(),
    getNotices: (pluginId) => deps.getNotices?.(pluginId),
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
  }
}

function createHostRegistry(): HostRegistry {
  return {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    settingsViews: shallowRef(new Map()),
    agentProviders: new Map(),
    contexts: new Map(),
    manifests: new Map(),
    activated: shallowRef(new Set()),
    starting: shallowRef(new Set()),
    disposed: new Set(),
    reportedIssues: new Set(),
    vaultListeners: new Set(),
    fileListeners: new Set(),
  }
}

/** What the plugins registered, as lists. */
function createRegisteredApi(registry: HostRegistry) {
  return {
    /** The settings views a plugin registered, by view id; reactive. */
    getRegisteredSettingsViews: (
      pluginId: string,
    ): ReadonlyMap<string, SettingsViewRegistration> => {
      const views = new Map<string, SettingsViewRegistration>()
      for (const item of registry.settingsViews.value.values()) {
        if (item.pluginId === pluginId) views.set(item.reg.id, item.reg)
      }
      return views
    },
    getRegisteredReaders: () => [...registry.readers.values()].map((r) => r.reg),
    getRegisteredTabKinds: () => [...registry.tabKinds.values()].map((t) => t.reg),
    getRegisteredCommands: () => [...registry.commands.value.values()].map((c) => c.reg),
  }
}

/** The contexts and manifests of the plugins. */
function createContextApi(registry: HostRegistry, contextHost: PluginContextHost) {
  return {
    contextHost,
    createContext: (pluginId: string, manifest?: Manifest): PluginContext => {
      let ctx = registry.contexts.get(pluginId)
      if (!ctx) {
        const effectiveManifest = manifest ?? registry.manifests.get(pluginId)
        ctx = createPluginContext(pluginId, contextHost, effectiveManifest)
        registry.contexts.set(pluginId, ctx)
        registry.disposed.delete(pluginId)
        registry.activated.value = new Set(registry.activated.value).add(pluginId)
      }
      return ctx
    },
    markPluginStarting: (pluginId: string): void => setStarting(registry, pluginId, true),
    markPluginStarted: (pluginId: string): void => setStarting(registry, pluginId, false),
    registerManifest: (pluginId: string, manifest: Manifest): void => {
      if (registry.contexts.has(pluginId)) {
        throw new Error(
          `Cannot replace the manifest for plugin "${pluginId}" while its context is cached`,
        )
      }
      registry.manifests.set(pluginId, manifest)
    },
  }
}

export function createPluginHost(deps: PluginHostDeps) {
  const registry = createHostRegistry()
  const statusViews = createStatusViewRegistry()
  const slotViews = createSlotViewRegistry(
    deps.reportViewIssue ? { reportIssue: deps.reportViewIssue } : {},
  )
  const contextHost = createHostContextBridge(deps, registry, {
    views: createViewRegistry(),
    drops: createDropRegistry(),
    statusViews,
    slotViews,
  })
  const parts = { registry, statusViews, slotViews, deps }

  return {
    ...createContextApi(registry, contextHost),
    ...createCommandApi(deps, registry),
    ...createRegisteredApi(registry),
    getSetting: contextHost.getSetting,
    onSettingChange: contextHost.onSettingChange,
    /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
    notifyVaultChange: (event: VaultChangeEvent): void =>
      notifyListeners(registry.vaultListeners, event, deps),
    disposePlugin: (pluginId: string) => disposePluginEntry(pluginId, parts),
    disposeAll: () => disposeAllEntries(parts),
    /** The status bar views the plugins registered, in order of registration; reactive. */
    statusViews: statusViews.views,
    /** The view a slot of the window draws, or null while no plugin registered one; reactive. */
    getSlotView: slotViews.getSlotView,
  }
}

export type PluginHost = ReturnType<typeof createPluginHost> & {
  // The slots of the window the owners of slots read, where the host serves them.
  readonly slots?: SlotHost
}
