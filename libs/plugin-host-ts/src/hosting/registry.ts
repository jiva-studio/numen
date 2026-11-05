/**
 * What the plugin host holds for the plugins: the registrations of each kind, and how a plugin's
 * registrations are taken away with it.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  ContextMenuRegistration,
  ContextMenuTarget,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { StatusViewRegistry } from '../statusViews'
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

export interface RegisteredContextMenu {
  pluginId: string
  reg: ContextMenuRegistration
}

export interface RegisteredAgentProvider {
  pluginId: string
  reg: AgentProviderRegistration
}

export interface RegisteredSettingsView {
  pluginId: string
  reg: SettingsViewRegistration
}

export interface HostRegistry {
  readonly vaultListeners: Set<PluginListener<VaultChangeEvent>>
  readonly fileListeners: Set<PluginListener<FileSelectEvent>>
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly contextMenus: { value: ReadonlyMap<string, RegisteredContextMenu> }
  readonly settingsViews: { value: ReadonlyMap<string, RegisteredSettingsView> }
  readonly agentProviders: Map<string, RegisteredAgentProvider>
  readonly contexts: Map<string, PluginContext>
  readonly manifests: Map<string, Manifest>
  /** The plugins that have had a context made, which is when their frontend starts to register. */
  readonly activated: { value: ReadonlySet<string> }
  /** The plugins whose `activate` has not returned yet: their declared commands are not missing yet. */
  readonly starting?: { value: ReadonlySet<string> }
  /** The registrations in progress: a read they trigger reports nothing, as the plugin is not done registering. */
  readonly registering?: { count: number }
  /** The plugins that were disposed and have had no context made since: their declared commands are not listed. */
  readonly disposed: Set<string>
  /** The mismatches between a declaration and a handler already reported. */
  readonly reportedIssues: Set<string>
}

export interface PluginItemRegistration<T> {
  readonly pluginId: string
  readonly registration: T
  readonly getContext?: () => PluginContext | undefined
}

export function clearSubscriptions(ctx?: PluginContext): void {
  if (!ctx) return
  while (ctx.subscriptions.length > 0) {
    const sub = ctx.subscriptions.pop()
    // Stryker disable next-line OptionalChaining: subscriptions only hold Disposable objects
    sub?.dispose()
  }
}

export function cleanupMapItems(
  items: Map<string, { pluginId: string; cleanup: () => void }>,
  pluginId: string,
): void {
  for (const item of [...items.values()]) {
    if (item.pluginId === pluginId) item.cleanup()
  }
}

// Stryker disable all: agentProviders cleanup is internal state for future host provider lookup
export function cleanupProviderItems(
  items: Map<string, { pluginId: string }>,
  pluginId: string,
): void {
  for (const [id, item] of items) {
    if (item.pluginId === pluginId) items.delete(id)
  }
}
// Stryker restore all

/** What the disposal of a plugin's entries works on. */
export interface DisposalParts {
  readonly registry: HostRegistry
  readonly statusViews: StatusViewRegistry
  readonly deps?: PluginHostDeps | undefined
}

/** A copy of the map of a reactive holder without the entries of one plugin, set back in the holder. */
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

/** Runs a registration; a listing read that it triggers reports no issue. */
export function runRegistration<T>(registry: HostRegistry, register: () => T): T {
  const { registering } = registry
  if (!registering) return register()
  registering.count += 1
  try {
    return register()
  } finally {
    registering.count -= 1
  }
}

export function disposePluginEntry(pluginId: string, parts: DisposalParts): void {
  const { registry, statusViews, deps } = parts
  deps?.closeTabsForPlugin?.(pluginId)
  deps?.clearNotices?.(pluginId)
  clearSubscriptions(registry.contexts.get(pluginId))
  registry.contexts.delete(pluginId)
  registry.activated.value = new Set([...registry.activated.value].filter((id) => id !== pluginId))
  setStarting(registry, pluginId, false)
  registry.manifests.delete(pluginId)
  registry.disposed.add(pluginId)
  cleanupMapItems(registry.readers, pluginId)
  cleanupMapItems(registry.tabKinds, pluginId)
  cleanupProviderItems(registry.agentProviders, pluginId)
  removePluginItems(registry.commands, pluginId)
  removePluginItems(registry.contextMenus, pluginId)
  removePluginItems(registry.settingsViews, pluginId)
  statusViews.disposePlugin(pluginId)
}

function matchesSingleTarget(pattern: string, actual: ContextMenuTarget): boolean {
  if (pattern === 'all') return true
  if (pattern === 'root') return actual.path === '' || actual.path === null
  if (pattern === 'folder') return actual.isFolder
  if (pattern === 'file') return !actual.isFolder
  if (pattern === 'note') return !actual.isFolder && actual.isNote
  return false
}

export function matchesContextMenuTarget(
  targetPattern: ContextMenuRegistration['target'],
  actual: ContextMenuTarget,
): boolean {
  if (!targetPattern) return true
  if (typeof targetPattern === 'string') {
    return matchesSingleTarget(targetPattern, actual)
  }
  return targetPattern.some((p) => matchesSingleTarget(p, actual))
}

export function disposeAllEntries(
  registry: HostRegistry,
  statusViews: StatusViewRegistry,
  deps: PluginHostDeps,
) {
  const allPluginIds = new Set<string>([
    ...registry.contexts.keys(),
    ...[...registry.readers.values()].map((r) => r.pluginId),
    ...[...registry.tabKinds.values()].map((t) => t.pluginId),
    ...[...registry.commands.value.values()].map((c) => c.pluginId),
    ...[...registry.contextMenus.value.values()].map((c) => c.pluginId),
    ...[...registry.settingsViews.value.values()].map((v) => v.pluginId),
    ...[...registry.agentProviders.values()].map((p) => p.pluginId),
    ...statusViews.views.value.map((v) => v.pluginId),
  ])
  for (const pluginId of allPluginIds) disposePluginEntry(pluginId, { registry, statusViews, deps })
}
