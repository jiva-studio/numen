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
  readonly loaders?: Record<string, () => Promise<unknown>>
  readonly manifests?: Record<string, PluginManifest>
}

interface ActivePluginEntry {
  readonly pluginId: string
  readonly lifecycle?: PluginLifecycle
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
  readonly catalogManifests: Readonly<Record<string, PluginManifest>>
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

    const manifest = ctx.manifests.get(pluginId) ?? ctx.catalogManifests[pluginId]
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

function checkPluginEnabled(host: PluginHost, id: string): boolean {
  const val = host.getSetting?.(id, 'enabled')
  return typeof val === 'boolean' ? val : true
}

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const moduleLoaders = new Map(Object.entries(deps.loaders ?? {}))
  const settingSubscriptions = new Map<string, Disposable>()
  const manifests = new Map<string, PluginManifest>()
  let hasStarted = false

  const activateContext = {
    host,
    loaders: moduleLoaders,
    catalogManifests: deps.manifests ?? {},
    activePlugins,
    manifests,
    ...defaults,
  }
  const activatePlugin = (id: string, module?: unknown) => activateEntry(activateContext, id, module)

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
        if (checkPluginEnabled(host, id)) void activatePlugin(id)
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
      await startLoader({ moduleLoaders, listen, isEnabled: (id) => checkPluginEnabled(host, id) }, activatePlugin)
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
