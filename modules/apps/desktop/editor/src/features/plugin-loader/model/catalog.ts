import type { PluginManifest } from '@numen/plugin-sdk'

export type PluginModuleLoader = () => Promise<unknown>

export interface PluginCatalog {
  readonly manifests: Record<string, PluginManifest>
  readonly loaders: Record<string, PluginModuleLoader>
}

const MANIFEST_PATH = /\/plugins\/([^/]+)\/manifest\.json$/
const FRONTEND_PATH = /\/plugins\/([^/]+)\/frontend\/index\.ts$/

function getDirectory(path: string, pattern: RegExp): string | undefined {
  return pattern.exec(path)?.[1]
}

/**
 * Builds the plugin catalog from the files found under `plugins/`: the manifests keyed by
 * their `id`, and one module loader per plugin that has a frontend directory.
 * A frontend without a manifest, and two manifests of one id, are faults and throw.
 */
export function buildPluginCatalog(
  manifestFiles: Record<string, unknown>,
  frontendFiles: Record<string, PluginModuleLoader>,
): PluginCatalog {
  const idByDirectory = new Map<string, string>()
  const directoryById = new Map<string, string>()
  const manifests: Record<string, PluginManifest> = {}
  for (const [path, file] of Object.entries(manifestFiles)) {
    const directory = getDirectory(path, MANIFEST_PATH)
    const manifest = file as PluginManifest
    if (directory === undefined || typeof manifest?.id !== 'string') continue
    const owner = directoryById.get(manifest.id)
    if (owner !== undefined) {
      throw new Error(`plugins "${owner}" and "${directory}" declare the same id "${manifest.id}"`)
    }
    directoryById.set(manifest.id, directory)
    idByDirectory.set(directory, manifest.id)
    manifests[manifest.id] = manifest
  }
  const loaders: Record<string, PluginModuleLoader> = {}
  for (const [path, loader] of Object.entries(frontendFiles)) {
    const directory = getDirectory(path, FRONTEND_PATH)
    if (directory === undefined) continue
    const id = idByDirectory.get(directory)
    if (id === undefined) {
      throw new Error(`plugin "${directory}" has a frontend directory and no manifest.json`)
    }
    loaders[id] = loader
  }
  return { manifests, loaders }
}
