/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting.
 */
import { shallowRef, type ShallowRef } from 'vue'
import type { Disposable, PluginLifecycle } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { LoaderHost } from './host'
import {
  getLifecycleHooks,
  removePluginStyles,
  resolveModule,
  type ResolveModuleContext,
} from './module'

export interface PluginLoaderDeps {
  readonly host: LoaderHost
  readonly baseUrl?: string
  readonly fetch?: typeof fetch
  readonly dynamicImport?: (url: string) => Promise<unknown>
  readonly onError?: (pluginId: string, error: unknown) => void
}

export const DEFAULT_PLUGIN_ENABLED = true

/** A plugin is enabled unless its setting is false. */
export function isPluginEnabledValue(value: unknown): boolean {
  return typeof value === 'boolean' ? value : DEFAULT_PLUGIN_ENABLED
}

interface ActivePluginEntry {
  readonly pluginId: string
  readonly lifecycle?: PluginLifecycle
}

interface ActivateContext extends ResolveModuleContext {
  readonly host: LoaderHost
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

    const manifest = ctx.manifests.get(pluginId)
    const { activate, deactivate } = getLifecycleHooks(resolved)
    if (activate) {
      await activate(host.createContext(pluginId, manifest))
    }

    const lifecycle: PluginLifecycle = {
      activate: activate ?? (() => undefined),
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
  readonly host: LoaderHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly manifests: Map<string, Manifest>
  readonly reportError: (pluginId: string, error: unknown) => void
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
  readonly host: LoaderHost
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

/** How long the start waits for one plugin to be read before it goes on to the next. */
export const PLUGIN_WAIT_MS = 3_000

interface StartLoaderDeps {
  readonly pluginIds: readonly string[]
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
  /** Reads the module of a plugin from the window. */
  readonly load: (id: string) => Promise<unknown>
}

/** What reading a plugin gave: its module, or the failure, which the activation reads again and reports. */
type Loaded = { readonly isRead: true; readonly module: unknown } | { readonly isRead: false }

/** A plugin that has no frontend has no module, and nothing to activate. */
const startPlugin = (
  activate: (id: string, module?: unknown) => Promise<boolean>,
  id: string,
  one: Loaded,
): Promise<boolean> =>
  one.isRead && one.module === undefined
    ? Promise.resolve(false)
    : activate(id, one.isRead ? one.module : undefined)

/** What the read of a plugin gave within PLUGIN_WAIT_MS, or 'late' where it gave nothing yet. */
async function readWithin(load: Promise<Loaded>): Promise<Loaded | 'late'> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<'late'>((resolve) => {
    timer = setTimeout(() => resolve('late'), PLUGIN_WAIT_MS)
  })
  try {
    return await Promise.race([load, late])
  } finally {
    clearTimeout(timer)
  }
}

/**
 * Reads every enabled plugin at once and activates them in the order of the list, so each plugin
 * registers what it provides in the same order on every start. A plugin that is not read within
 * PLUGIN_WAIT_MS is passed over and activated when it is read; it holds back no other plugin. The
 * promise settles when every plugin has been activated or has failed.
 */
async function startLoader(
  deps: StartLoaderDeps,
  activate: (id: string, module?: unknown) => Promise<boolean>,
) {
  for (const pluginId of deps.pluginIds) deps.listen(pluginId)
  const enabled = deps.pluginIds.filter((id) => deps.isEnabled(id))
  const loading = new Map<string, Promise<Loaded>>(
    enabled.map((id) => [
      id,
      deps.load(id).then(
        (module): Loaded => ({ isRead: true, module }),
        (): Loaded => ({ isRead: false }),
      ),
    ]),
  )
  const running: Promise<unknown>[] = []
  for (const pluginId of enabled) {
    const load = loading.get(pluginId) as Promise<Loaded>
    const outcome = await readWithin(load)
    if (outcome === 'late') {
      running.push(load.then((one) => startPlugin(activate, pluginId, one)))
      continue
    }
    const run = startPlugin(activate, pluginId, outcome)
    running.push(run)
    await run
  }
  await Promise.all(running)
}

interface CloseLoaderDeps {
  readonly host: LoaderHost
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
    baseUrl: deps.baseUrl ?? '/plugin',
    fetchFn:
      deps.fetch ??
      ((input: RequestInfo | URL, init?: RequestInit) => globalThis.fetch(input, init)),
    importFn: deps.dynamicImport ?? ((url: string) => import(/* @vite-ignore */ url)),
    // Without a reporter the failure is already recorded in the failures map.
    reportError: deps.onError ?? (() => undefined),
  }
}

