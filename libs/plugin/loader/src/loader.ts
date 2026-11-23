/**
 * Loads plugin frontends: each module is fetched from the window, activated with a context
 * created by the host, and deactivated according to its `enabled` setting. The plugins of the window
 * slots start with the window; every other plugin starts on the first use of what it contributes.
 * Callers that start a plugin while its start goes on join that start, and the plugin counts as
 * starting until the last of them ends.
 */
import type { Disposable, JsonValue } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { LoaderHost } from './host'
import type { PluginStyles } from './styles'
import {
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
  readonly styles: PluginStyles
  readonly baseUrl?: string
  readonly fetch?: typeof fetch
  readonly dynamicImport?: (url: string) => Promise<PluginModule>
  readonly onError?: (pluginId: string, error: unknown) => void
}

export const DEFAULT_PLUGIN_ENABLED = true

/** A plugin is enabled unless its setting is false. */
export function isPluginEnabledValue(value: JsonValue | undefined): boolean {
  return typeof value === 'boolean' ? value : DEFAULT_PLUGIN_ENABLED
}

interface DeactivateDeps {
  readonly host: LoaderHost
  readonly styles: PluginStyles
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly manifests: Map<string, Manifest>
  readonly reportError: (pluginId: string, error: unknown) => void
}

