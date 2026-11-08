/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting.
 */
import { shallowRef, type ShallowRef } from 'vue'
import type { Disposable } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { LoaderHost } from './host'
import {
  removePluginStyles,
  isWindowSlotManifest,
  readPluginManifest,
  resolveModule,
} from './module'
import { startLoader } from './loading/startLoader'
import { activateEntry, type ActivateContext, type ActivePluginEntry } from './loading/activation'
import type { StartLimit } from './loading/startLimit'
import { createPluginStarts } from './loading/pluginStarts'
import { createSignal } from './loading/signal'

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
      !moduleLoaders.has(id) && isWindowSlotManifest(await readPluginManifest(id, activateContext)),
    hasTabKinds: async (id: string): Promise<boolean> =>
      !moduleLoaders.has(id) &&
      ((await readPluginManifest(id, activateContext).catch(() => undefined))?.contributes?.tabKinds
        .length ?? 0) > 0,
    activatePlugin: (id: string, module?: unknown, limit?: StartLimit) =>
      enqueue(id, () => activateEntry(activateContext, id, { module, limit })),
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
  readonly activatePlugin: (id: string, module?: unknown, limit?: StartLimit) => Promise<boolean>
  readonly deactivatePlugin: (id: string) => Promise<boolean>
  readonly isInterface: (id: string) => Promise<boolean>
  readonly hasTabKinds: (id: string) => Promise<boolean>
}

function createTracking(deps: PluginLoaderDeps, state: LoaderState, activation: Activation) {
  const tracked = shallowRef<ReadonlySet<string>>(new Set())
  const isRunning = shallowRef(false)
  const isEnabled = (id: string): boolean =>
    isPluginEnabledValue(deps.host.getSetting?.(id, 'enabled'))
  const listPlugins = (): string[] => [
    ...new Set([...state.moduleLoaders.keys(), ...tracked.value]),
  ]
  const starts = createPluginStarts({
    isRunning,
    listPlugins,
    isEnabled,
    activePlugins: state.activePlugins,
    failures: state.failures,
    hasTabKinds: activation.hasTabKinds,
  })
  return { tracked, isRunning, isEnabled, listPlugins, starts }
}

function createRuntime(deps: PluginLoaderDeps, state: LoaderState, activation: Activation) {
  const { host } = deps
  const { activatePlugin, deactivatePlugin, loadModule } = activation
  const { activePlugins, moduleLoaders } = state
  const { tracked, isRunning, isEnabled, listPlugins, starts } = createTracking(
    deps,
    state,
    activation,
  )
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
    activatePlugin(id).catch((error: unknown) => {
      deps.onError?.(id, error)
    })
  }

  return {
    ...starts,
    registerPluginModule: (id: string, loader: () => Promise<unknown>) => {
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
    /** Settles once the plugins of the window slots have started or INTERFACE_WAIT_MS has passed. */
    waitForInterface: (): Promise<void> => interfaceStarted.promise,
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
