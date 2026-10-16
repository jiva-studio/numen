/**
 * Dynamic frontend plugin loader for desktop editor.
 */
import { shallowRef } from 'vue'
import type { Disposable, PluginLifecycle, PluginManifest } from '@numen/plugin-sdk'
import type { PluginHost } from '@/entities/plugin'

export interface PluginLoaderDeps {
  readonly host: PluginHost
  readonly baseUrl?: string
  readonly fetch?: typeof fetch
  readonly dynamicImport?: (url: string) => Promise<unknown>
}

interface ActivePluginEntry {
  readonly pluginId: string
  readonly lifecycle?: PluginLifecycle
}

const IN_TREE_LOADERS: Record<string, () => Promise<unknown>> = {
  'md.numen.pdf': () => import('@plugins/reader-pdf/frontend'),
  'md.numen.epub': () => import('@plugins/reader-epub/frontend'),
  'md.numen.editor-markdown': () => import('@plugins/editor-markdown/frontend'),
  'md.numen.view-file-tree': () => import('@plugins/view-file-tree/frontend'),
  'md.numen.view-plex-graph': () => import('@plugins/view-plex-graph/frontend'),
  'md.numen.media-player': () => import('@plugins/media-player/frontend'),
  'md.numen.downloader': () => import('@plugins/downloader/frontend'),
}

function injectPluginStyles(baseUrl: string, pluginId: string, stylesPath: string): void {
  if (typeof document === 'undefined') return
  const linkId = `plugin-styles-${pluginId}`
  if (document.getElementById(linkId) || document.querySelector(`link[data-plugin-id="${pluginId}"]`)) {
    return
  }
  const link = document.createElement('link')
  link.id = linkId
  link.rel = 'stylesheet'
  const normalizedPath = stylesPath.startsWith('/') ? stylesPath.slice(1) : stylesPath
  link.href = `${baseUrl}/${pluginId}/${normalizedPath}`
  link.setAttribute('data-plugin-id', pluginId)
  document.head.appendChild(link)
}

function removePluginStyles(pluginId: string): void {
  if (typeof document === 'undefined') return
  const links = document.querySelectorAll(`link[data-plugin-id="${pluginId}"], link#plugin-styles-${pluginId}`)
  for (const link of links) {
    link.remove()
  }
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

interface LoadModuleOptions {
  readonly baseUrl: string
  readonly fetchFn: typeof fetch
  readonly importFn: (url: string) => Promise<unknown>
}

async function loadDynamicModule(
  pluginId: string,
  options: LoadModuleOptions,
): Promise<unknown> {
  const { baseUrl, fetchFn, importFn } = options
  let entrypoint = 'dist/index.js'
  try {
    const res = await fetchFn(`${baseUrl}/${pluginId}/manifest.json`)
    if (res.ok) {
      const manifest = (await res.json()) as PluginManifest
      if (manifest?.frontend?.styles) {
        injectPluginStyles(baseUrl, pluginId, manifest.frontend.styles)
      }
      if (manifest?.frontend?.entrypoint) {
        entrypoint = manifest.frontend.entrypoint
      }
    }
  } catch {
    // Proceed with default entrypoint
  }

  const normalizedEntrypoint = entrypoint.startsWith('/') ? entrypoint.slice(1) : entrypoint
  const moduleUrl = `${baseUrl}/${pluginId}/${normalizedEntrypoint}`
  return importFn(moduleUrl)
}

interface ResolveModuleContext extends LoadModuleOptions {
  readonly loaders: Map<string, () => Promise<unknown>>
}

async function resolveModule(
  pluginId: string,
  candidate: unknown,
  ctx: ResolveModuleContext,
): Promise<unknown> {
  if (candidate !== undefined) {
    if (typeof candidate === 'function' && !('activate' in candidate)) {
      return (candidate as () => Promise<unknown>)()
    }
    return candidate
  }
  const loader = ctx.loaders.get(pluginId)
  if (loader) return loader()
  return loadDynamicModule(pluginId, ctx)
}

interface ActivateContext extends ResolveModuleContext {
  readonly host: PluginHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
}

async function activateEntry(
  ctx: ActivateContext,
  pluginId: string,
  pluginModule?: unknown,
): Promise<boolean> {
  const { host, activePlugins } = ctx
  if (activePlugins.value.has(pluginId)) return true
  try {
    const resolved = await resolveModule(pluginId, pluginModule, ctx)
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
  } catch {
    removePluginStyles(pluginId)
    host.disposePlugin(pluginId)
    return false
  }
}

async function deactivateEntry(
  host: PluginHost,
  activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> },
  pluginId: string,
): Promise<boolean> {
  const entry = activePlugins.value.get(pluginId)
  if (!entry) {
    removePluginStyles(pluginId)
    host.disposePlugin(pluginId)
    return false
  }
  try {
    if (entry.lifecycle?.deactivate) {
      await entry.lifecycle.deactivate()
    }
  } finally {
    removePluginStyles(pluginId)
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
  const activations: Promise<void>[] = []
  for (const pluginId of deps.moduleLoaders.keys()) {
    deps.listen(pluginId)
    if (deps.isEnabled(pluginId)) {
      activations.push(
        activate(pluginId)
          .then(() => {})
          .catch(() => {}),
      )
    }
  }
  await Promise.all(activations)
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
  const baseUrl = deps.baseUrl ?? '/api/plugins'
  const fetchFn = deps.fetch ?? ((input: RequestInfo | URL, init?: RequestInit) => globalThis.fetch(input, init))
  const importFn = deps.dynamicImport ?? ((url: string) => import(/* @vite-ignore */ url))
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<unknown>>()
  const settingSubscriptions = new Map<string, Disposable>()
  let hasStarted = false

  const isPluginEnabled = (pluginId: string): boolean => {
    const val = host.getSetting?.(pluginId, 'enabled')
    if (typeof val === 'boolean') return val
    return true
  }

  const activatePlugin = (pluginId: string, pluginModule?: unknown) =>
    activateEntry(
      { host, loaders: moduleLoaders, activePlugins, baseUrl, fetchFn, importFn },
      pluginId,
      pluginModule,
    )

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

