/**
 * How a plugin module is found and read: its stylesheet, its lifecycle hooks, its loader.
 */
import { parseManifest, type Manifest } from '@numen/core-client'
import type { PluginLifecycle, SlotName, Unvalidated } from '@numen/plugin-sdk'
import { WINDOW_SLOTS } from './slots'
import { loadPluginStyle, unregisterPluginStyle } from './styles'

/** The bundle every plugin frontend builds to, relative to the plugin folder. */
const FRONTEND_ENTRYPOINT = 'dist/index.js'

/** The stylesheet every plugin frontend builds to, relative to the plugin folder. */
const FRONTEND_STYLES = 'dist/styles.css'

export function removePluginStyles(pluginId: string): void {
  unregisterPluginStyle(pluginId)
}

/**
 * What a plugin bundle exports: the lifecycle hooks at the top or under `default`. An ES module
 * namespace is an object whatever the bundle holds, so the hooks are read where they are found.
 */
export type PluginModule = object

type HookName = keyof Pick<PluginLifecycle, 'activate' | 'deactivate'>

function readHook<K extends HookName>(
  sources: readonly PluginModule[],
  key: K,
): PluginLifecycle[K] | undefined {
  for (const source of sources) {
    const hook = (source as Partial<PluginLifecycle>)[key]
    if (typeof hook === 'function') return hook
  }
  return undefined
}

/** The lifecycle hooks a loaded bundle exports, at the top or under `default`; a bundle that exports none yields none. */
export function parseLifecycleHooks(resolved: Unvalidated): {
  activate?: PluginLifecycle['activate'] | undefined
  deactivate?: PluginLifecycle['deactivate'] | undefined
} {
  if (typeof resolved !== 'object' || resolved === null) return {}
  const inner = (resolved as { default?: PluginModule | null }).default
  const sources = typeof inner === 'object' && inner !== null ? [inner, resolved] : [resolved]
  return { activate: readHook(sources, 'activate'), deactivate: readHook(sources, 'deactivate') }
}

export interface LoadModuleOptions {
  readonly baseUrl: string
  readonly fetchFn: typeof fetch
  readonly importFn: (url: string) => Promise<PluginModule>
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
): Promise<PluginModule | undefined> {
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
  readonly loaders: Map<string, () => Promise<PluginModule>>
}

export async function resolveModule(
  pluginId: string,
  candidate: PluginModule | undefined,
  ctx: ResolveModuleContext,
): Promise<PluginModule | undefined> {
  if (candidate !== undefined) {
    if (typeof candidate === 'function' && !('activate' in candidate)) {
      return (candidate as () => Promise<PluginModule>)()
    }
    return candidate
  }
  const loader = ctx.loaders.get(pluginId)
  if (loader) return loader()
  return loadDynamicModule(pluginId, ctx)
}
