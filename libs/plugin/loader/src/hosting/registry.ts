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
import type { Manifest } from '@numen/core-client'
import type { SlotViewRegistry } from '../slot-views'
import type { PluginListener } from './events'
import { disposeSubscriptions } from './guards/subscriptions'
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

export interface HostRegistry {
  readonly vaultListeners: Set<PluginListener<VaultChangeEvent>>
  readonly fileListeners: Set<PluginListener<FileSelectEvent>>
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly handlers: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly settingsViews: { value: ReadonlyMap<string, RegisteredSettingsView> }
  readonly contexts: Map<string, PluginContext>
  readonly manifests: Map<string, Manifest>
  readonly contextManifests?: { value: ReadonlyMap<string, Manifest> }
  readonly activated: { value: ReadonlySet<string> }
  readonly starting?: { value: ReadonlySet<string> }
  readonly disposed: Set<string>
  readonly reportedIssues: Set<string>
}

export interface PluginItemRegistration<T> {
  readonly pluginId: string
  readonly registration: T
  readonly getContext?: () => PluginContext | undefined
}

function clearSubscriptions(ctx?: PluginContext): void {
  if (ctx) disposeSubscriptions(ctx.plugin.subscriptions)
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
  readonly deps?: PluginHostDeps | undefined
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
export function setStarting(registry: HostRegistry, pluginId: string, isStarting: boolean): void {
  const { starting } = registry
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
  // The plugin is marked disposed before its subscriptions are cleared: each handler removed is read
  // at once by the parts that follow the commands, and an active plugin without it is reported.
  registry.disposed.add(pluginId)
  registry.activated.value = new Set([...registry.activated.value].filter((id) => id !== pluginId))
  setStarting(registry, pluginId, false)
  deps?.closeTabsForPlugin?.(pluginId)
  deps?.clearNotices?.(pluginId)
  clearSubscriptions(registry.contexts.get(pluginId))
  registry.contexts.delete(pluginId)
  registry.manifests.delete(pluginId)
  if (registry.contextManifests?.value.has(pluginId)) {
    const next = new Map(registry.contextManifests.value)
    next.delete(pluginId)
    registry.contextManifests.value = next
  }
  cleanupMapItems(registry.readers, pluginId)
  cleanupMapItems(registry.tabKinds, pluginId)
  removePluginItems(registry.commands, pluginId)
  removePluginItems(registry.handlers, pluginId)
  removePluginItems(registry.settingsViews, pluginId)
  slotViews?.disposePlugin(pluginId)
  removePluginListeners(registry.vaultListeners, pluginId)
  removePluginListeners(registry.fileListeners, pluginId)
}

export function disposeAllEntries(parts: DisposalParts & { readonly deps: PluginHostDeps }) {
  const { registry, slotViews } = parts
  const allPluginIds = new Set<string>([
    ...registry.contexts.keys(),
    ...[...registry.readers.values()].map((r) => r.pluginId),
    ...[...registry.tabKinds.values()].map((t) => t.pluginId),
    ...[...registry.commands.value.values()].map((c) => c.pluginId),
    ...[...registry.handlers.value.values()].map((c) => c.pluginId),
    ...[...registry.settingsViews.value.values()].map((v) => v.pluginId),
    ...(slotViews?.views.value ?? []).map((v) => v.pluginId),
  ])
  for (const pluginId of allPluginIds) disposePluginEntry(pluginId, parts)
}
