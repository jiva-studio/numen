/**
 * How a plugin module is found and read: its stylesheet, its lifecycle hooks, its loader.
 */
import { parseManifest, type Manifest } from '@numen/core-client'
import { WINDOW_SLOTS, type PluginLifecycle, type SlotName } from '@numen/plugin-sdk'
import { loadPluginStyle, unregisterPluginStyle } from './styles'

/** The bundle every plugin frontend builds to, relative to the plugin folder. */
const FRONTEND_ENTRYPOINT = 'dist/index.js'

/** The stylesheet every plugin frontend builds to, relative to the plugin folder. */
const FRONTEND_STYLES = 'dist/styles.css'

export function removePluginStyles(pluginId: string): void {
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

export function getLifecycleHooks(resolved: unknown): {
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

export interface LoadModuleOptions {
  readonly baseUrl: string
  readonly fetchFn: typeof fetch
  readonly importFn: (url: string) => Promise<unknown>
  readonly manifests: Map<string, Manifest>
}

/** The manifest.json of a plugin, read from the window once and kept in `manifests`. */
export async function readPluginManifest(
  pluginId: string,
  options: Pick<LoadModuleOptions, 'baseUrl' | 'fetchFn' | 'manifests'>,
): Promise<Manifest> {
  const { baseUrl, fetchFn, manifests } = options
  const cached = manifests.get(pluginId)
  if (cached) return cached
  const res = await fetchFn(`${baseUrl}/${pluginId}/manifest.json`)
  if (!res.ok) {
    throw new Error(`manifest.json of plugin "${pluginId}" answered ${res.status}`)
  }
  const parsed = parseManifest(await res.text())
  if (!parsed.ok) {
    throw new Error(`manifest.json of plugin "${pluginId}" is invalid: ${parsed.error}`)
  }
  manifests.set(pluginId, parsed.value)
  return parsed.value
}

/** Whether the plugin fills a slot of the window itself, which the window waits for before it is drawn. */
export function isWindowSlotManifest(manifest: Manifest): boolean {
  return (manifest.contributes?.slots ?? []).some((one) =>
    WINDOW_SLOTS.includes(one.slot as SlotName),
  )
}

async function loadDynamicModule(
  pluginId: string,
  options: LoadModuleOptions,
): Promise<unknown | undefined> {
  const { baseUrl, fetchFn, importFn } = options
  const manifest = await readPluginManifest(pluginId, options)
  if (manifest.frontend === undefined) return undefined
  const style = loadPluginStyle(pluginId, `${baseUrl}/${pluginId}/${FRONTEND_STYLES}`, fetchFn)
  const [loaded] = await Promise.all([
    importFn(`${baseUrl}/${pluginId}/${FRONTEND_ENTRYPOINT}`),
    style,
  ])
  return loaded
}

export interface ResolveModuleContext extends LoadModuleOptions {
  readonly loaders: Map<string, () => Promise<unknown>>
}

export async function resolveModule(
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
