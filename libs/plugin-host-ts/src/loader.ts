/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting.
 */
import { shallowRef, watch, type ShallowRef } from 'vue'
import type { Disposable } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { LoaderHost } from './host'
import {
  getLifecycleHooks,
  removePluginStyles,
  isInterfaceManifest,
  readPluginManifest,
  resolveModule,
  type ResolveModuleContext,
} from './module'
import { PLUGIN_START_LIMIT_MS } from './loading/limits'
import { startLoader } from './loading/startLoader'

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
  readonly deactivate?: (() => Promise<void> | void) | undefined
}

interface ActivateContext extends ResolveModuleContext {
  readonly host: LoaderHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly failures: { value: ReadonlyMap<string, string> }
  readonly reportError: (pluginId: string, error: unknown) => void
  /** The number of the latest start of each plugin, so a start cancelled at the limit takes back only its own. */
  readonly starts: Map<string, number>
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

/** Takes back what a plugin registered, for a start that failed. */
function discardEntry(ctx: ActivateContext, pluginId: string): void {
  removePluginStyles(pluginId)
  ctx.host.disposePlugin(pluginId)
}

/**
 * Reads and activates the plugin. A start that ends after the limit has passed is taken back, unless the
 * plugin has been started again since.
 */
async function startEntry(
  ctx: ActivateContext,
  pluginId: string,
  start: { readonly module: unknown; readonly isExpired: () => boolean; readonly number: number },
): Promise<boolean> {
  const { module: pluginModule, isExpired } = start
  const discardExpired = (): void => {
    if (ctx.starts.get(pluginId) === start.number) discardEntry(ctx, pluginId)
  }
  const { host, activePlugins } = ctx
  try {
    const resolved = await resolveModule(pluginId, pluginModule, ctx)
    if (!resolved) return false

    const manifest = ctx.manifests.get(pluginId)
    const { activate, deactivate } = getLifecycleHooks(resolved)
    if (activate) {
      host.markPluginStarting?.(pluginId)
      await activate(host.createContext(pluginId, manifest))
      host.markPluginStarted?.(pluginId)
    }
    if (isExpired()) {
      discardExpired()
      return false
    }

    const next = new Map(activePlugins.value)
    next.set(pluginId, { pluginId, deactivate })
    activePlugins.value = next
    setFailure(ctx, pluginId, undefined)
    return true
  } catch (error) {
    if (isExpired()) {
      discardExpired()
      return false
    }
    discardEntry(ctx, pluginId)
    setFailure(ctx, pluginId, getFailureMessage(error))
    ctx.reportError(pluginId, error)
    return false
  }
}

/**
 * Starts the plugin within PLUGIN_START_LIMIT_MS. A plugin that has not started by then counts as
 * failed: what it has registered is taken back at once, what its start registers afterwards is taken
 * back too, and that start never makes it active. Its backend is left running, and its next use starts
 * it again.
 */
async function activateEntry(
  ctx: ActivateContext,
  pluginId: string,
  pluginModule?: unknown,
): Promise<boolean> {
  if (ctx.activePlugins.value.has(pluginId)) return true
  let isExpired = false
  let timer: ReturnType<typeof setTimeout> | undefined
  const limit = new Promise<'expired'>((resolve) => {
    timer = setTimeout(() => resolve('expired'), PLUGIN_START_LIMIT_MS)
  })
  const number = (ctx.starts.get(pluginId) ?? 0) + 1
  ctx.starts.set(pluginId, number)
  const starting = startEntry(ctx, pluginId, {
    module: pluginModule,
    isExpired: () => isExpired,
    number,
  })
  const outcome = await Promise.race([starting, limit]).finally(() => clearTimeout(timer))
  if (outcome !== 'expired') return outcome
  isExpired = true
  discardEntry(ctx, pluginId)
  const error = new Error(`the plugin did not start within ${PLUGIN_START_LIMIT_MS / 1000} s`)
  setFailure(ctx, pluginId, error.message)
  ctx.reportError(pluginId, error)
  return false
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
    await entry.deactivate?.()
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
      // Stryker disable next-line ArrowFunction: default globalThis.fetch wrapper is overridden in unit tests
      ((input: RequestInfo | URL, init?: RequestInit) => globalThis.fetch(input, init)),
    // Stryker disable next-line ArrowFunction: default dynamic import wrapper is overridden in unit tests
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
    starts: new Map(),
  }
  const deactivateContext: DeactivateDeps = {
    host,
    activePlugins,
    manifests,
    reportError: defaults.reportError,
  }

