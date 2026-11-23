/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import type { SettingsViewRegistration, VaultChangeEvent } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { formatCommandId } from '../commandId'
import { createDropRegistry } from '../drops'
import { createEditorExtensionRegistry } from '../editorExtensions'
import { createSlotRegistry } from '../slots'
import { createSlotViewRegistry } from '../slot-views'
import { createPluginStyles, type PluginStyles } from '../styles'
import { createViewRegistry } from '../views'
import type { PluginParts } from '../host'
import { createPluginParts } from './context'
import { isNoteEditorKind } from './customTabBuffer'
import { addListener, notifyListeners, reportError, requestPluginStart } from './events'
import { createCommandApi } from './commandApi'
import { createWhenReader } from '../when'
import { APP_KEYBINDINGS, isSharedCommand } from '@numen/plugin-sdk'
import { APP_COMMAND_OWNER } from '../commandId'
import { createKeybindingRegistry, type KeybindingSource } from '../keybindings'
import { registerReaderEntry, registerTabKindEntry } from './entries'
import {
  addEntry,
  createHostRegistry,
  disposeAllEntries,
  disposePluginEntry,
  setStarting,
  type HostRegistry,
} from './registry'
import type { KeybindingApi, PluginHost, PluginLifecycle, RegisteredApi } from './hostApi'
import { createUiServiceBridge, createWorkServiceBridge } from './bridge/services'
import { createSettingBridge, createViewBridge, type SharedRegistries } from './bridge/contextHosts'
import type {
  ContextEventHost,
  ContextRegistrationHost,
  HostDeps,
  PluginContextHost,
  PluginHostDeps,
} from './types'

function getEntryKey(pluginId: string, id: string): string {
  return `${pluginId}:${id}`
}

function createEventBridge(
  deps: Pick<PluginHostDeps, 'window'>,
  registry: HostRegistry,
): ContextEventHost {
  return {
    onVaultChange: (pluginId, handler) =>
      addListener(registry.listeners.vault, { pluginId, handler }),
    onFileSelect: (pluginId, handler) =>
      addListener(registry.listeners.file, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.listeners.vault, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.listeners.file, event, deps),
  }
}

