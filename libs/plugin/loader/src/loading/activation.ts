/**
 * One start of a plugin frontend: its module is read and activated with a context of its own within
 * the start limit, and a start that fails or passes the limit takes back what it registered.
 */
import type { Caught } from '@numen/plugin-sdk'
import type { LoaderHost } from '../host'
import {
  parseLifecycleHooks,
  removePluginStyles,
  resolveModule,
  type PluginModule,
  type ResolveModuleContext,
} from '../module'
import { PLUGIN_START_LIMIT_MS } from './limits'
import { createStartScope, type StartScope } from './startContext'
import { createStartLimit, type StartLimit } from './startLimit'

export interface ActivePluginEntry {
  readonly pluginId: string
  readonly deactivate?: (() => Promise<void> | void) | undefined
}

export interface ActivateContext extends ResolveModuleContext {
  readonly host: LoaderHost
  readonly activePlugins: { value: ReadonlyMap<string, ActivePluginEntry> }
  readonly failures: { value: ReadonlyMap<string, string> }
  readonly reportError: (pluginId: string, error: Caught) => void
  readonly starts: Map<string, number>
}

function getFailureMessage(error: Caught): string {
  return error instanceof Error ? error.message : `the plugin threw a ${typeof error}, not an Error`
}

function setFailure(ctx: ActivateContext, pluginId: string, error: string | undefined): void {
  const next = new Map(ctx.failures.value)
  if (error === undefined) next.delete(pluginId)
  else next.set(pluginId, error)
  ctx.failures.value = next
}

function discardEntry(ctx: ActivateContext, pluginId: string): void {
  removePluginStyles(pluginId)
  ctx.host.disposePlugin(pluginId)
}

/** What a start ends with: whether the plugin became active, or 'expired' once it has passed its limit. */
type StartOutcome = boolean | 'expired'

/** One start of a plugin: its module, its number, its limit and what it registers. */
interface EntryStart {
  readonly module: PluginModule | undefined
  readonly number: number
  readonly limit: StartLimit
  readonly scope: StartScope
}

async function activateModule(
  ctx: ActivateContext,
  pluginId: string,
  start: EntryStart & { readonly resolved: PluginModule },
): Promise<StartOutcome> {
  const { host, activePlugins } = ctx
  const isLatest = (): boolean => ctx.starts.get(pluginId) === start.number
  const { activate, deactivate } = parseLifecycleHooks(start.resolved)
  if (activate) {
    if (isLatest()) host.markPluginStarting?.(pluginId)
    await activate(
      start.scope.wrapContext(host.createContext(pluginId, ctx.manifests.get(pluginId))),
    )
    if (isLatest()) host.markPluginStarted?.(pluginId)
  }
  if (start.scope.isCancelled() || start.limit.isPassed()) {
    if (isLatest()) discardEntry(ctx, pluginId)
    return 'expired'
  }
  activePlugins.value = new Map(activePlugins.value).set(pluginId, { pluginId, deactivate })
  setFailure(ctx, pluginId, undefined)
  start.scope.end()
  return true
}

async function startEntry(
  ctx: ActivateContext,
  pluginId: string,
  start: EntryStart,
): Promise<StartOutcome> {
  const isExpired = (): boolean => start.scope.isCancelled() || start.limit.isPassed()
  try {
    const resolved = await resolveModule(pluginId, start.module, ctx)
    if (isExpired()) return 'expired'
    if (!resolved) return false
    return await activateModule(ctx, pluginId, { ...start, resolved })
  } catch (error) {
    if (isExpired()) {
      if (ctx.starts.get(pluginId) === start.number) discardEntry(ctx, pluginId)
      return 'expired'
    }
    start.scope.cancel()
    discardEntry(ctx, pluginId)
    setFailure(ctx, pluginId, getFailureMessage(error))
    ctx.reportError(pluginId, error)
    return false
  }
}

/**
 * Starts the plugin within its limit, which runs from the read of its module where the caller passes
 * it and from now otherwise. A plugin that has not started by then counts as failed: what it has
 * registered is taken back at once, what its start registers afterwards is taken back too, and it
 * never becomes active. Its backend is left running.
 */
export async function activateEntry(
  ctx: ActivateContext,
  pluginId: string,
  begun: { readonly module?: PluginModule | undefined; readonly limit?: StartLimit | undefined },
): Promise<boolean> {
  const limit = begun.limit ?? createStartLimit()
  if (ctx.activePlugins.value.has(pluginId)) {
    limit.stop()
    return true
  }
  // A new start is going on: the failure of an earlier one is no longer where the plugin stands.
  if (ctx.failures.value.has(pluginId)) setFailure(ctx, pluginId, undefined)
  const scope = createStartScope()
  const number = (ctx.starts.get(pluginId) ?? 0) + 1
  ctx.starts.set(pluginId, number)
  const starting = startEntry(ctx, pluginId, { module: begun.module, limit, scope, number })
  const outcome = await Promise.race([starting, limit.expired]).finally(limit.stop)
  if (outcome !== 'expired') return outcome
  scope.cancel()
  discardEntry(ctx, pluginId)
  const error = new Error(`the plugin did not start within ${PLUGIN_START_LIMIT_MS / 1000} s`)
  setFailure(ctx, pluginId, error.message)
  ctx.reportError(pluginId, error)
  return false
}