  return {
    loadModule: (id: string) => resolveModule(id, undefined, activateContext),
    isInterface: async (id: string): Promise<boolean> =>
      !moduleLoaders.has(id) && isInterfaceManifest(await readPluginManifest(id, activateContext)),
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
  readonly isInterface: (id: string) => Promise<boolean>
}

/** A promise and the function that settles it. */
function createSignal(): { readonly promise: Promise<void>; readonly settle: () => void } {
  let settle: () => void = () => undefined
  const promise = new Promise<void>((resolve) => {
    settle = resolve
  })
  return { promise, settle }
}

function createRuntime(deps: PluginLoaderDeps, state: LoaderState, activation: Activation) {
  const { host } = deps
  const { activatePlugin, deactivatePlugin, loadModule } = activation
  const { activePlugins, moduleLoaders } = state
  const trackedIds = new Set<string>()
  const settingSubscriptions = new Map<string, Disposable>()
  let hasStarted = false
  const interfaceStarted = createSignal()

  const isEnabled = (id: string): boolean => isPluginEnabledValue(host.getSetting?.(id, 'enabled'))
  const listenDeps = { host, subscriptions: settingSubscriptions }
  const listen = (id: string) =>
    listenToPluginState(
      { ...listenDeps, activate: activatePlugin, deactivate: deactivatePlugin },
      id,
    )

  const activateInBackground = (id: string): void => {
    // Stryker disable next-line BlockStatement,CallExpression,OptionalChaining: activatePlugin handles its own errors inside activateEntry
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
      const { settle: onInterfaceStarted } = interfaceStarted
      const startDeps = { ...activation, load: loadModule, pluginIds, listen, isEnabled }
      await startLoader({ ...startDeps, onInterfaceStarted }, activatePlugin).finally(
        onInterfaceStarted,
      )
    },
    /** Settles once the plugins of the interface have started or INTERFACE_WAIT_MS has passed. */
    waitForInterface: (): Promise<void> => interfaceStarted.promise,
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

  /**
   * Starts again, within PLUGIN_START_LIMIT_MS, an enabled plugin whose last start failed; its failure
   * stays until a start succeeds. Settles with whether the plugin is active.
   */
  const startFailedPlugin = async (id: string): Promise<boolean> => {
    if (!failures.value.has(id)) return activePlugins.value.has(id)
    if (!isPluginEnabledValue(host.getSetting?.(id, 'enabled'))) return false
    return activation.activatePlugin(id)
  }

  return {
    ...createRuntime(deps, state, activation),
    ...activation,
    registerPluginManifest: (id: string, manifest: Manifest) => {
      manifests.set(id, manifest)
      host.registerManifest?.(id, manifest)
    },
    startFailedPlugin,
    /**
     * Settles once the plugin is active or has failed, or after PLUGIN_START_LIMIT_MS. A plugin whose last
     * start failed is started again first.
     */
    settlePlugin: (id: string): Promise<void> =>
      failures.value.has(id)
        ? startFailedPlugin(id).then(() => undefined)
        : new Promise<void>((resolve) => {
            const isSettled = () => activePlugins.value.has(id) || failures.value.has(id)
            if (isSettled()) return resolve()
            const stop = watch([activePlugins, failures], () => {
              if (!isSettled()) return
              stop()
              clearTimeout(timer)
              resolve()
            })
            const timer = setTimeout(() => {
              stop()
              resolve()
            }, PLUGIN_START_LIMIT_MS)
          }),
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    getPluginFailure: (id: string): string | undefined => failures.value.get(id),
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
  }
}
export type PluginLoader = ReturnType<typeof createPluginLoader>