function createRegistrationBridge(
  deps: Pick<HostDeps, 'window' | 'kinds' | 'tabs' | 'vault'>,
  registry: HostRegistry,
  styles: PluginStyles,
): ContextRegistrationHost {
  const entryDeps = { ...deps, styles }
  return {
    registerReader: (pluginId, registration) =>
      registerReaderEntry(entryDeps, registry.kinds, {
        pluginId,
        registration,
        getContext: () => registry.plugins.contexts.get(pluginId)?.plugin,
      }),
    registerTabKind: (pluginId, registration) =>
      registerTabKindEntry(entryDeps, registry.kinds, {
        pluginId,
        registration,
        getContext: () => registry.plugins.contexts.get(pluginId)?.plugin,
        isNoteEditor: isNoteEditorKind(
          registry.plugins.contextManifests?.value.get(pluginId),
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

function createHostContextBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  shared: SharedRegistries,
): PluginContextHost {
  return {
    registration: createRegistrationBridge(deps, registry, shared.styles),
    views: createViewBridge(shared),
    settings: createSettingBridge(deps),
    events: createEventBridge(deps, registry),
    tabs: deps.tabs,
    vault: deps.vault,
    ui: createUiServiceBridge(deps, {
      slots: shared.slots,
      manifests: registry.plugins.manifests,
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
  plugins.findPluginsOf((one) => one.id === pluginId && one.settingsViews.length > 0).length > 0

function createRegisteredApi(
  deps: Pick<PluginHostDeps, 'plugins' | 'window'>,
  registry: HostRegistry,
): RegisteredApi {
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
    getRegisteredReaders: () => [...registry.kinds.readers.values()].map((r) => r.reg),
    getRegisteredTabKinds: () => [...registry.kinds.tabKinds.values()].map((t) => t.reg),
    getRegisteredCommands: () => [...registry.commands.value.values()].map((c) => c.reg),
  }
}

function createKeybindingApi({
  window,
  commands,
  slots,
}: Pick<HostDeps, 'window' | 'commands' | 'slots'>): KeybindingApi {
  const keybindings = createKeybindingRegistry({
    isMac: () => window.app.platform === 'darwin',
    whenReader: commands.whenReader,
    report: (owner, message) => slots.reportSlotIssue(owner, message, 'warning'),
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

type EntryParts = Parameters<typeof disposeAllEntries>[0]

interface LifecycleParts {
  readonly registry: HostRegistry
  readonly contextHost: PluginContextHost
  readonly keybindings: KeybindingApi
  readonly entries: EntryParts
}

function createLifecycleApi({
  registry,
  contextHost,
  keybindings,
  entries,
}: LifecycleParts): PluginLifecycle {
  return {
    createContext: (pluginId: string, manifest?: Manifest): PluginParts => {
      let parts = registry.plugins.contexts.get(pluginId)
      if (!parts) {
        const effectiveManifest = manifest ?? registry.plugins.manifests.get(pluginId)
        parts = createPluginParts(pluginId, contextHost, effectiveManifest)
        registry.plugins.contexts.set(pluginId, parts)
        if (effectiveManifest && registry.plugins.contextManifests) {
          registry.plugins.contextManifests.value = new Map(
            registry.plugins.contextManifests.value,
          ).set(pluginId, effectiveManifest)
        }
        registry.plugins.disposed.delete(pluginId)
        registry.plugins.activated.value = new Set(registry.plugins.activated.value).add(pluginId)
      }
      return parts
    },
    markPluginStarting: (pluginId: string): void => setStarting(registry.plugins, pluginId, true),
    markPluginStarted: (pluginId: string): void => setStarting(registry.plugins, pluginId, false),
    registerManifest: (pluginId: string, manifest: Manifest): void => {
      if (registry.plugins.contexts.has(pluginId)) {
        throw new Error(
          `Cannot replace the manifest for plugin "${pluginId}" while its context is cached`,
        )
      }
      registry.plugins.manifests.set(pluginId, manifest)
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
    watchSetting: contextHost.settings.watchSetting,
  }
}

export function createPluginHost(windowDeps: PluginHostDeps): PluginHost {
  const whenReader = windowDeps.commands.whenReader ?? createWhenReader()
  const styles = createPluginStyles()
  const deps: HostDeps = { ...windowDeps, commands: { ...windowDeps.commands, whenReader } }
  const { slots: slotDeps } = deps
  const registry = createHostRegistry()
  const slotViews = createSlotViewRegistry({ reportIssue: slotDeps.reportViewIssue, styles })
  const keybindings = createKeybindingApi(deps)
  const commands = createCommandApi(deps, registry, keybindings.registry)
  const findSlotCommand = slotDeps.createSlotCommandFinder({
    getPluginCommands: commands.getPaletteCommands,
    hasPluginItems: commands.hasKnownItems,
  })
  const readSlotEntries = () =>
    slotDeps.getSlotEntries(registry.plugins.contextManifests?.value ?? new Map())
  const slots = createSlotRegistry({
    getKeyFor: keybindings.getKeyFor,
    getEntries: readSlotEntries,
    drawers: {
      getPoints: () =>
        slotDeps.getSlotPoints(registry.plugins.contextManifests?.value ?? new Map()),
      getLayout: slotDeps.getSlotLayout,
    },
    findCommand: findSlotCommand,
    findView: slotViews.findView,
    reportIssue: slotDeps.reportSlotIssue,
    when: {
      getValues: slotDeps.getSlotWhenValues,
      reader: whenReader,
    },
    requestPlugin: (pluginId) => requestPluginStart(deps, pluginId),
  })
  const contextHost = createHostContextBridge(deps, registry, {
    views: createViewRegistry({ ...deps.plugins, styles }),
    drops: createDropRegistry(deps.plugins),
    editorExtensions: createEditorExtensionRegistry({
      reportError: (pluginId, error) => reportError(deps, pluginId, error),
    }),
    slotViews,
    slots,
    styles,
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
      notifyListeners(registry.listeners.vault, event, deps),
  }
}
