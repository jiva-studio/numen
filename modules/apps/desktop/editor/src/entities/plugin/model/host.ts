/**
 * Plugin host managing dynamic registration of readers, tab kinds, and commands.
 */
import { defineComponent, h, shallowRef, type Component } from 'vue'
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  ReaderRegistration,
  TabKindRegistration,
} from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import { createPluginContext } from './context'
import type { PluginContextHost, PluginHostDeps, PluginPaletteCommand } from './types'

export function createPluginPaneWrapper(component: unknown): Component {
  return defineComponent({
    name: 'PluginPaneWrapper',
    props: {
      state: {
        type: null,
        required: true,
      },
    },
    setup(props) {
      return () => {
        if (!component) return null
        const stateObj =
          typeof props.state === 'object' && props.state !== null
            ? (props.state as Record<string, unknown>)
            : {}
        return h(component as Component, {
          ...stateObj,
          state: props.state,
        })
      }
    },
  })
}

function matchesShortcut(event: KeyboardEvent, shortcut?: string): boolean {
  if (!shortcut) return false
  const parts = shortcut.toLowerCase().split('+').map((part) => part.trim())
  const key = parts[parts.length - 1]
  const hasMod =
    parts.includes('ctrl') ||
    parts.includes('control') ||
    parts.includes('mod') ||
    parts.includes('cmd')
  const hasAlt = parts.includes('alt') || parts.includes('option')
  const hasShift = parts.includes('shift')

  if (hasMod !== (event.ctrlKey || event.metaKey)) return false
  if (hasAlt !== event.altKey) return false
  if (hasShift !== event.shiftKey) return false
  return event.key.toLowerCase() === key
}

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
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
  readonly commands: { value: ReadonlyMap<string, RegisteredCommand> }
  readonly agentProviders: Map<string, RegisteredAgentProvider>
  readonly contexts: Map<string, PluginContext>
}

interface PluginItemRegistration<T> {
  readonly pluginId: string
  readonly registration: T
}

function createReaderTabKind(pluginId: string, registration: ReaderRegistration): AnyTabKind {
  const kindId = registration.id.startsWith('reader:') ? registration.id : `reader:${registration.id}`
  return {
    kind: kindId,
    open: (at: string) => ({
      path: at,
      file: at,
      pluginId,
      title: registration.title ?? at.split('/').pop() ?? at,
    }),
    getTitle: (state: unknown) => {
      const s = state as { title?: string; path?: string } | undefined
      return registration.title ?? (s?.title || s?.path?.split('/').pop() || s?.path || '')
    },
    pane: createPluginPaneWrapper(registration.component),
    identity: (at: string) => at,
    getTarget: (state: unknown) => {
      const path = (state as { path?: string } | undefined)?.path
      const isAudio = path ? /\.(mp3|m4a|wav|ogg|aac)$/i.test(path) : false
      return {
        ...(path !== undefined ? { file: path } : {}),
        source: isAudio ? ('recording' as const) : ('other' as const),
      }
    },
    getOpenTab: (state: unknown) => ({ path: (state as { path?: string })?.path ?? '' }),
  }
}

function createCustomTabKind(pluginId: string, registration: TabKindRegistration): AnyTabKind {
  return {
    kind: registration.kind,
    open: (at: string) => ({
      at,
      path: at,
      pluginId,
      title: registration.title ?? registration.kind,
    }),
    getTitle: (state: unknown) => {
      const s = state as { title?: string; at?: string } | undefined
      return registration.title ?? (s?.title || s?.at || registration.kind)
    },
    pane: createPluginPaneWrapper(registration.component),
    identity: (at: string) => at || registration.kind,
  }
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
): CommandRegistration | undefined {
  const direct = commands.get(id)?.reg
  if (direct) return direct
  if (id.startsWith('plugin:')) {
    return commands.get(id.slice('plugin:'.length))?.reg
  }
  return undefined
}

function registerReaderEntry(
  deps: PluginHostDeps,
  readers: Map<string, RegisteredReader>,
  item: PluginItemRegistration<ReaderRegistration>,
): Disposable {
  const { pluginId, registration } = item
  const kindId = registration.id.startsWith('reader:') ? registration.id : `reader:${registration.id}`
  const cleanups: (() => void)[] = [deps.registerKind(createReaderTabKind(pluginId, registration))]
  for (const ext of registration.fileExtensions) {
    cleanups.push(deps.registerExtensionReader(ext, (path) => void deps.openTab(kindId, path)))
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

function registerTabKindEntry(
  deps: PluginHostDeps,
  tabKinds: Map<string, RegisteredTabKind>,
  item: PluginItemRegistration<TabKindRegistration>,
): Disposable {
  const { pluginId, registration } = item
  const unregister = deps.registerKind(createCustomTabKind(pluginId, registration))
  let isDisposed = false
  const cleanup = () => {
    if (isDisposed) return
    isDisposed = true
    unregister()
    tabKinds.delete(registration.kind)
  }
  tabKinds.set(registration.kind, { pluginId, reg: registration, cleanup })
  return { dispose: cleanup }
}

function createHostContextBridge(deps: PluginHostDeps, registry: HostRegistry): PluginContextHost {
  return {
    registerReader: (pluginId, registration) =>
      registerReaderEntry(deps, registry.readers, { pluginId, registration }),
    registerTabKind: (pluginId, registration) =>
      registerTabKindEntry(deps, registry.tabKinds, { pluginId, registration }),
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
      return { dispose: () => { registry.agentProviders.delete(registration.id) } }
    },
    getSetting: <T = unknown>(pluginId: string, key: string) =>
      deps.getSetting ? (deps.getSetting(pluginId, key) as T | undefined) : undefined,
    onSettingChange: <T = unknown>(
      pluginId: string,
      key: string,
      handler: (value: T) => void | Promise<void>,
    ) => {
      if (!deps.onSettingChange) return { dispose: () => {} }
      return { dispose: deps.onSettingChange(pluginId, key, handler as (v: unknown) => void | Promise<void>) }
    },
    openTab: (kind: string, at?: string) => deps.openTab(kind, at),
  }
}

function extractPaletteCommands(
  commands: ReadonlyMap<string, { reg: CommandRegistration }>,
): PluginPaletteCommand[] {
  const list: PluginPaletteCommand[] = []
  for (const { reg } of commands.values()) {
    if (reg.canShowInPalette === false) continue
    list.push({
      id: `plugin:${reg.id}`,
      title: reg.title,
      ...(reg.shortcut ? { shortcut: reg.shortcut } : {}),
      isOffered: () => true,
    })
  }
  return list
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
  }

  const contextHost = createHostContextBridge(deps, registry)
  const disposePlugin = (pluginId: string) => disposePluginEntry(pluginId, registry, deps)

  return {
    contextHost,
    createContext: (pluginId: string): PluginContext => {
      let ctx = registry.contexts.get(pluginId)
      if (!ctx) {
        ctx = createPluginContext(pluginId, contextHost)
        registry.contexts.set(pluginId, ctx)
      }
      return ctx
    },
    getPaletteCommands: () => extractPaletteCommands(registry.commands.value),
    hasCommand: (id: string): boolean => resolveCommand(registry.commands.value, id) !== undefined,
    executeCommand: async (id: string): Promise<boolean> => {
      const reg = resolveCommand(registry.commands.value, id)
      if (!reg) return false
      await reg.handler()
      return true
    },
    handleShortcut: (event: KeyboardEvent): boolean => {
      for (const { reg } of registry.commands.value.values()) {
        if (matchesShortcut(event, reg.shortcut)) {
          void reg.handler()
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
