/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef } from 'vue'
import type {
  PluginContext,
  SettingsViewRegistration,
  VaultChangeEvent,
  DropRegistry,
  JsonValue,
  SlotHost,
} from '@numen/plugin-sdk'
import { ReportedError, type Manifest } from '@numen/core-client'
import { createAppInfo } from '../appInfo'
import { formatCommandId } from '../commandId'
import { createDropRegistry } from '../drops'
import { createSlotRegistry } from '../slots'
import { createSlotViewRegistry, type SlotViewRegistry } from '../slot-views'
import { createViewRegistry, type ViewRegistry } from '../views'
import { createPluginContext } from './context'
import { addListener, notifyListeners } from './events'
import { createCommandApi } from './commandApi'
import { createWhenReader, type WhenReader } from '../when'
import { APP_KEYBINDINGS, isSharedCommand } from '@numen/plugin-sdk'
import { APP_COMMAND_OWNER } from '../commandId'
import { createKeybindingRegistry, type KeybindingSource } from '../keybindings'
import { createPluginPaneWrapper } from './customTabKind'
import { registerReaderEntry, registerTabKindEntry } from './entries'
import {
  addEntry,
  disposeAllEntries,
  disposePluginEntry,
  setStarting,
  type HostRegistry,
} from './registry'
import { createVaultServiceBridge, createWindowServiceBridge } from './bridge/services'
import type {
  ContextEventHost,
  ContextRegistrationHost,
  ContextSettingHost,
  ContextTabHost,
  ContextViewHost,
  PluginContextHost,
  PluginHostDeps,
} from './types'

export { createPluginPaneWrapper }

/** The target the editor activates its plugins in. */
const EDITOR_TARGET = 'desktop.editor'

function getEntryKey(pluginId: string, id: string): string {
  return `${pluginId}:${id}`
}

function createEventBridge(deps: PluginHostDeps, registry: HostRegistry): ContextEventHost {
  return {
    onVaultChange: (pluginId, handler) =>
      addListener(registry.vaultListeners, { pluginId, handler }),
    onFileSelect: (pluginId, handler) => addListener(registry.fileListeners, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.vaultListeners, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.fileListeners, event, deps),
  }
}

function createRegistrationBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
): ContextRegistrationHost {
  return {
    registerReader: (pluginId, registration) =>
      registerReaderEntry(deps, registry, {
        pluginId,
        registration,
        getContext: () => registry.contexts.get(pluginId),
      }),
    registerTabKind: (pluginId, registration) =>
      registerTabKindEntry(deps, registry, {
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
      addEntry(registry.settingsViews, getEntryKey(pluginId, registration.id), {
        pluginId,
        reg: registration,
      }),
  }
}

function createSettingBridge(deps: PluginHostDeps): ContextSettingHost {
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

function createViewBridge(shared: SharedRegistries): ContextViewHost {
  const { views, drops, slotViews } = shared
  return {
    registerSlotView: slotViews.registerSlotView,
    registerView: views.registerView,
    mountView: views.mountView,
    registerDropHandler: drops.registerDropHandler,
    drops: { canDrop: drops.canDrop, drop: drops.drop },
  }
}

function createTabBridge(deps: PluginHostDeps): ContextTabHost {
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
    info: {
      app: deps.app ?? createAppInfo({ getPlatform: () => 'linux' }),
      target: EDITOR_TARGET,
      openExternal: (url) =>
        deps.openExternal
          ? deps.openExternal(url)
          : Promise.reject(new ReportedError('This window cannot open web links')),
    },
    registration: createRegistrationBridge(deps, registry),
    views: createViewBridge(shared),
    settings: createSettingBridge(deps),
    events: createEventBridge(deps, registry),
    tabs: createTabBridge(deps),
    vault: createVaultServiceBridge(deps),
    window: createWindowServiceBridge(deps, shared.slots, registry.manifests),
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
    registry: keybindings,
    registerKeybindings,
    unregisterKeybindings,
    unregisterAllKeybindings: () => [...owners].forEach(unregisterKeybindings),
    /** The key of a command as declared, such as `mod+p`, for a hint; undefined where none is bound. */
    getKeyFor: keybindings.getKeyFor,
  }
}

type KeybindingApi = ReturnType<typeof createKeybindingApi>

type EntryParts = Parameters<typeof disposeAllEntries>[0]

interface LifecycleParts {
  readonly registry: HostRegistry
  readonly contextHost: PluginContextHost
  readonly keybindings: KeybindingApi
  readonly entries: EntryParts
}

function createLifecycleApi({ registry, contextHost, keybindings, entries }: LifecycleParts) {
  return {
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
      keybindings.registerKeybindings(pluginId, manifest.contributes?.keybindings ?? [])
    },
    disposePlugin: (pluginId: string) => {
      disposePluginEntry(pluginId, entries)
      keybindings.unregisterKeybindings(pluginId)
    },
    disposeAll: () => {
      disposeAllEntries(entries)
      keybindings.unregisterAllKeybindings()
    },
    getSetting: contextHost.settings.getSetting,
    onSettingChange: contextHost.settings.onSettingChange,
  }
}

export function createPluginHost(windowDeps: PluginHostDeps) {
  const whenReader = windowDeps.whenReader ?? createWhenReader()
  const deps = { ...windowDeps, whenReader }
  const registry: HostRegistry = {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    handlers: shallowRef(new Map()),
    settingsViews: shallowRef(new Map()),
    contexts: new Map(),
    manifests: new Map(),
    activated: shallowRef(new Set()),
    starting: shallowRef(new Set()),
    disposed: new Set(),
    reportedIssues: new Set(),
    vaultListeners: new Set(),
    fileListeners: new Set(),
  }
  const slotViews = createSlotViewRegistry(
    deps.reportViewIssue ? { reportIssue: deps.reportViewIssue } : {},
  )
  const keybindings = createKeybindingApi(deps)
  const commands = createCommandApi(deps, registry, keybindings.registry)
  const findSlotCommand = deps.createSlotCommandFinder?.({
    getPluginCommands: commands.getPaletteCommands,
    hasPluginItems: commands.hasCommandItems,
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
  const entries = { registry, slotViews, deps }

  return {
    lifecycle: createLifecycleApi({ registry, contextHost, keybindings, entries }),
    commands,
    keybindings,
    registered: createRegisteredApi(registry),
    /** The entries of the slots of the window, as their owners read them. */
    slots,
    slotViews,
    contextHost,
    /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
    notifyVaultChange: (event: VaultChangeEvent): void =>
      notifyListeners(registry.vaultListeners, event, deps),
  }
}

export type PluginHost = ReturnType<typeof createPluginHost>
