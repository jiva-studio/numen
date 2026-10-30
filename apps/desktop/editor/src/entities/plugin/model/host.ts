/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { shallowRef, type Component } from 'vue'
import type {
  AgentProviderRegistration,
  CommandNote,
  CommandRegistration,
  ContextMenuRegistration,
  ContextMenuTarget,
  Disposable,
  PluginContext,
  PluginManifest,
  FileSelectEvent,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import { createDropRegistry, createViewRegistry, type ViewRegistry } from '@numen/plugin-host'
import type { DropRegistry } from '@numen/plugin-sdk'
import { createPluginContext } from './context'
import {
  addListener,
  notifyListeners,
  reportError,
  runCommand,
  type PluginListener,
} from './events'
import { extractPaletteCommands } from './palette'
import { matchesShortcut } from './shortcuts'
import { createPluginPaneWrapper, createReaderTabKind, createCustomTabKind } from './customTabKind'
import { createDeclaredTabKind, isDeclaredTabKind } from './pluginTabKind'
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

interface RegisteredContextMenu {
  pluginId: string
  reg: ContextMenuRegistration
}

interface RegisteredAgentProvider {
  pluginId: string
  reg: AgentProviderRegistration
}

interface RegisteredSettingsView {
  pluginId: string
  reg: SettingsViewRegistration
}

interface HostRegistry {
  readonly vaultListeners: Set<PluginListener<VaultChangeEvent>>
  readonly fileListeners: Set<PluginListener<FileSelectEvent>>
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly contextMenus: { value: ReadonlyMap<string, RegisteredContextMenu> }
  readonly settingsViews: { value: ReadonlyMap<string, RegisteredSettingsView> }
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
  deps?.clearNotices?.(pluginId)
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

  const nextContextMenus = new Map(registry.contextMenus.value)
  for (const [id, item] of nextContextMenus) {
    if (item.pluginId === pluginId) nextContextMenus.delete(id)
  }
  registry.contextMenus.value = nextContextMenus

  const nextSettingsViews = new Map(registry.settingsViews.value)
  for (const [key, item] of nextSettingsViews) {
    if (item.pluginId === pluginId) nextSettingsViews.delete(key)
  }
  registry.settingsViews.value = nextSettingsViews
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
  for (const ext of deps.getReaderExtensions?.(pluginId, registration.id) ?? []) {
    cleanups.push(
      deps.registerExtensionReader(
        ext,
        (path, selector) => {
          deps
            .openTab(kindId, path)
            .then((tabId) => {
              if (!selector || !deps.getTabState) return
              const state = deps.getTabState(tabId) as { selector?: unknown } | undefined
              // A new object each time, so a reader watching it moves to the same place again.
              if (state) state.selector = { ...selector }
            })
            .catch((error: unknown) => reportError(deps, pluginId, error))
        },
        registration.icon as Component | undefined,
      ),
    )
  }
  let isDisposed = false
  const entry: RegisteredReader = {
    pluginId,
    reg: registration,
    cleanup: () => {
      if (isDisposed) return
      isDisposed = true
      for (const fn of cleanups) fn()
      if (readers.get(registration.id) === entry) readers.delete(registration.id)
    },
  }
  readers.set(registration.id, entry)
  return { dispose: entry.cleanup }
}

function extractWhere(options?: unknown): unknown {
  if (typeof options === 'string') return options
  if (typeof options === 'object' && options !== null && 'where' in options) {
    return (options as { where?: unknown }).where
  }
  return undefined
}

function registerTabKindEntry(
  deps: PluginHostDeps,
  tabKinds: Map<string, RegisteredTabKind>,
  item: PluginItemRegistration<TabKindRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const kind = isDeclaredTabKind(registration)
    ? createDeclaredTabKind(pluginId, registration)
    : createCustomTabKind(pluginId, registration, { ...deps, getContext })
  const cleanups: (() => void)[] = [deps.registerKind(kind)]
  if (deps.registerEditor) {
    const unreg = deps.registerEditor(
      registration.kind,
      (path: string, _title?: string, options?: unknown) => {
        const where = extractWhere(options)
        if (where === 'split' && deps.openTabAdjacent) {
          deps
            .openTabAdjacent(registration.kind, path)
            .catch((error: unknown) => reportError(deps, pluginId, error))
        } else {
          deps
            .openTab(registration.kind, path)
            .catch((error: unknown) => reportError(deps, pluginId, error))
        }
      },
    )
    if (typeof unreg === 'function') cleanups.push(unreg)
  }
  let isDisposed = false
  const entry: RegisteredTabKind = {
    pluginId,
    reg: registration,
    cleanup: () => {
      if (isDisposed) return
      isDisposed = true
      for (const fn of cleanups) fn()
      if (tabKinds.get(registration.kind) === entry) tabKinds.delete(registration.kind)
    },
  }
  tabKinds.set(registration.kind, entry)
  return { dispose: entry.cleanup }
}

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

function createHostContextBridge(
  deps: PluginHostDeps,
  registry: HostRegistry,
  views: ViewRegistry,
  drops: DropRegistry,
): PluginContextHost {
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
      const entry = { pluginId, reg: registration }
      registry.commands.value = new Map(registry.commands.value).set(registration.id, entry)
      return {
        dispose: () => {
          if (registry.commands.value.get(registration.id) !== entry) return
          const updated = new Map(registry.commands.value)
          updated.delete(registration.id)
          registry.commands.value = updated
        },
      }
    },
    registerContextMenu: (pluginId, registration) => {
      const entry = { pluginId, reg: registration }
      registry.contextMenus.value = new Map(registry.contextMenus.value).set(registration.id, entry)
      return {
        dispose: () => {
          if (registry.contextMenus.value.get(registration.id) !== entry) return
          const updated = new Map(registry.contextMenus.value)
          updated.delete(registration.id)
          registry.contextMenus.value = updated
        },
      }
    },
    registerSettingsView: (pluginId, registration) => {
      const key = getSettingsViewKey(pluginId, registration.id)
      const entry = { pluginId, reg: registration }
      registry.settingsViews.value = new Map(registry.settingsViews.value).set(key, entry)
      return {
        dispose: () => {
          if (registry.settingsViews.value.get(key) !== entry) return
          const updated = new Map(registry.settingsViews.value)
          updated.delete(key)
          registry.settingsViews.value = updated
        },
      }
    },
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
    registerView: views.registerView,
    mountView: views.mountView,
    registerDropHandler: drops.registerDropHandler,
    drops: { canDrop: drops.canDrop, drop: drops.drop },
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
    ...createEventBridge(deps, registry),
    openTab: (kind: string, target?: string) => deps.openTab(kind, target),
    navigateTab: (kind: string, target: string) => deps.navigateTab?.(kind, target),
    closeTab: (id: string) => deps.closeTab?.(id),
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
  | 'getFiles'
  | 'getAgent'
  | 'getBuffers'
  | 'getNotices'
  | 'getTasks'
  | 'getTexts'
> {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getNotes: () => deps.getNotes?.(),
    getFiles: () => deps.getFiles?.(),
    getAgent: () => deps.getAgent?.(),
    getBuffers: () => deps.getBuffers?.(),
    getNotices: (pluginId) => deps.getNotices?.(pluginId),
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
  }
}

