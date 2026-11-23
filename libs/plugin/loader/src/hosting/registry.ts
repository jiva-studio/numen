/**
 * What the plugin host holds for the plugins: the registrations of each kind, and how a plugin's
 * registrations are taken away with it.
 */
import type {
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import { shallowRef } from 'vue'
import type { Manifest } from '@numen/core-client'
import type { HostedParts } from './context'
import type { SlotViewRegistry } from '../slot-views'
import type { PluginListener } from './events'
import type { PluginHostDeps } from './types'

export interface RegisteredReader {
  pluginId: string
  reg: ReaderRegistration
  cleanup: () => void
}

export interface RegisteredTabKind {
  pluginId: string
  reg: TabKindRegistration
  cleanup: () => void
}

export interface RegisteredCommand {
  pluginId: string
  reg: CommandRegistration
}

interface RegisteredSettingsView {
  pluginId: string
  reg: SettingsViewRegistration
}

/** The listeners of the plugins to a change of the vault and to the select of a file. */
interface PluginListeners {
  readonly vault: Set<PluginListener<VaultChangeEvent>>
  readonly file: Set<PluginListener<FileSelectEvent>>
}

/** The entries that own the ids of the tabs a window holds: the tab kinds and the readers. */
export interface TabKindOwners {
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
}

/** The plugins the host made a context for: their parts and manifests, and which are active, starting and stopped. */
export interface PluginRecords {
  readonly contexts: Map<string, HostedParts>
  readonly manifests: Map<string, Manifest>
  readonly contextManifests?: { value: ReadonlyMap<string, Manifest> }
  readonly activated: { value: ReadonlySet<string> }
  readonly starting?: { value: ReadonlySet<string> }
  readonly disposed: Set<string>
}

export interface HostRegistry {
  readonly listeners: PluginListeners
  readonly kinds: TabKindOwners
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly handlers: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly settingsViews: { value: ReadonlyMap<string, RegisteredSettingsView> }
  readonly plugins: PluginRecords
  readonly reportedIssues: Set<string>
}

/** An empty registry; the window reads a change of the holders that are refs. */
export function createHostRegistry(): HostRegistry {
  return {
    listeners: { vault: new Set(), file: new Set() },
    kinds: { readers: new Map(), tabKinds: new Map() },
    commands: shallowRef(new Map()),
    handlers: shallowRef(new Map()),
    settingsViews: shallowRef(new Map()),
    plugins: {
      contexts: new Map(),
      manifests: new Map(),
      contextManifests: shallowRef(new Map()),
      activated: shallowRef(new Set()),
      starting: shallowRef(new Set()),
      disposed: new Set(),
    },
    reportedIssues: new Set(),
  }
}

export interface PluginItemRegistration<T> {
  readonly pluginId: string
  readonly registration: T
  readonly getContext?: () => PluginContext | undefined
}

function cleanupMapItems(
  items: Map<string, { pluginId: string; cleanup: () => void }>,
  pluginId: string,
): void {
  for (const item of [...items.values()]) {
    if (item.pluginId === pluginId) item.cleanup()
  }
}

/** What the disposal of a plugin's entries works on. */
export interface DisposalParts {
  readonly registry: HostRegistry
  readonly slotViews?: Pick<SlotViewRegistry, 'disposePlugin' | 'views'> | undefined
  readonly deps?: Pick<PluginHostDeps, 'window' | 'kinds'> | undefined
}

function removePluginListeners<E>(listeners: Set<PluginListener<E>>, pluginId: string): void {
  for (const listener of [...listeners])
    if (listener.pluginId === pluginId) listeners.delete(listener)
}

function removePluginItems<T extends { pluginId: string }>(
  holder: { value: ReadonlyMap<string, T> },
  pluginId: string,
): void {
  const next = new Map(holder.value)
  for (const [key, item] of next) {
    if (item.pluginId === pluginId) next.delete(key)
  }
  holder.value = next
}

/** Adds a plugin to the plugins whose `activate` has not returned, or removes it. */
export function setStarting(plugins: PluginRecords, pluginId: string, isStarting: boolean): void {
  const { starting } = plugins
  if (!starting) return
  const next = new Set(starting.value)
  if (isStarting) next.add(pluginId)
  else next.delete(pluginId)
  starting.value = next
}

/** Puts an entry in a reactive map under a key; disposing it removes the entry if it is still there. */
export function addEntry<T>(
  holder: { value: ReadonlyMap<string, T> },
  key: string,
  entry: T,
): Disposable {
  holder.value = new Map(holder.value).set(key, entry)
  return {
    dispose: () => {
      if (holder.value.get(key) !== entry) return
      const updated = new Map(holder.value)
      updated.delete(key)
      holder.value = updated
    },
  }
}

export function disposePluginEntry(pluginId: string, parts: DisposalParts): void {
  const { registry, slotViews, deps } = parts
  const { plugins } = registry
  // The plugin is marked disposed before its subscriptions are cleared: each handler removed is read
  // at once by the parts that follow the commands, and an active plugin without it is reported.
  plugins.disposed.add(pluginId)
  plugins.activated.value = new Set([...plugins.activated.value].filter((id) => id !== pluginId))
  setStarting(plugins, pluginId, false)
  deps?.kinds.closeTabsForPlugin(pluginId)
  deps?.window.clearNotices(pluginId)
  plugins.contexts.get(pluginId)?.subscriptions.disposeAll()
  plugins.contexts.delete(pluginId)
  plugins.manifests.delete(pluginId)
  if (plugins.contextManifests?.value.has(pluginId)) {
    const next = new Map(plugins.contextManifests.value)
    next.delete(pluginId)
    plugins.contextManifests.value = next
  }
  cleanupMapItems(registry.kinds.readers, pluginId)
  cleanupMapItems(registry.kinds.tabKinds, pluginId)
  removePluginItems(registry.commands, pluginId)
  removePluginItems(registry.handlers, pluginId)
  removePluginItems(registry.settingsViews, pluginId)
  slotViews?.disposePlugin(pluginId)
  removePluginListeners(registry.listeners.vault, pluginId)
  removePluginListeners(registry.listeners.file, pluginId)
}

export function disposeAllEntries(
  parts: DisposalParts & { readonly deps: Pick<PluginHostDeps, 'window' | 'kinds'> },
) {
  const { registry, slotViews } = parts
  const allPluginIds = new Set<string>([
    ...registry.plugins.contexts.keys(),
    ...[...registry.kinds.readers.values()].map((r) => r.pluginId),
    ...[...registry.kinds.tabKinds.values()].map((t) => t.pluginId),
    ...[...registry.commands.value.values()].map((c) => c.pluginId),
    ...[...registry.handlers.value.values()].map((c) => c.pluginId),
    ...[...registry.settingsViews.value.values()].map((v) => v.pluginId),
    ...(slotViews?.views.value ?? []).map((v) => v.pluginId),
  ])
  for (const pluginId of allPluginIds) disposePluginEntry(pluginId, parts)
}
