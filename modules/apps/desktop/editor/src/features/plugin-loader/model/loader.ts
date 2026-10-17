/**
 * Dynamic frontend plugin loader for desktop editor.
 */
import { shallowRef } from 'vue'
import type { Disposable, PluginLifecycle, PluginManifest } from '@numen/plugin-sdk'
import { registerPluginStyle, unregisterPluginStyle, type PluginHost } from '@/entities/plugin'

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

import * as flashcardsStudio from '@plugins/flashcards-studio/frontend'

const IN_TREE_MANIFESTS: Record<string, PluginManifest> = {
  'md.numen.pdf': {
    id: 'md.numen.pdf',
    name: 'PDF Reader',
    version: '1.0.0',
    permissions: ['vault:read'],
  },
  'md.numen.epub': {
    id: 'md.numen.epub',
    name: 'EPUB Book Reader',
    version: '1.0.0',
    permissions: ['vault:read'],
  },
  'md.numen.editor-markdown': {
    id: 'md.numen.editor-markdown',
    name: 'Markdown Editor',
    version: '1.0.0',
    permissions: ['vault:read', 'vault:write'],
  },
  'md.numen.view-file-tree': {
    id: 'md.numen.view-file-tree',
    name: 'File Tree Explorer',
    version: '1.0.0',
    permissions: ['vault:read'],
  },
  'md.numen.view-plex-graph': {
    id: 'md.numen.view-plex-graph',
    name: 'Plex Graph',
    version: '1.0.0',
    permissions: ['vault:read'],
  },
  'md.numen.media-player': {
    id: 'md.numen.media-player',
    name: 'Media Player',
    version: '1.0.0',
    permissions: ['vault:read'],
  },
  'md.numen.downloader': {
    id: 'md.numen.downloader',
    name: 'Downloader',
    version: '1.0.0',
    permissions: ['vault:write'],
  },
  'md.numen.flashcards-studio': {
    id: 'md.numen.flashcards-studio',
    name: 'Flashcards Studio',
    version: '1.0.0',
    permissions: ['vault:read', 'vault:write'],
  },
}

const IN_TREE_LOADERS: Record<string, () => Promise<unknown>> = {
  'md.numen.pdf': () => import('@plugins/reader-pdf/frontend'),
  'md.numen.epub': () => import('@plugins/reader-epub/frontend'),
  'md.numen.editor-markdown': () => import('@plugins/editor-markdown/frontend'),
  'md.numen.view-file-tree': () => import('@plugins/view-file-tree/frontend'),
  'md.numen.view-plex-graph': () => import('@plugins/view-plex-graph/frontend'),
  'md.numen.media-player': () => import('@plugins/media-player/frontend'),
  'md.numen.downloader': () => import('@plugins/downloader/frontend'),
  'md.numen.flashcards-studio': async () => flashcardsStudio,
}

function injectPluginStyles(baseUrl: string, pluginId: string, stylesPath: string): void {
  const normalizedPath = stylesPath.startsWith('/') ? stylesPath.slice(1) : stylesPath
  const fullUrl = `${baseUrl}/${pluginId}/${normalizedPath}`
  registerPluginStyle(pluginId, fullUrl)
}

function removePluginStyles(pluginId: string): void {
  unregisterPluginStyle(pluginId)
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
  readonly manifests: Map<string, PluginManifest>
}

async function loadDynamicModule(pluginId: string, options: LoadModuleOptions): Promise<unknown> {
  const { baseUrl, fetchFn, importFn, manifests } = options
  let entrypoint = 'dist/index.js'
  try {
    const res = await fetchFn(`${baseUrl}/${pluginId}/manifest.json`)
    if (res.ok) {
      const manifest = (await res.json()) as PluginManifest
      manifests.set(pluginId, manifest)
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

    const manifest = ctx.manifests.get(pluginId) ?? IN_TREE_MANIFESTS[pluginId]
    const { activate, deactivate } = getLifecycleHooks(resolved)
    if (activate) {
      await activate(host.createContext(pluginId, manifest))
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

interface DeactivateDeps {
  readonly host: PluginHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly manifests: Map<string, PluginManifest>
}

async function deactivateEntry(deps: DeactivateDeps, pluginId: string): Promise<boolean> {
  const entry = deps.activePlugins.value.get(pluginId)
  if (!entry) {
    removePluginStyles(pluginId)
    deps.manifests.delete(pluginId)
    deps.host.disposePlugin(pluginId)
    return false
  }
  try {
    if (entry.lifecycle?.deactivate) {
      await entry.lifecycle.deactivate()
    }
  } finally {
    removePluginStyles(pluginId)
    deps.manifests.delete(pluginId)
    deps.host.disposePlugin(pluginId)
    const next = new Map(deps.activePlugins.value)
    next.delete(pluginId)
    deps.activePlugins.value = next
  }
  return true
}

interface ListenerDeps {
  readonly host: PluginHost
  readonly subscriptions: Map<string, Disposable>
  readonly activate: (id: string) => Promise<boolean>
  readonly deactivate: (id: string) => Promise<boolean>
}

function listenToPluginState(deps: ListenerDeps, pluginId: string) {
  if (deps.subscriptions.has(pluginId) || !deps.host.onSettingChange) return
  const sub = deps.host.onSettingChange<boolean>(pluginId, 'enabled', async (enabled) => {
    if (enabled === false) await deps.deactivate(pluginId)
    else await deps.activate(pluginId)
  })
  deps.subscriptions.set(pluginId, sub)
}

interface StartLoaderDeps {
  readonly moduleLoaders: Map<string, () => Promise<unknown>>
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
}

async function startLoader(deps: StartLoaderDeps, activate: (id: string) => Promise<boolean>) {
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

function resolveDefaultDeps(deps: PluginLoaderDeps) {
  return {
    baseUrl: deps.baseUrl ?? '/api/plugins',
    fetchFn:
      deps.fetch ??
      ((input: RequestInfo | URL, init?: RequestInit) => globalThis.fetch(input, init)),
    importFn: deps.dynamicImport ?? ((url: string) => import(/* @vite-ignore */ url)),
  }
}

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<unknown>>(Object.entries(IN_TREE_LOADERS))
  const settingSubscriptions = new Map<string, Disposable>()
  const manifests = new Map<string, PluginManifest>()
  let hasStarted = false

  const isPluginEnabled = (id: string): boolean => {
    const val = host.getSetting?.(id, 'enabled')
    return typeof val === 'boolean' ? val : true
  }

  const activatePlugin = (id: string, module?: unknown) =>
    activateEntry(
      { host, loaders: moduleLoaders, activePlugins, manifests, ...defaults },
      id,
      module,
    )

  const deactivatePlugin = (id: string) => deactivateEntry({ host, activePlugins, manifests }, id)

  const listen = (id: string) =>
    listenToPluginState(
      {
        host,
        subscriptions: settingSubscriptions,
        activate: activatePlugin,
        deactivate: deactivatePlugin,
      },
      id,
    )

  return {
    registerPluginModule: (id: string, loader: () => Promise<unknown>) => {
      moduleLoaders.set(id, loader)
      if (hasStarted) {
        listen(id)
        if (isPluginEnabled(id)) void activatePlugin(id)
      }
    },
    registerPluginManifest: (id: string, manifest: PluginManifest) => {
      manifests.set(id, manifest)
      host.registerManifest?.(id, manifest)
    },
    activatePlugin,
    deactivatePlugin,
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    start: async (): Promise<void> => {
      hasStarted = true
      await startLoader({ moduleLoaders, listen, isEnabled: isPluginEnabled }, activatePlugin)
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
