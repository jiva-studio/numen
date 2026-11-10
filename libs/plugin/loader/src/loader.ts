/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting.
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

function createActivation(deps: PluginLoaderDeps, state: LoaderState) {
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const { activePlugins, failures, manifests, moduleLoaders } = state
  const queues = new Map<string, Promise<void>>()

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

  return {
    loadModule: (id: string) => resolveModule(id, undefined, activateContext),
    isInterface: async (id: string): Promise<boolean> =>
      !moduleLoaders.has(id) && isWindowSlotManifest(await readPluginManifest(id, activateContext)),
    hasTabKinds: async (id: string): Promise<boolean> =>
      !moduleLoaders.has(id) &&
      ((await readPluginManifest(id, activateContext).catch(() => undefined))?.contributes?.tabKinds
        .length ?? 0) > 0,
    activatePlugin: (id: string, module?: PluginModule, limit?: StartLimit) =>
      enqueue(id, () => activateEntry(activateContext, id, { module, limit })),
    deactivatePlugin: (id: string) => enqueue(id, () => deactivateEntry(deactivateContext, id)),
  }
}

type LoaderState = Pick<PluginStatus['state'], 'activePlugins' | 'failures' | 'moduleLoaders'> & {
  readonly manifests: Map<string, Manifest>
}

interface Activation {
  readonly loadModule: (id: string) => Promise<PluginModule | undefined>
  readonly activatePlugin: (
    id: string,
    module?: PluginModule,
    limit?: StartLimit,
  ) => Promise<boolean>
  readonly deactivatePlugin: (id: string) => Promise<boolean>
  readonly isInterface: (id: string) => Promise<boolean>
  readonly hasTabKinds: (id: string) => Promise<boolean>
}

function createTabPluginsWait(status: PluginStatus, activation: Activation) {
  const { isEnabled, listPlugins } = status.state
  const { settlePlugin } = status
  return async (): Promise<void> => {
    const withTabs = await Promise.all(
      listPlugins()
        .filter((id) => isEnabled(id))
        .map(async (id) => ((await activation.hasTabKinds(id)) ? [id] : [])),
    )
    await Promise.all(withTabs.flat().map(settlePlugin))
  }
}

function createLoaderStatus(
  status: PluginStatus,
  activation: Activation,
  waitForInterface: () => Promise<void>,
) {
  const { activePlugins, failures } = status.state
  return {
    isPluginActive: (id: string) => activePlugins.value.has(id),
    getActivePluginIds: () => [...activePlugins.value.keys()],
    getPluginFailure: status.getPluginFailure,
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
    isPluginStarting: status.isPluginStarting,
    settlePlugin: status.settlePlugin,
    waitForTabPlugins: createTabPluginsWait(status, activation),
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

  const activateInBackground = (id: string): void => {
    // Stryker disable next-line BlockStatement,CallExpression,OptionalChaining: activatePlugin handles its own errors inside activateEntry
    activatePlugin(id).catch((error: Caught) => {
      deps.onError?.(id, error)
    })
  }

  return {
    status: createLoaderStatus(status, activation, () => interfaceStarted.promise),
    registerPluginModule: (id: string, loader: () => Promise<PluginModule>) => {
      moduleLoaders.set(id, loader)
      if (isRunning.value) {
        listen(id)
        if (isEnabled(id)) activateInBackground(id)
      }
    },
    trackPlugins: async (ids: readonly string[]): Promise<void> => {
      for (const id of ids) {
        if (tracked.value.has(id)) continue
        tracked.value = new Set([...tracked.value, id])
        if (!isRunning.value) continue
        listen(id)
        if (isEnabled(id)) await activatePlugin(id)
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
    manifests: new Map(),
  }
  const activation = createActivation(deps, state)
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
