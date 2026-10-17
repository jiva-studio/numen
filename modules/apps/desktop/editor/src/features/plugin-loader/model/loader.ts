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
  readonly onError?: (pluginId: string, error: unknown) => void
}

export const DEFAULT_PLUGIN_ENABLED = true

/** A plugin is enabled unless its setting says false. */
export function isPluginEnabledValue(value: unknown): boolean {
  return typeof value === 'boolean' ? value : DEFAULT_PLUGIN_ENABLED
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
  readonly getImportToken: () => number
}

async function loadDynamicModule(
  pluginId: string,
  options: LoadModuleOptions,
): Promise<unknown | undefined> {
  const { baseUrl, fetchFn, importFn, manifests } = options
  const res = await fetchFn(`${baseUrl}/${pluginId}/manifest.json`)
  if (!res.ok) {
    throw new Error(`manifest.json of plugin "${pluginId}" answered ${res.status}`)
  }
  const manifest = (await res.json()) as PluginManifest
  manifests.set(pluginId, manifest)
  if (manifest.frontend === undefined) return undefined
  if (manifest.frontend.styles) {
    injectPluginStyles(baseUrl, pluginId, manifest.frontend.styles)
  }
  const entrypoint = manifest.frontend.entrypoint ?? 'dist/index.js'
  const normalizedEntrypoint = entrypoint.startsWith('/') ? entrypoint.slice(1) : entrypoint
  const token = options.getImportToken()
  return importFn(`${baseUrl}/${pluginId}/${normalizedEntrypoint}?activation=${token}`)
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
  readonly failures: { value: ReadonlyMap<string, string> }
  readonly reportError: (pluginId: string, error: unknown) => void
}

function getFailureMessage(error: unknown): string {
  return error instanceof Error ? error.message : `the plugin threw a ${typeof error}, not an Error`
}

function setFailure(ctx: ActivateContext, pluginId: string, error: string | undefined): void {
  const next = new Map(ctx.failures.value)
  if (error === undefined) next.delete(pluginId)
  else next.set(pluginId, error)
  ctx.failures.value = next
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
    setFailure(ctx, pluginId, undefined)
    return true
  } catch (error) {
    removePluginStyles(pluginId)
    host.disposePlugin(pluginId)
    setFailure(ctx, pluginId, getFailureMessage(error))
    ctx.reportError(pluginId, error)
    return false
  }
}

interface DeactivateDeps {
  readonly host: PluginHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly manifests: Map<string, PluginManifest>
  readonly failures: { value: ReadonlyMap<string, string> }
  readonly reportError: (pluginId: string, error: unknown) => void
}

async function deactivateEntry(deps: DeactivateDeps, pluginId: string): Promise<boolean> {
  const entry = deps.activePlugins.value.get(pluginId)
  const failures = new Map(deps.failures.value)
  failures.delete(pluginId)
  deps.failures.value = failures
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
  } catch (error) {
    deps.reportError(pluginId, error)
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
    if (isPluginEnabledValue(enabled)) await deps.activate(pluginId)
    else await deps.deactivate(pluginId)
  })
  deps.subscriptions.set(pluginId, sub)
}

interface StartLoaderDeps {
  readonly pluginIds: readonly string[]
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
}

async function startLoader(deps: StartLoaderDeps, activate: (id: string) => Promise<boolean>) {
  for (const pluginId of deps.pluginIds) {
    deps.listen(pluginId)
    if (deps.isEnabled(pluginId)) await activate(pluginId)
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

function resolveDefaultDeps(deps: PluginLoaderDeps) {
  return {
    baseUrl: deps.baseUrl ?? '/api/plugins',
    fetchFn:
      deps.fetch ??
      ((input: RequestInfo | URL, init?: RequestInit) => globalThis.fetch(input, init)),
    importFn: deps.dynamicImport ?? ((url: string) => import(/* @vite-ignore */ url)),
    reportError:
      deps.onError ??
      ((pluginId: string, error: unknown) => console.error(`plugin "${pluginId}" failed`, error)),
  }
}

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const failures = shallowRef<ReadonlyMap<string, string>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<unknown>>(
    Object.entries(deps.loaders ?? {}),
  )
  const trackedIds = new Set<string>()
  const settingSubscriptions = new Map<string, Disposable>()
  const manifests = new Map<string, PluginManifest>()
  const queues = new Map<string, Promise<unknown>>()
  let importToken = 0
  let hasStarted = false

  const isPluginEnabled = (id: string): boolean =>
    isPluginEnabledValue(host.getSetting?.(id, 'enabled'))

  const enqueue = <T>(id: string, task: () => Promise<T>): Promise<T> => {
    const run = (queues.get(id) ?? Promise.resolve()).then(task)
    queues.set(
      id,
      run.catch(() => undefined),
    )
    return run
  }

  const activatePlugin = (id: string, module?: unknown) =>
    enqueue(id, () =>
      activateEntry(
        {
          host,
          loaders: moduleLoaders,
          catalogManifests: deps.manifests ?? {},
          activePlugins,
          failures,
          manifests,
          getImportToken: () => ++importToken,
          ...defaults,
        },
        id,
        module,
      ),
    )

  const deactivatePlugin = (id: string) =>
    enqueue(id, () =>
      deactivateEntry(
        { host, activePlugins, manifests, failures, reportError: defaults.reportError },
        id,
      ),
    )

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

  const getPluginIds = (): string[] => [...new Set([...moduleLoaders.keys(), ...trackedIds])]

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
    trackPlugins: async (ids: readonly string[]): Promise<void> => {
      for (const id of ids) {
        if (trackedIds.has(id)) continue
        trackedIds.add(id)
        if (!hasStarted) continue
        listen(id)
        if (isPluginEnabled(id)) await activatePlugin(id)
      }
    },
    activatePlugin,
    deactivatePlugin,
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    getPluginFailure: (id: string): string | undefined => failures.value.get(id),
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
    start: async (): Promise<void> => {
      hasStarted = true
      await startLoader(
        { pluginIds: getPluginIds(), listen, isEnabled: isPluginEnabled },
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