function matchesSingleTarget(pattern: string, actual: ContextMenuTarget): boolean {
  if (pattern === 'all') return true
  if (pattern === 'root') return actual.path === '' || actual.path === null
  if (pattern === 'folder') return actual.isFolder
  if (pattern === 'file') return !actual.isFolder
  if (pattern === 'note') return !actual.isFolder && actual.isNote
  return false
}

function matchesContextMenuTarget(
  targetPattern: ContextMenuRegistration['target'],
  actual: ContextMenuTarget,
): boolean {
  if (!targetPattern || targetPattern === 'all') return true
  if (typeof targetPattern === 'string') {
    return matchesSingleTarget(targetPattern, actual)
  }
  return targetPattern.some((p) => matchesSingleTarget(p, actual))
}

function disposeAllEntries(registry: HostRegistry, deps: PluginHostDeps) {
  const allPluginIds = new Set<string>([
    ...registry.contexts.keys(),
    ...[...registry.readers.values()].map((r) => r.pluginId),
    ...[...registry.tabKinds.values()].map((t) => t.pluginId),
    ...[...registry.commands.value.values()].map((c) => c.pluginId),
    ...[...registry.contextMenus.value.values()].map((c) => c.pluginId),
    ...[...registry.settingsViews.value.values()].map((v) => v.pluginId),
    ...[...registry.agentProviders.values()].map((p) => p.pluginId),
  ])
  for (const pluginId of allPluginIds) disposePluginEntry(pluginId, registry, deps)
}

