/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef } from 'vue'
import type {
  PluginContext,
  SettingsViewRegistration,
  VaultChangeEvent,
  DropRegistry,
} from '@numen/plugin-sdk'
import { ReportedError, type Manifest } from '@numen/core-client'
import { createAppInfo } from '../appInfo'
import { formatCommandId } from '../commandId'
import { createDropRegistry } from '../drops'
import { createSlotRegistry } from '../slots/slotRegistry'
import { createSlotViewRegistry, type SlotViewRegistry } from '../slot-views/slotViews'
import { createViewRegistry, type ViewRegistry } from '../views'
import { createPluginContext } from './context'
import { addListener, notifyListeners } from './events'
import { createCommandApi } from './commandApi'
import { createWhenReader, type WhenReader } from '../when/whenReader'
import {
  APP_KEYBINDINGS,
  isSharedCommand,
  type JsonValue,
  type SlotHost,
} from '@numen/plugin-sdk'
import { APP_COMMAND_OWNER } from '../commandId'
import {
  createKeybindingRegistry,
  type KeybindingSource,
} from '../keybindings/keybindingRegistry'
import { createPluginPaneWrapper } from './customTabKind'
import { registerReaderEntry, registerTabKindEntry } from './entries'
import {
  addEntry,
  disposeAllEntries,
  disposePluginEntry,
  setStarting,
  type HostRegistry,
} from './registry'
import type {
  EventPort,
  PluginContextHost,
  PluginHostDeps,
  RegistrationPort,
  SettingsPort,
  TabsPort,
} from './types'
import { createUiBridge, createVaultBridge, createWorkBridge } from './bridge/services'

export { createPluginPaneWrapper }

/** The target the editor activates its plugins in. */
const EDITOR_TARGET = 'desktop.editor'

function getSettingsViewKey(pluginId: string, viewId: string): string {
  return `${pluginId}:${viewId}`
}

function createEventBridge(deps: PluginHostDeps, registry: HostRegistry): EventPort {
  return {
    onVaultChange: (pluginId, handler) =>
      addListener(registry.vaultListeners, { pluginId, handler }),
    onFileSelect: (pluginId, handler) => addListener(registry.fileListeners, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.vaultListeners, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.fileListeners, event, deps),
  }
}

/** The registrations a plugin makes into the host's own registry. */
type OwnRegistrations = Pick<
  RegistrationPort,
  | 'registerReader'
  | 'registerTabKind'
  | 'registerCommand'
  | 'registerSettingsView'
  | 'registerAgentProvider'
>

function createOwnRegistrations(deps: PluginHostDeps, registry: HostRegistry): OwnRegistrations {
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
      const holder = isSharedCommand(registration.id) ? registry.handlers : registry.commands
      const fullId = formatCommandId(pluginId, registration.id)
      if (holder.value.has(fullId)) {
        throw new Error(`Command "${fullId}" is already registered`)
      }
      return addEntry(holder, fullId, { pluginId, reg: registration })
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

function createSettingBridge(deps: PluginHostDeps): SettingsPort {
  return {
    getSetting: <T extends JsonValue = JsonValue>(pluginId: string, key: string) =>
      deps.getSetting ? (deps.getSetting(pluginId, key) as T | undefined) : undefined,
    onSettingChange: <T extends JsonValue = JsonValue>(
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
          handler as (v: JsonValue | undefined) => void | Promise<void>,
        ),
      }
    },
  }
}

function createRegistrationBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  shared: SharedRegistries,
): RegistrationPort {
  const { views, drops, slotViews } = shared
  return {
    ...createOwnRegistrations(deps, registry),
    registerSlotView: slotViews.registerSlotView,
    registerView: views.registerView,
    mountView: views.mountView,
    registerDropHandler: drops.registerDropHandler,
  }
}

function createTabBridge(deps: PluginHostDeps): TabsPort {
  return {
    openTab: (kind, target) => deps.openTab(kind, target),
    navigateTab: (kind, target) => deps.navigateTab?.(kind, target),
    closeTab: (id) => deps.closeTab?.(id),
  }
}

/** The registries other than the host's own, which the context bridge delegates to. */
interface SharedRegistries {
  readonly views: ViewRegistry
  readonly drops: DropRegistry
  readonly slotViews: SlotViewRegistry
  readonly slots: SlotHost
}

function createHostContextBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  shared: SharedRegistries,
): PluginContextHost {
  return {
    app: deps.app ?? createAppInfo({ getPlatform: () => 'linux' }),
    target: EDITOR_TARGET,
    openExternal: (url) =>
      deps.openExternal
        ? deps.openExternal(url)
        : Promise.reject(new ReportedError('This window cannot open web links')),
    registrations: createRegistrationBridge(deps, registry, shared),
    settings: createSettingBridge(deps),
    events: createEventBridge(deps, registry),
    tabs: createTabBridge(deps),
    vault: createVaultBridge(deps),
    ui: createUiBridge(deps, {
      slots: shared.slots,
      drops: { canDrop: shared.drops.canDrop, drop: shared.drops.drop },
    }),
    work: createWorkBridge(deps),
  }
}

function createHostRegistry(): HostRegistry {
  return {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    handlers: shallowRef(new Map()),
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

function createKeybindingApi(deps: PluginHostDeps & { readonly whenReader: WhenReader }) {
  const keybindings = createKeybindingRegistry({
    isMac: () => deps.app?.platform === 'darwin',
    whenReader: deps.whenReader,
    report: (owner, message) => deps.reportSlotIssue?.(owner, message, 'warning'),
  })
  keybindings.register(APP_COMMAND_OWNER, APP_KEYBINDINGS)
  const owners = new Set<string>()
  const registerKeybindings = (pluginId: string, sources: readonly KeybindingSource[]) => {
    owners.add(pluginId)
    keybindings.register(pluginId, sources)
  }
  const unregisterKeybindings = (pluginId: string) => {
    owners.delete(pluginId)
    keybindings.unregister(pluginId)
  }
  return {
    keybindings,
    registerKeybindings,
    unregisterKeybindings,
    unregisterAllKeybindings: () => [...owners].forEach(unregisterKeybindings),
  }
}

function createContextApi(
  registry: HostRegistry,
  contextHost: PluginContextHost,
  registerKeybindings: (pluginId: string, sources: readonly KeybindingSource[]) => void,
) {
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
      registerKeybindings(pluginId, manifest.contributes?.keybindings ?? [])
    },
  }
}

export function createPluginHost(windowDeps: PluginHostDeps) {
  const whenReader = windowDeps.whenReader ?? createWhenReader()
  const deps = { ...windowDeps, whenReader }
  const registry = createHostRegistry()
  const slotViews = createSlotViewRegistry(
    deps.reportViewIssue ? { reportIssue: deps.reportViewIssue } : {},
  )
  const { keybindings, registerKeybindings, unregisterKeybindings, unregisterAllKeybindings } =
    createKeybindingApi(deps)
  const commandApi = createCommandApi(deps, registry, keybindings)
  const findSlotCommand = deps.createSlotCommandFinder?.({
    getPluginCommands: commandApi.getPaletteCommands,
    hasPluginItems: commandApi.hasCommandItems,
  })
  const slots = createSlotRegistry({
    getKeyFor: keybindings.getKeyFor,
    getEntries: () => deps.getSlotEntries?.(),
    findCommand: (command, target) => findSlotCommand?.(command, target),
    findView: slotViews.findView,
    ...(deps.reportSlotIssue ? { reportIssue: deps.reportSlotIssue } : {}),
    ...(deps.getSlotWhenValues ? { getWhenValues: deps.getSlotWhenValues } : {}),
    whenReader,
  })
  const contextHost = createHostContextBridge(deps, registry, {
    views: createViewRegistry(),
    drops: createDropRegistry(),
    slotViews,
    slots,
  })
  const parts = { registry, slotViews, deps }

  return {
    ...createContextApi(registry, contextHost, registerKeybindings),
    ...commandApi,
    registerKeybindings,
    unregisterKeybindings,
    /** The key of a command as declared, such as `mod+p`, for a hint; undefined where none is bound. */
    getKeyFor: keybindings.getKeyFor,
    ...createRegisteredApi(registry),
    getSetting: contextHost.settings.getSetting,
    onSettingChange: contextHost.settings.onSettingChange,
    /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
    notifyVaultChange: (event: VaultChangeEvent): void =>
      notifyListeners(registry.vaultListeners, event, deps),
    disposePlugin: (pluginId: string) => {
      disposePluginEntry(pluginId, parts)
      unregisterKeybindings(pluginId)
    },
    disposeAll: () => {
      disposeAllEntries(parts)
      unregisterAllKeybindings()
    },
    /** The entries of the slots of the window, as their owners read them. */
    slots,
    /** The view a slot of the window draws, or null while no plugin registered one; reactive. */
    getSlotView: slotViews.getSlotView,
  }
}

export type PluginHost = ReturnType<typeof createPluginHost>
