/**
 * Dynamic frontend plugin loader for desktop editor.
 */
import { shallowRef } from 'vue'
import type { Disposable, PluginLifecycle } from '@numen/plugin-sdk'
import type { PluginHost } from '@/entities/plugin'

export interface PluginLoaderDeps {
  readonly host: PluginHost
}

interface ActivePluginEntry {
  readonly pluginId: string
  readonly lifecycle?: PluginLifecycle
}

const loadInTree = (name: string) => import(`@plugins/${name}/frontend/index.ts`)

const IN_TREE_LOADERS: Record<string, () => Promise<unknown>> = {
  'md.numen.pdf': () => import('@plugins/reader-pdf/frontend'),
  'md.numen.epub': () => import('@plugins/reader-epub/frontend'),
  'md.numen.editor-markdown': () => import('@plugins/editor-markdown/frontend'),
  'md.numen.view-file-tree': () => import('@plugins/view-file-tree/frontend'),
  'md.numen.view-plex-graph': () => import('@plugins/view-plex-graph/frontend'),
  'md.numen.media-player': () => import('@plugins/media-player/frontend'),
}

function extractLifecycleFunction(
  target: Record<string, unknown>,
  fallback: Record<string, unknown>,
  key: string,
): ((...args: unknown[]) => unknown) | undefined {
  if (typeof target[key] === 'function') {
    return target[key] as (...args: unknown[]) => unknown
  }
  if (typeof fallback[key] === 'function') {
    return fallback[key] as (...args: unknown[]) => unknown
  }
  return undefined
}

function getLifecycleHooks(resolved: unknown): {
  activate?: PluginLifecycle['activate'] | undefined
  deactivate?: PluginLifecycle['deactivate'] | undefined
} {
  if (typeof resolved !== 'object' || resolved === null) return {}
  const mod = resolved as Record<string, unknown>
  const target = (
    typeof mod.default === 'object' && mod.default !== null ? mod.default : mod
  ) as Record<string, unknown>
  const activate = extractLifecycleFunction(target, mod, 'activate') as
    PluginLifecycle['activate'] | undefined
  const deactivate = extractLifecycleFunction(target, mod, 'deactivate') as
    PluginLifecycle['deactivate'] | undefined
  return { activate, deactivate }
}

async function resolveModule(
  loaders: Map<string, () => Promise<unknown>>,
  pluginId: string,
  candidate?: unknown,
): Promise<unknown> {
  if (candidate !== undefined) {
    if (typeof candidate === 'function' && !('activate' in candidate)) {
      return (candidate as () => Promise<unknown>)()
    }
    return candidate
  }
  const loader = loaders.get(pluginId)
  return loader ? loader() : undefined
}

interface ActivateContext {
  readonly host: PluginHost
  readonly loaders: Map<string, () => Promise<unknown>>
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
}

async function activateEntry(
  ctx: ActivateContext,
  pluginId: string,
  pluginModule?: unknown,
): Promise<boolean> {
  const { host, loaders, activePlugins } = ctx
  if (activePlugins.value.has(pluginId)) return true
  const resolved = await resolveModule(loaders, pluginId, pluginModule)
  if (!resolved) return false

  const { activate, deactivate } = getLifecycleHooks(resolved)
  if (activate) {
    await activate(host.createContext(pluginId))
  }

  const lifecycle: PluginLifecycle = {
    activate: activate ?? (() => {}),
    ...(deactivate ? { deactivate } : {}),
  }

  const next = new Map(activePlugins.value)
  next.set(pluginId, { pluginId, lifecycle })
  activePlugins.value = next
  return true
}

async function deactivateEntry(
  host: PluginHost,
  activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> },
  pluginId: string,
): Promise<boolean> {
  const entry = activePlugins.value.get(pluginId)
  if (!entry) {
    host.disposePlugin(pluginId)
    return false
  }
  try {
    if (entry.lifecycle?.deactivate) {
      await entry.lifecycle.deactivate()
    }
  } finally {
    host.disposePlugin(pluginId)
    const next = new Map(activePlugins.value)
    next.delete(pluginId)
    activePlugins.value = next
  }
  return true
}

interface StartLoaderDeps {
  readonly moduleLoaders: Map<string, () => Promise<unknown>>
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
}

async function startLoader(
  deps: StartLoaderDeps,
  activate: (id: string) => Promise<boolean>,
) {
  for (const pluginId of deps.moduleLoaders.keys()) {
    deps.listen(pluginId)
    if (deps.isEnabled(pluginId)) {
      try {
        await activate(pluginId)
      } catch {
        // Plugin activation failure should not halt startup
      }
    }
  }
}

interface CloseLoaderDeps {
  readonly host: PluginHost
  readonly subscriptions: Map<string, Disposable>
}

async function closeLoader(
  deps: CloseLoaderDeps,
  activeIds: readonly string[],
  deactivate: (id: string) => Promise<boolean>,
) {
  for (const sub of deps.subscriptions.values()) sub.dispose()
  deps.subscriptions.clear()
  for (const pluginId of activeIds) {
    await deactivate(pluginId)
  }
  deps.host.disposeAll()
}

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<unknown>>(Object.entries(IN_TREE_LOADERS))
  const settingSubscriptions = new Map<string, Disposable>()
  let hasStarted = false

  const isPluginEnabled = (pluginId: string): boolean => {
    const val = host.getSetting?.(pluginId, 'enabled')
    if (typeof val === 'boolean') return val
    return true
  }

  const activatePlugin = (pluginId: string, pluginModule?: unknown) =>
    activateEntry({ host, loaders: moduleLoaders, activePlugins }, pluginId, pluginModule)

  const deactivatePlugin = (pluginId: string) =>
    deactivateEntry(host, activePlugins, pluginId)

  function listenToPluginState(pluginId: string) {
    if (settingSubscriptions.has(pluginId) || !host.onSettingChange) return
    const sub = host.onSettingChange<boolean>(pluginId, 'enabled', async (enabled) => {
      if (enabled === false) await deactivatePlugin(pluginId)
      else await activatePlugin(pluginId)
    })
    settingSubscriptions.set(pluginId, sub)
  }

  return {
    registerPluginModule: (pluginId: string, loader: () => Promise<unknown>) => {
      moduleLoaders.set(pluginId, loader)
      if (hasStarted) {
        listenToPluginState(pluginId)
        if (isPluginEnabled(pluginId)) void activatePlugin(pluginId)
      }
    },
    activatePlugin,
    deactivatePlugin,
    isPluginActive: (pluginId: string) => activePlugins.value.has(pluginId),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    start: async (): Promise<void> => {
      hasStarted = true
      await startLoader(
        { moduleLoaders, listen: listenToPluginState, isEnabled: isPluginEnabled },
        activatePlugin,
      )
    },
    close: async (): Promise<void> => {
      hasStarted = false
      await closeLoader(
        { host, subscriptions: settingSubscriptions },
        [...activePlugins.value.keys()],
        deactivatePlugin,
      )
    },
  }
}

export type PluginLoader = ReturnType<typeof createPluginLoader>