export function createPluginHost(deps: PluginHostDeps) {
  const registry: HostRegistry = {
    readers: new Map(),
    tabKinds: new Map(),
    commands: shallowRef(new Map()),
    contextMenus: shallowRef(new Map()),
    settingsViews: shallowRef(new Map()),
    agentProviders: new Map(),
    contexts: new Map(),
    manifests: new Map(),
    vaultListeners: new Set(),
    fileListeners: new Set(),
  }

  const contextHost = createHostContextBridge(
    deps,
    registry,
    createViewRegistry(),
    createDropRegistry(),
  )
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
    executeCommand: async (id: string, note: CommandNote): Promise<boolean> => {
      const entry = resolveCommand(registry.commands.value, id)
      if (!entry) return false
      await runCommand(deps, entry.pluginId, entry.reg, note)
      return true
    },
    getContextMenuItems: (target: ContextMenuTarget): readonly ContextMenuRegistration[] => {
      const items: ContextMenuRegistration[] = []
      for (const { reg } of registry.contextMenus.value.values()) {
        if (matchesContextMenuTarget(reg.target, target)) {
          items.push(reg)
        }
      }
      return items.sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    },
    getContextMenuIcon: (id: string): unknown => registry.contextMenus.value.get(id)?.reg.icon,
    runContextMenu: async (id: string, target: ContextMenuTarget): Promise<boolean> => {
      const entry = registry.contextMenus.value.get(id)
      if (!entry) return false
      await entry.reg.handler(target)
      return true
    },
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
    handleShortcut: (event: KeyboardEvent, getNote: () => Promise<CommandNote>): boolean => {
      for (const { pluginId, reg } of registry.commands.value.values()) {
        if (matchesShortcut(event, reg.shortcut)) {
          getNote()
            .then((note) => {
              if (reg.scope === 'note' && note.path === '') return
              return runCommand(deps, pluginId, reg, note)
            })
            .catch((error: unknown) => reportError(deps, pluginId, error))
          return true
        }
      }
      return false
    },
    getSetting: contextHost.getSetting,
    onSettingChange: contextHost.onSettingChange,
    /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
    notifyVaultChange: (event: VaultChangeEvent): void =>
      notifyListeners(registry.vaultListeners, event, deps),
    disposePlugin,
    disposeAll: () => disposeAllEntries(registry, deps),
    getRegisteredReaders: () => [...registry.readers.values()].map((r) => r.reg),
    getRegisteredTabKinds: () => [...registry.tabKinds.values()].map((t) => t.reg),
    getRegisteredCommands: () => [...registry.commands.value.values()].map((c) => c.reg),
    getRegisteredContextMenus: () => [...registry.contextMenus.value.values()].map((c) => c.reg),
  }
}

export type PluginHost = ReturnType<typeof createPluginHost>
