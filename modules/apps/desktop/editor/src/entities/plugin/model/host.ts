/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef } from 'vue'
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  PluginManifest,
  FileSelectEvent,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import { createPluginContext } from './context'
import { addListener, notifyListeners, runCommand, type PluginListener } from './events'
import { extractPaletteCommands } from './palette'
import { matchesShortcut } from './shortcuts'
import { createPluginPaneWrapper, createReaderTabKind, createCustomTabKind } from './customTabKind'
import type { PluginContextHost, PluginHostDeps } from './types'

export { createPluginPaneWrapper }

interface RegisteredReader {
  pluginId: string
  reg: ReaderRegistration
  cleanup: () => void
}

interface RegisteredTabKind {
  pluginId: string
  reg: TabKindRegistration
  cleanup: () => void
}

interface RegisteredCommand {
  pluginId: string
  reg: CommandRegistration
}

interface RegisteredAgentProvider {
  pluginId: string
  reg: AgentProviderRegistration
}

interface HostRegistry {
  readonly vaultListeners: Set<PluginListener<VaultChangeEvent>>
  readonly fileListeners: Set<PluginListener<FileSelectEvent>>
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly agentProviders: Map<string, RegisteredAgentProvider>
  readonly contexts: Map<string, PluginContext>
  readonly manifests: Map<string, PluginManifest>
}

interface PluginItemRegistration<T> {
  readonly pluginId: string
  readonly registration: T
  readonly getContext?: () => PluginContext | undefined
}

function clearSubscriptions(ctx?: PluginContext): void {
  if (!ctx) return
  while (ctx.subscriptions.length > 0) {
    const sub = ctx.subscriptions.pop()
    sub?.dispose()
  }
}

function cleanupMapItems(
  items: Map<string, { pluginId: string; cleanup: () => void }>,
  pluginId: string,
): void {
  for (const item of [...items.values()]) {
    if (item.pluginId === pluginId) item.cleanup()
  }
}

function cleanupProviderItems(items: Map<string, { pluginId: string }>, pluginId: string): void {
  for (const [id, item] of items) {
    if (item.pluginId === pluginId) items.delete(id)
  }
}

function disposePluginEntry(pluginId: string, registry: HostRegistry, deps?: PluginHostDeps): void {
  deps?.closeTabsForPlugin?.(pluginId)
  clearSubscriptions(registry.contexts.get(pluginId))
  registry.contexts.delete(pluginId)
  registry.manifests.delete(pluginId)
  cleanupMapItems(registry.readers, pluginId)
  cleanupMapItems(registry.tabKinds, pluginId)
  cleanupProviderItems(registry.agentProviders, pluginId)

  const nextCommands = new Map(registry.commands.value)
  for (const [id, item] of nextCommands) {
    if (item.pluginId === pluginId) nextCommands.delete(id)
  }
  registry.commands.value = nextCommands
}

function resolveCommand(
  commands: ReadonlyMap<string, RegisteredCommand>,
  id: string,
): RegisteredCommand | undefined {
  const direct = commands.get(id)
  if (direct) return direct
  if (id.startsWith('plugin:')) {
    return commands.get(id.slice('plugin:'.length))
  }
  return undefined
}

function registerReaderEntry(
  deps: PluginHostDeps,
  readers: Map<string, RegisteredReader>,
  item: PluginItemRegistration<ReaderRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const kindId = registration.id.startsWith('reader:')
    ? registration.id
    : `reader:${registration.id}`
  const cleanups: (() => void)[] = [
    deps.registerKind(createReaderTabKind(pluginId, registration, getContext)),
  ]
  for (const ext of registration.fileExtensions) {
    cleanups.push(
      deps.registerExtensionReader(ext, (path, spans) => {
        void deps.openTab(kindId, path).then((tabId) => {
          if (spans && spans.length > 0 && deps.getTabState) {
            const state = deps.getTabState(tabId) as
              | { spans?: unknown; focusSpans?: (...s: readonly unknown[]) => Promise<void> }
              | undefined
            if (state) {
              state.spans = spans
              void state.focusSpans?.(...spans)
            }
          }
        })
      }),
    )
  }
  let isDisposed = false
  const cleanup = () => {
    if (isDisposed) return
    isDisposed = true
    for (const fn of cleanups) fn()
    readers.delete(registration.id)
  }
  readers.set(registration.id, { pluginId, reg: registration, cleanup })
  return { dispose: cleanup }
}

function extractWhere(options?: unknown): unknown {
  if (typeof options === 'string') return options
  if (typeof options === 'object' && options !== null && 'where' in options) {
    return (options as { where?: unknown }).where
  }
  return undefined
}

const BUILTIN_TAB_KINDS = new Set(['deck', 'stencil', 'preset'])