function createActivation(deps: PluginLoaderDeps, state: LoaderState) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const { activePlugins, failures, manifests, moduleLoaders } = state
  const queues = new Map<string, Promise<unknown>>()

  const enqueue = <T>(id: string, task: () => Promise<T>): Promise<T> => {
    const run = (queues.get(id) ?? Promise.resolve()).then(task)
    queues.set(
      id,
      run.catch(() => undefined),
    )
    return run
  }

  const activateContext: ActivateContext = {
    host,
    loaders: moduleLoaders,
    activePlugins,
    failures,
    manifests,
    ...defaults,
  }
  const deactivateContext: DeactivateDeps = {
    host,
    activePlugins,
    manifests,
    reportError: defaults.reportError,
  }

  return {
    loadModule: (id: string) => resolveModule(id, undefined, activateContext),
    activatePlugin: (id: string, module?: unknown) =>
      enqueue(id, () => activateEntry(activateContext, id, module)),
    deactivatePlugin: (id: string) => enqueue(id, () => deactivateEntry(deactivateContext, id)),
  }
}

interface LoaderState {
  readonly activePlugins: ShallowRef<ReadonlyMap<string, ActivePluginEntry>>
  readonly failures: ShallowRef<ReadonlyMap<string, string>>
  readonly manifests: Map<string, Manifest>
  readonly moduleLoaders: Map<string, () => Promise<unknown>>
}

interface Activation {
  readonly loadModule: (id: string) => Promise<unknown>
  readonly activatePlugin: (id: string, module?: unknown) => Promise<boolean>
  readonly deactivatePlugin: (id: string) => Promise<boolean>
}

function createRuntime(deps: PluginLoaderDeps, state: LoaderState, activation: Activation) {
  const { host } = deps
  const { activatePlugin, deactivatePlugin, loadModule } = activation
  const { activePlugins, moduleLoaders } = state
  const trackedIds = new Set<string>()
  const settingSubscriptions = new Map<string, Disposable>()
  let hasStarted = false

  const isEnabled = (id: string): boolean => isPluginEnabledValue(host.getSetting?.(id, 'enabled'))
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

  const activateInBackground = (id: string): void => {
    activatePlugin(id).catch((error: unknown) => {
      deps.onError?.(id, error)
    })
  }

  return {
    registerPluginModule: (id: string, loader: () => Promise<unknown>) => {
      moduleLoaders.set(id, loader)
      if (hasStarted) {
        listen(id)
        if (isEnabled(id)) activateInBackground(id)
      }
    },
    trackPlugins: async (ids: readonly string[]): Promise<void> => {
      for (const id of ids) {
        if (trackedIds.has(id)) continue
        trackedIds.add(id)
        if (!hasStarted) continue
        listen(id)
        if (isEnabled(id)) await activatePlugin(id)
      }
    },
    start: async (): Promise<void> => {
      hasStarted = true
      const pluginIds = [...new Set([...moduleLoaders.keys(), ...trackedIds])]
      await startLoader({ pluginIds, listen, isEnabled, load: loadModule }, activatePlugin)
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

export function createPluginLoader(deps: PluginLoaderDeps) {
  const { host } = deps
  const state: LoaderState = {
    activePlugins: shallowRef(new Map()),
    failures: shallowRef(new Map()),
    manifests: new Map(),
    moduleLoaders: new Map(),
  }
  const { activePlugins, failures, manifests } = state
  const activation = createActivation(deps, state)

  return {
    ...createRuntime(deps, state, activation),
    ...activation,
    registerPluginManifest: (id: string, manifest: Manifest) => {
      manifests.set(id, manifest)
      host.registerManifest?.(id, manifest)
    },
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    getPluginFailure: (id: string): string | undefined => failures.value.get(id),
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
  }
}
export type PluginLoader = ReturnType<typeof createPluginLoader>
