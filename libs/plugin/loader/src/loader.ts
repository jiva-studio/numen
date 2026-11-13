/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting. The plugins of the window
 * slots start with the window; every other plugin starts on the first use of what it contributes.
 * Callers that start a plugin while its start goes on join that start, and the plugin counts as
 * starting until the last of them ends.
 */
import type { Caught, Disposable, JsonValue } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { LoaderHost } from './host'
import {
  removePluginStyles,
  isWindowSlotManifest,
  readPluginManifest,
  resolveModule,
  type PluginModule,
} from './module'
import {
  activateEntry,
  createPluginStatus,
  createSignal,
  startLoader,
  type ActivateContext,
  type ActivePluginEntry,
  type PluginStatus,
  type StartLimit,
} from './loading'

export interface PluginLoaderDeps {
  readonly host: LoaderHost
  readonly status?: PluginStatus
  readonly baseUrl?: string
  readonly fetch?: typeof fetch
  readonly dynamicImport?: (url: string) => Promise<PluginModule>
  readonly onError?: (pluginId: string, error: Caught) => void
}

export const DEFAULT_PLUGIN_ENABLED = true

/** A plugin is enabled unless its setting is false. */
export function isPluginEnabledValue(value: JsonValue | undefined): boolean {
  return typeof value === 'boolean' ? value : DEFAULT_PLUGIN_ENABLED
}

interface DeactivateDeps {
  readonly host: LoaderHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly manifests: Map<string, Manifest>
  readonly reportError: (pluginId: string, error: Caught) => void
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

function createStartRuns(starting: LoaderState['starting']) {
  const runs = new Map<string, number>()
  const going = new Map<string, Promise<boolean>>()
  const count = (id: string, step: number): void => {
    const total = (runs.get(id) ?? 0) + step
    if (total > 0) runs.set(id, total)
    else runs.delete(id)
    const next = new Set(starting.value)
    if (total > 0) next.add(id)
    else next.delete(id)
    starting.value = next
  }
  return {
    track: (id: string, run: () => Promise<boolean>): Promise<boolean> => {
      count(id, 1)
      return run().finally(() => count(id, -1))
    },
    startOnce: (id: string, start: () => Promise<boolean>): Promise<boolean> => {
      const found = going.get(id)
      if (found) return found
      const one = start().finally(() => going.delete(id))
      going.set(id, one)
      return one
    },
  }
}

function createActivation(deps: PluginLoaderDeps, state: LoaderState) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const { activePlugins, failures, manifests, moduleLoaders } = state
  const queues = new Map<string, Promise<void>>()
  const runs = createStartRuns(state.starting)

