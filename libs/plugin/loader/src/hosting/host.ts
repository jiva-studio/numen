/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef } from 'vue'
import type {
  SettingsViewRegistration,
  VaultChangeEvent,
  DropRegistry,
  JsonValue,
  SlotHost,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { formatCommandId } from '../commandId'
import { createDropRegistry } from '../drops'
import { createSlotRegistry, type DeclaredSlotEntry } from '../slots'
import { createSlotViewRegistry, type SlotViewRegistry } from '../slot-views'
import { createViewRegistry, type ViewRegistry } from '../views'
import type { PluginParts } from '../host'
import { createPluginParts } from './context'
import { isNoteEditorKind } from './customTabBuffer'
import { addListener, notifyListeners, requestPluginStart } from './events'
import { createCommandApi } from './commandApi'
import { createWhenReader } from '../when'
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
import {
  createUiServiceBridge,
  createVaultServiceBridge,
  createWorkServiceBridge,
} from './bridge/services'
import type {
  ContextEventHost,
  ContextRegistrationHost,
  ContextSettingHost,
  ContextTabHost,
  ContextViewHost,
  HostDeps,
  HostPlugins,
  HostSlots,
  PluginContextHost,
  PluginHostDeps,
} from './types'

export { createPluginPaneWrapper }

function getEntryKey(pluginId: string, id: string): string {
  return `${pluginId}:${id}`
}

function createEventBridge(
  deps: Pick<PluginHostDeps, 'window'>,
  registry: HostRegistry,
): ContextEventHost {
  return {
    onVaultChange: (pluginId, handler) =>
      addListener(registry.vaultListeners, { pluginId, handler }),
    onFileSelect: (pluginId, handler) => addListener(registry.fileListeners, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.vaultListeners, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.fileListeners, event, deps),
  }
}

function createRegistrationBridge(
  deps: Pick<PluginHostDeps, 'window' | 'kinds' | 'tabs' | 'vault'>,
  registry: HostRegistry,
): ContextRegistrationHost {
  return {
    registerReader: (pluginId, registration) =>
      registerReaderEntry(deps, registry, {
        pluginId,
        registration,
        getContext: () => registry.contexts.get(pluginId)?.plugin,
      }),
    registerTabKind: (pluginId, registration) =>
      registerTabKindEntry(deps, registry, {
        pluginId,
        registration,
        getContext: () => registry.contexts.get(pluginId)?.plugin,
        isNoteEditor: isNoteEditorKind(
          registry.contextManifests?.value.get(pluginId),
          registration.kind,
        ),
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

const NO_PLUGINS: HostPlugins = {}

const NO_SLOTS: HostSlots = {}

function createSettingBridge({
  plugins = NO_PLUGINS,
}: Pick<PluginHostDeps, 'plugins'>): ContextSettingHost {
  return {
    getSetting: <T extends JsonValue = JsonValue>(pluginId: string, key: string) =>
      plugins.getSetting ? (plugins.getSetting(pluginId, key) as T | undefined) : undefined,
    onSettingChange: <T extends JsonValue = JsonValue>(
      pluginId: string,
      key: string,
      handler: (value: T) => void | Promise<void>,
    ) => {
      // Without a host subscription there is nothing to unsubscribe.
      if (!plugins.onSettingChange) return { dispose: () => undefined }
      return {
        dispose: plugins.onSettingChange(
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

function createTabBridge({ tabs }: Pick<PluginHostDeps, 'tabs'>): ContextTabHost {
  return {
    openTab: (kind, target) => tabs?.openTab?.(kind, target),
    navigateTab: (kind, target) => tabs?.navigateTab?.(kind, target),
    closeTab: (id) => tabs?.closeTab?.(id),
  }
}

/** The registries other than the host's own, which the context bridge delegates to. */
interface SharedRegistries {
  readonly views: ViewRegistry
  readonly drops: DropRegistry
  readonly slotViews: SlotViewRegistry
  readonly slots: SlotHost
  readonly readSlotEntries: () => readonly DeclaredSlotEntry[] | undefined
}

function createHostContextBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  shared: SharedRegistries,
): PluginContextHost {
  return {
    registration: createRegistrationBridge(deps, registry),
    views: createViewBridge(shared),
    settings: createSettingBridge(deps),
    events: createEventBridge(deps, registry),
    tabs: createTabBridge(deps),
    vault: createVaultServiceBridge(deps),
    ui: createUiServiceBridge(deps, {
      slots: shared.slots,
      manifests: registry.manifests,
      readSlotEntries: shared.readSlotEntries,
    }),
    work: createWorkServiceBridge(deps),
  }
}

function readSettingsViews(
  registry: HostRegistry,
  pluginId: string,
): ReadonlyMap<string, SettingsViewRegistration> {
  const views = new Map<string, SettingsViewRegistration>()
  for (const item of registry.settingsViews.value.values()) {
    if (item.pluginId === pluginId) views.set(item.reg.id, item.reg)
  }
  return views
}

const hasDeclaredSettingsViews = (
  { plugins }: Pick<PluginHostDeps, 'plugins'>,
  pluginId: string,
): boolean =>
  (plugins?.findPluginsOf?.((one) => one.id === pluginId && one.settingsViews.length > 0) ?? [])
    .length > 0

function createRegisteredApi(
  deps: Pick<PluginHostDeps, 'plugins' | 'window'>,
  registry: HostRegistry,
) {
  return {
    /** The settings views a plugin registered, by view id; reactive. */
    getRegisteredSettingsViews: (pluginId: string) => readSettingsViews(registry, pluginId),
    /**
     * The settings views of a plugin for its settings page, by view id; reactive. A plugin that
     * declares settings views and has registered none starts, and its views follow once it has.
     */
    getSettingsViews: (pluginId: string): ReadonlyMap<string, SettingsViewRegistration> => {
      const views = readSettingsViews(registry, pluginId)
      if (views.size === 0 && hasDeclaredSettingsViews(deps, pluginId))
        requestPluginStart(deps, pluginId)
      return views
    },
    getRegisteredReaders: () => [...registry.readers.values()].map((r) => r.reg),
    getRegisteredTabKinds: () => [...registry.tabKinds.values()].map((t) => t.reg),
    getRegisteredCommands: () => [...registry.commands.value.values()].map((c) => c.reg),
  }
}

function createKeybindingApi({
  window,
  commands,
  slots,
}: Pick<HostDeps, 'window' | 'commands' | 'slots'>) {
  const keybindings = createKeybindingRegistry({
    isMac: () => window.app?.platform === 'darwin',
    whenReader: commands.whenReader,
    report: (owner, message) => slots?.reportSlotIssue?.(owner, message, 'warning'),
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
    createContext: (pluginId: string, manifest?: Manifest): PluginParts => {
      let parts = registry.contexts.get(pluginId)
      if (!parts) {
        const effectiveManifest = manifest ?? registry.manifests.get(pluginId)
        parts = createPluginParts(pluginId, contextHost, effectiveManifest)
        registry.contexts.set(pluginId, parts)
        if (effectiveManifest && registry.contextManifests) {
          registry.contextManifests.value = new Map(registry.contextManifests.value).set(
            pluginId,
            effectiveManifest,
          )
        }
        registry.disposed.delete(pluginId)
        registry.activated.value = new Set(registry.activated.value).add(pluginId)
      }
      return parts
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
  const whenReader = windowDeps.commands?.whenReader ?? createWhenReader()
  const deps: HostDeps = { ...windowDeps, commands: { ...windowDeps.commands, whenReader } }
  const { slots: slotDeps = NO_SLOTS } = deps
  const registry: HostRegistry = {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    handlers: shallowRef(new Map()),
    settingsViews: shallowRef(new Map()),
    contexts: new Map(),
    manifests: new Map(),
    contextManifests: shallowRef(new Map()),
    activated: shallowRef(new Set()),
    starting: shallowRef(new Set()),
    disposed: new Set(),
    reportedIssues: new Set(),
    vaultListeners: new Set(),
    fileListeners: new Set(),
  }
  const slotViews = createSlotViewRegistry(
    slotDeps.reportViewIssue ? { reportIssue: slotDeps.reportViewIssue } : {},
  )
  const keybindings = createKeybindingApi(deps)
  const commands = createCommandApi(deps, registry, keybindings.registry)
  const findSlotCommand = slotDeps.createSlotCommandFinder?.({
    getPluginCommands: commands.getPaletteCommands,
    hasPluginItems: commands.hasKnownItems,
  })
  const readSlotEntries = () =>
    slotDeps.getSlotEntries?.(registry.contextManifests?.value ?? new Map())
  const slots = createSlotRegistry({
    getKeyFor: keybindings.getKeyFor,
    getEntries: readSlotEntries,
    findCommand: (command, target) => findSlotCommand?.(command, target),
    findView: slotViews.findView,
    ...(slotDeps.reportSlotIssue ? { reportIssue: slotDeps.reportSlotIssue } : {}),
    ...(slotDeps.getSlotWhenValues ? { getWhenValues: slotDeps.getSlotWhenValues } : {}),
    whenReader,
    requestPlugin: (pluginId) => requestPluginStart(deps, pluginId),
  })
  const contextHost = createHostContextBridge(deps, registry, {
    views: createViewRegistry(deps.plugins),
    drops: createDropRegistry(deps.plugins),
    slotViews,
    slots,
    readSlotEntries,
  })
  const entries = { registry, slotViews, deps }

  return {
    lifecycle: createLifecycleApi({ registry, contextHost, keybindings, entries }),
    commands,
    keybindings,
    registered: createRegisteredApi(deps, registry),
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
