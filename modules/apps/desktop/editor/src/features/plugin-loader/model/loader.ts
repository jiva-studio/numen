/**
 * Dynamic frontend plugin loader for desktop editor.
 */
import { shallowRef } from 'vue'
import type { PluginLifecycle } from '@numen/plugin-sdk'
import type { PluginHost } from '@/entities/plugin'

export interface PluginLoaderDeps {
  readonly host: PluginHost
}

interface ActivePluginEntry {
  readonly pluginId: string
  readonly lifecycle?: PluginLifecycle
}

const IN_TREE_LOADERS: Record<string, () => Promise<unknown>> = {
  'md.numen.pdf': () => import('@plugins/reader-pdf/frontend'),
  'md.numen.epub': () => import('@plugins/reader-epub/frontend'),
  'md.numen.editor-markdown': () => import('@plugins/editor-markdown/frontend'),
  'editor-markdown': () => import('@plugins/editor-markdown/frontend'),
  'md.numen.view-file-tree': () => import('@plugins/view-file-tree/frontend'),
  'view-file-tree': () => import('@plugins/view-file-tree/frontend'),
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

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<unknown>>(Object.entries(IN_TREE_LOADERS))

  async function activatePlugin(pluginId: string, pluginModule?: unknown): Promise<boolean> {
    if (activePlugins.value.has(pluginId)) return true
    const resolved = await resolveModule(moduleLoaders, pluginId, pluginModule)
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

  async function deactivatePlugin(pluginId: string): Promise<boolean> {
    const entry = activePlugins.value.get(pluginId)
    if (!entry) return false
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

  return {
    registerPluginModule: (pluginId: string, loader: () => Promise<unknown>) => {
      moduleLoaders.set(pluginId, loader)
    },
    activatePlugin,
    deactivatePlugin,
    isPluginActive: (pluginId: string) => activePlugins.value.has(pluginId),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    start: async (): Promise<void> => {
      for (const pluginId of moduleLoaders.keys()) {
        try {
          await activatePlugin(pluginId)
        } catch {
          // Plugin activation failure should not halt startup
        }
      }
    },
    close: async (): Promise<void> => {
      for (const pluginId of [...activePlugins.value.keys()]) {
        await deactivatePlugin(pluginId)
      }
      host.disposeAll()
    },
  }
}

export type PluginLoader = ReturnType<typeof createPluginLoader>