function registerTabKindEntry(
  deps: PluginHostDeps,
  tabKinds: Map<string, RegisteredTabKind>,
  item: PluginItemRegistration<TabKindRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  if (BUILTIN_TAB_KINDS.has(registration.kind)) {
    return { dispose: () => {} }
  }
  const cleanups: (() => void)[] = [
    deps.registerKind(createCustomTabKind(pluginId, registration, { ...deps, getContext })),
  ]
  if (deps.registerEditor) {
    const unreg = deps.registerEditor(
      registration.kind,
      (path: string, _title?: string, options?: unknown) => {
        const where = extractWhere(options)
        if (where === 'beside' && deps.openTabBeside) {
          void deps.openTabBeside(registration.kind, path)
        } else {
          void deps.openTab(registration.kind, path)
        }
      },
    )
    if (typeof unreg === 'function') cleanups.push(unreg)
  }
  let isDisposed = false
  const cleanup = () => {
    if (isDisposed) return
    isDisposed = true
    for (const fn of cleanups) fn()
    tabKinds.delete(registration.kind)
  }
  tabKinds.set(registration.kind, { pluginId, reg: registration, cleanup })
  return { dispose: cleanup }
}

function createHostContextBridge(deps: PluginHostDeps, registry: HostRegistry): PluginContextHost {
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
      const next = new Map(registry.commands.value)
      next.set(registration.id, { pluginId, reg: registration })
      registry.commands.value = next
      return {
        dispose: () => {
          const updated = new Map(registry.commands.value)
          updated.delete(registration.id)
          registry.commands.value = updated
        },
      }
    },
    registerAgentProvider: (pluginId, registration) => {
      registry.agentProviders.set(registration.id, { pluginId, reg: registration })
      return {
        dispose: () => {
          registry.agentProviders.delete(registration.id)
        },
      }
    },
    getSetting: <T = unknown>(pluginId: string, key: string) =>
      deps.getSetting ? (deps.getSetting(pluginId, key) as T | undefined) : undefined,
    onSettingChange: <T = unknown>(
      pluginId: string,
      key: string,
      handler: (value: T) => void | Promise<void>,
    ) => {
      if (!deps.onSettingChange) return { dispose: () => {} }
      return {
        dispose: deps.onSettingChange(
          pluginId,
          key,
          handler as (v: unknown) => void | Promise<void>,
        ),
      }
    },
    onVaultChange: (pluginId, handler) =>
      addListener(registry.vaultListeners, { pluginId, handler }),
    onFileSelect: (pluginId, handler) =>
      addListener(registry.fileListeners, { pluginId, handler }),
    emitVaultChange: (_pluginId, event) => notifyListeners(registry.vaultListeners, event, deps),
    emitFileSelect: (_pluginId, event) => notifyListeners(registry.fileListeners, event, deps),
    openTab: (kind: string, at?: string) => deps.openTab(kind, at),
    closeTab: (id: string) => deps.closeTab?.(id),
  }
}

function disposeAllEntries(registry: HostRegistry, deps: PluginHostDeps) {
  const allPluginIds = new Set<string>([
    ...registry.contexts.keys(),
    ...[...registry.readers.values()].map((r) => r.pluginId),
    ...[...registry.tabKinds.values()].map((t) => t.pluginId),
    ...[...registry.commands.value.values()].map((c) => c.pluginId),
    ...[...registry.agentProviders.values()].map((p) => p.pluginId),
  ])
  for (const pluginId of allPluginIds) disposePluginEntry(pluginId, registry, deps)
}

export function createPluginHost(deps: PluginHostDeps) {
  const registry: HostRegistry = {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    agentProviders: new Map(),
    contexts: new Map(),
    manifests: new Map(),
    vaultListeners: new Set(),
    fileListeners: new Set(),
  }

  const contextHost = createHostContextBridge(deps, registry)
  const disposePlugin = (pluginId: string) => disposePluginEntry(pluginId, registry, deps)

  return {
    contextHost,
    createContext: (pluginId: string, manifest?: PluginManifest): PluginContext => {
      let ctx = registry.contexts.get(pluginId)
      if (!ctx) {
        const effectiveManifest = manifest ?? registry.manifests.get(pluginId)
        ctx = createPluginContext(pluginId, contextHost, effectiveManifest)
        registry.contexts.set(pluginId, ctx)
      }
      return ctx
    },
    registerManifest: (pluginId: string, manifest: PluginManifest): void => {
      if (registry.contexts.has(pluginId)) {
        throw new Error(
          `Cannot replace the manifest for plugin "${pluginId}" while its context is cached`,
        )
      }
      registry.manifests.set(pluginId, manifest)
    },
    getPaletteCommands: () => extractPaletteCommands(registry.commands.value),
    hasCommand: (id: string): boolean => resolveCommand(registry.commands.value, id) !== undefined,
    executeCommand: async (id: string): Promise<boolean> => {
      const entry = resolveCommand(registry.commands.value, id)
      if (!entry) return false
      await runCommand(deps, entry.pluginId, entry.reg)
      return true
    },
    handleShortcut: (event: KeyboardEvent): boolean => {
      for (const { pluginId, reg } of registry.commands.value.values()) {
        if (matchesShortcut(event, reg.shortcut)) {
          void runCommand(deps, pluginId, reg)
          return true
        }
      }
      return false
    },
    getSetting: contextHost.getSetting,
    onSettingChange: contextHost.onSettingChange,
    disposePlugin,
    disposeAll: () => disposeAllEntries(registry, deps),
    getRegisteredReaders: () => [...registry.readers.values()].map((r) => r.reg),
    getRegisteredTabKinds: () => [...registry.tabKinds.values()].map((t) => t.reg),
    getRegisteredCommands: () => [...registry.commands.value.values()].map((c) => c.reg),
  }
}

export type PluginHost = ReturnType<typeof createPluginHost>