async function deactivateEntry(deps: DeactivateDeps, pluginId: string): Promise<boolean> {
  const entry = deps.activePlugins.value.get(pluginId)
  if (!entry) {
    deps.styles.unregister(pluginId)
    deps.manifests.delete(pluginId)
    deps.host.disposePlugin(pluginId)
    return false
  }
  try {
    await entry.deactivate?.()
  } catch (error) {
    deps.reportError(pluginId, error)
  } finally {
    deps.styles.unregister(pluginId)
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
  readonly activate: (id: string) => Promise<boolean>
  readonly deactivate: (id: string) => Promise<boolean>
}

function createSettingSubscriptions(deps: ListenerDeps) {
  const subscriptions = new Map<string, Disposable>()
  return {
    listen: (id: string) => {
      if (subscriptions.has(id) || !deps.host.watchSetting) return
      const sub = deps.host.watchSetting<boolean>(id, 'enabled', async (enabled) => {
        if (isPluginEnabledValue(enabled)) await deps.activate(id)
        else await deps.deactivate(id)
      })
      subscriptions.set(id, sub)
    },
    close: () => {
      for (const sub of subscriptions.values()) sub.dispose()
      subscriptions.clear()
    },
  }
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
    styles: deps.styles,
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

interface LoaderLifetime {
  readonly signal: AbortSignal
  readonly previous: Promise<void> | undefined
}

function createActivation(deps: PluginLoaderDeps, state: LoaderState, lifetime: LoaderLifetime) {
  const { signal } = lifetime
  const { host } = deps
  const defaults = resolveDefaultDeps(deps)
  const { activePlugins, failures, manifests, moduleLoaders } = state
  const queues = new Map<string, Promise<void>>()
  const runs = createStartRuns(state.starting)

  const enqueue = <T>(id: string, task: () => Promise<T>): Promise<T> => {
    const run = (queues.get(id) ?? lifetime.previous ?? Promise.resolve()).then(task)
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
    signal,
    starts: new Map(),
  }
  const readManifest = async (id: string): Promise<Manifest | undefined> =>
    moduleLoaders.has(id) ? manifests.get(id) : readPluginManifest(id, activateContext)

  const activatePlugin = (id: string, module?: PluginModule, limit?: StartLimit) => {
    if (signal.aborted) {
      limit?.stop()
      return Promise.resolve(false)
    }
    return runs.track(id, () =>
      enqueue(id, () => activateEntry(activateContext, id, { module, limit })),
    )
  }
  const deactivate = (id: string) => enqueue(id, () => deactivateEntry(activateContext, id))

  return {
    lifetime,
    closePlugins: async () => {
      await Promise.all(queues.values())
      for (const id of activePlugins.value.keys()) await deactivate(id)
    },
    activation: {
      loadModule: (id: string) =>
        signal.aborted ? Promise.resolve(undefined) : resolveModule(id, undefined, activateContext),
      readManifest,
      isInterface: async (id: string): Promise<boolean> => {
        const manifest = await readManifest(id)
        return manifest !== undefined && isWindowSlotManifest(manifest)
      },
      activatePlugin,
      startPlugin: (id: string): Promise<boolean> => runs.startOnce(id, () => activatePlugin(id)),
      deactivatePlugin: (id: string) => (signal.aborted ? Promise.resolve(false) : deactivate(id)),
    },
  }
}

type LoaderState = Pick<
  PluginStatus['state'],
  'activePlugins' | 'failures' | 'moduleLoaders' | 'manifests' | 'starting'
>

function createLoaderStatus(
  status: PluginStatus,
  waitForInterface: () => Promise<void>,
): LoaderStatus {
  const { failures } = status.state
  return {
    getFailedPlugins: (): { id: string; error: string }[] =>
      [...failures.value].map(([id, error]) => ({ id, error })),
    startPlugin: status.startPlugin,
    startTabPlugins: status.startTabPlugins,
    waitForInterface,
  }
}

function createInterfaceStarter(
  activation: ReturnType<typeof createActivation>['activation'],
  isEnabled: (id: string) => boolean,
) {
  return async (id: string): Promise<void> => {
    if (!isEnabled(id) || !(await activation.isInterface(id).catch(() => false))) return
    await activation.activatePlugin(id)
  }
}

function createRuntime(
  deps: PluginLoaderDeps,
  status: PluginStatus,
  run: ReturnType<typeof createActivation>,
) {
  const { host } = deps
  const { activation, lifetime } = run
  const { activatePlugin, deactivatePlugin, loadModule } = activation
  const { moduleLoaders, tracked, isRunning, isEnabled, listPlugins } = status.state
  const settings = createSettingSubscriptions({
    host,
    activate: activatePlugin,
    deactivate: deactivatePlugin,
  })
  const interfaceStarted = createSignal()

  const startInterfacePlugin = createInterfaceStarter(activation, isEnabled)

  return {
    waitForInterface: () => interfaceStarted.promise,
    registerPluginModule: (id: string, loader: () => Promise<PluginModule>) => {
      moduleLoaders.set(id, loader)
      if (!isRunning.value) return
      settings.listen(id)
      // Stryker disable next-line BlockStatement,CallExpression,OptionalChaining: activatePlugin handles its own errors inside activateEntry
      startInterfacePlugin(id).catch((error: unknown) => deps.onError?.(id, error))
    },
    trackPlugins: async (ids: readonly string[]): Promise<void> => {
      for (const id of ids) {
        if (tracked.value.has(id)) continue
        tracked.value = new Set([...tracked.value, id])
        if (!isRunning.value || lifetime.signal.aborted) continue
        settings.listen(id)
        await startInterfacePlugin(id)
      }
    },
    start: async (): Promise<void> => {
      if (lifetime.previous) await lifetime.previous
      if (lifetime.signal.aborted) return
      isRunning.value = true
      const { settle: onInterfaceStarted } = interfaceStarted
      const startDeps = {
        ...activation,
        load: loadModule,
        pluginIds: listPlugins(),
        listen: settings.listen,
        isEnabled,
      }
      await startLoader({ ...startDeps, onInterfaceStarted }, activatePlugin).finally(
        onInterfaceStarted,
      )
    },
    close: async (): Promise<void> => {
      interfaceStarted.settle()
      if (lifetime.previous) await lifetime.previous
      isRunning.value = false
      settings.close()
      await run.closePlugins()
      for (const id of status.state.manifests.keys()) deps.styles.unregister(id)
      host.disposeAll()
    },
  }
}

function createLoaderRun(deps: PluginLoaderDeps, status: PluginStatus, previous?: Promise<void>) {
  const lifetime = new AbortController()
  const run = createActivation(deps, status.state, { signal: lifetime.signal, previous })
  const { activation } = run
  status.state.starter.current = {
    start: activation.startPlugin,
    readManifest: activation.readManifest,
  }
  const runtime = createRuntime(deps, status, run)
  return {
    runtime,
    activation,
    close: () => {
      lifetime.abort()
      return runtime.close()
    },
  }
}

/** `startPlugin` starts the plugin once for every caller that asks while its start goes on. */
interface PluginActivation {
  readonly loadModule: (id: string) => Promise<PluginModule | undefined>
  readonly readManifest: (id: string) => Promise<Manifest | undefined>
  readonly isInterface: (id: string) => Promise<boolean>
  readonly activatePlugin: (
    id: string,
    module?: PluginModule,
    limit?: StartLimit,
  ) => Promise<boolean>
  readonly startPlugin: (id: string) => Promise<boolean>
  readonly deactivatePlugin: (id: string) => Promise<boolean>
}

/** `waitForInterface` settles once the plugins of the window slots have started or INTERFACE_WAIT_MS has passed. */
interface LoaderStatus {
  readonly getFailedPlugins: () => { id: string; error: string }[]
  readonly startPlugin: PluginStatus['startPlugin']
  readonly startTabPlugins: PluginStatus['startTabPlugins']
  readonly waitForInterface: () => Promise<void>
}

export interface PluginLoader {
  readonly registerPluginModule: (id: string, loader: () => Promise<PluginModule>) => void
  readonly registerPluginManifest: (id: string, manifest: Manifest) => void
  readonly trackPlugins: (ids: readonly string[]) => Promise<void>
  readonly start: () => Promise<void>
  readonly close: () => Promise<void>
  readonly activation: PluginActivation
  readonly status: LoaderStatus
}

export function createPluginLoader(deps: PluginLoaderDeps): PluginLoader {
  const { host } = deps
  const status =
    deps.status ??
    createPluginStatus({
      isEnabled: (id) => isPluginEnabledValue(host.getSetting?.(id, 'enabled')),
    })
  let run = createLoaderRun(deps, status)
  let closing: Promise<void> | undefined
  return {
    registerPluginModule: (id: string, loader: () => Promise<PluginModule>) =>
      run.runtime.registerPluginModule(id, loader),
    registerPluginManifest: (id: string, manifest: Manifest) => {
      status.state.manifests.set(id, manifest)
      host.registerManifest?.(id, manifest)
    },
    trackPlugins: (ids: readonly string[]) => run.runtime.trackPlugins(ids),
    start: async () => {
      if (closing) {
        run = createLoaderRun(deps, status, closing)
        closing = undefined
      }
      await run.runtime.start()
    },
    close: () => (closing ??= run.close()),
    get activation() {
      return run.activation
    },
    status: createLoaderStatus(status, () => run.runtime.waitForInterface()),
  }
}