  const enqueue = <T>(id: string, task: () => Promise<T>): Promise<T> => {
    const run = (queues.get(id) ?? Promise.resolve()).then(task)
    queues.set(
      id,
      run.then(
        () => undefined,
        () => undefined,
      ),
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

  const readManifest = async (id: string): Promise<Manifest | undefined> =>
    moduleLoaders.has(id) ? manifests.get(id) : readPluginManifest(id, activateContext)

  const activatePlugin = (id: string, module?: PluginModule, limit?: StartLimit) =>
    runs.track(id, () => enqueue(id, () => activateEntry(activateContext, id, { module, limit })))

  return {
    loadModule: (id: string) => resolveModule(id, undefined, activateContext),
    readManifest,
    isInterface: async (id: string): Promise<boolean> => {
      const manifest = await readManifest(id)
      return manifest !== undefined && isWindowSlotManifest(manifest)
    },
    activatePlugin,
    /** Starts the plugin once for every caller that asks while its start goes on. */
    startPlugin: (id: string): Promise<boolean> => runs.startOnce(id, () => activatePlugin(id)),
    deactivatePlugin: (id: string) => enqueue(id, () => deactivateEntry(deactivateContext, id)),
  }
}

type LoaderState = Pick<
  PluginStatus['state'],
  'activePlugins' | 'failures' | 'moduleLoaders' | 'manifests' | 'starting'
>

interface Activation {
  readonly loadModule: (id: string) => Promise<PluginModule | undefined>
  readonly readManifest: (id: string) => Promise<Manifest | undefined>
  readonly activatePlugin: (
    id: string,
    module?: PluginModule,
    limit?: StartLimit,
  ) => Promise<boolean>
  readonly startPlugin: (id: string) => Promise<boolean>
  readonly deactivatePlugin: (id: string) => Promise<boolean>
  readonly isInterface: (id: string) => Promise<boolean>
}

function createLoaderStatus(status: PluginStatus, waitForInterface: () => Promise<void>) {
  const { activePlugins, failures } = status.state
  return {
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    getPluginFailure: status.getPluginFailure,
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
    isPluginStarting: status.isPluginStarting,
    startPlugin: status.startPlugin,
    startPluginsOf: status.startPluginsOf,
    findPluginsOf: status.findPluginsOf,
    startTabPlugins: status.startTabPlugins,
    /** Settles once the plugins of the window slots have started or INTERFACE_WAIT_MS has passed. */
    waitForInterface,
  }
}

function createRuntime(deps: PluginLoaderDeps, status: PluginStatus, activation: Activation) {
  const { host } = deps
  const { activatePlugin, deactivatePlugin, loadModule } = activation
  const { activePlugins, moduleLoaders, tracked, isRunning, isEnabled, listPlugins } = status.state
  const settingSubscriptions = new Map<string, Disposable>()
  const interfaceStarted = createSignal()

  const listenDeps = { host, subscriptions: settingSubscriptions }
  const listen = (id: string) =>
    listenToPluginState(
      { ...listenDeps, activate: activatePlugin, deactivate: deactivatePlugin },
      id,
    )

  /** A plugin listed while the window runs starts at once only where it fills a window slot. */
  const startInterfacePlugin = async (id: string): Promise<void> => {
    if (!isEnabled(id) || !(await activation.isInterface(id).catch(() => false))) return
    await activatePlugin(id)
  }

  return {
    status: createLoaderStatus(status, () => interfaceStarted.promise),
    registerPluginModule: (id: string, loader: () => Promise<PluginModule>) => {
      moduleLoaders.set(id, loader)
      if (!isRunning.value) return
      listen(id)
      // Stryker disable next-line BlockStatement,CallExpression,OptionalChaining: activatePlugin handles its own errors inside activateEntry
      startInterfacePlugin(id).catch((error: Caught) => deps.onError?.(id, error))
    },
    trackPlugins: async (ids: readonly string[]): Promise<void> => {
      for (const id of ids) {
        if (tracked.value.has(id)) continue
        tracked.value = new Set([...tracked.value, id])
        if (!isRunning.value) continue
        listen(id)
        await startInterfacePlugin(id)
      }
    },
    start: async (): Promise<void> => {
      isRunning.value = true
      const pluginIds = listPlugins()
      const { settle: onInterfaceStarted } = interfaceStarted
      const startDeps = { ...activation, load: loadModule, pluginIds, listen, isEnabled }
      await startLoader({ ...startDeps, onInterfaceStarted }, activatePlugin).finally(
        onInterfaceStarted,
      )
    },
    close: async (): Promise<void> => {
      isRunning.value = false
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
  const status =
    deps.status ??
    createPluginStatus({
      isEnabled: (id) => isPluginEnabledValue(host.getSetting?.(id, 'enabled')),
    })
  const state: LoaderState = {
    activePlugins: status.state.activePlugins,
    failures: status.state.failures,
    moduleLoaders: status.state.moduleLoaders,
    manifests: status.state.manifests,
    starting: status.state.starting,
  }
  const activation = createActivation(deps, state)
  status.state.starter.current = {
    start: activation.startPlugin,
    readManifest: activation.readManifest,
  }
  const runtime = createRuntime(deps, status, activation)

  return {
    registerPluginModule: runtime.registerPluginModule,
    registerPluginManifest: (id: string, manifest: Manifest) => {
      state.manifests.set(id, manifest)
      host.registerManifest?.(id, manifest)
    },
    trackPlugins: runtime.trackPlugins,
    start: runtime.start,
    close: runtime.close,
    activation,
    status: runtime.status,
  }
}
export type PluginLoader = ReturnType<typeof createPluginLoader>
