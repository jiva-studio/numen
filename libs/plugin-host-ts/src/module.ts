/**
 * How a plugin module is found and read: its stylesheet, its lifecycle hooks, its loader.
 */
import { parseManifest, type Manifest, type PluginLifecycle } from '@numen/plugin-sdk'
import { loadPluginStyle, unregisterPluginStyle } from './styles'

function getStyleUrl(baseUrl: string, pluginId: string, stylesPath: string): string {
  const normalizedPath = stylesPath.startsWith('/') ? stylesPath.slice(1) : stylesPath
  return `${baseUrl}/${pluginId}/${normalizedPath}`
}

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

async function loadDynamicModule(
  pluginId: string,
  options: LoadModuleOptions,
): Promise<unknown | undefined> {
  const { baseUrl, fetchFn, importFn, manifests } = options
  const res = await fetchFn(`${baseUrl}/${pluginId}/manifest.json`)
  if (!res.ok) {
    throw new Error(`manifest.json of plugin "${pluginId}" answered ${res.status}`)
  }
  const document: unknown = await res.json()
  const parsed = parseManifest(document)
  if (!parsed.ok) {
    throw new Error(`manifest.json of plugin "${pluginId}" is invalid: ${parsed.error}`)
  }
  const manifest = parsed.value
  manifests.set(pluginId, manifest)
  if (manifest.frontend === undefined) return undefined
  const { styles } = manifest.frontend
  const style = styles
    ? loadPluginStyle(pluginId, getStyleUrl(baseUrl, pluginId, styles), fetchFn)
    : undefined
  const entrypoint =
    manifest.frontend.entrypoint === '' ? 'dist/index.js' : manifest.frontend.entrypoint
  const normalizedEntrypoint = entrypoint.startsWith('/') ? entrypoint.slice(1) : entrypoint
  const [loaded] = await Promise.all([
    importFn(`${baseUrl}/${pluginId}/${normalizedEntrypoint}`),
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
