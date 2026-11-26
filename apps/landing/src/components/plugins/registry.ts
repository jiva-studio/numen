/**
 * The plugin registry the catalog is built from: the file `tools/plugin-registry/collect.mjs`
 * writes and the registry workflow publishes. The build reads the file named by
 * NUMEN_PLUGIN_REGISTRY_FILE, or else the published one; a registry that cannot be read fails the
 * build.
 */
import { readFile } from 'node:fs/promises'

/** Where the registry is published. */
export const REGISTRY_URL = 'https://dl.numen.md/plugins/registry/latest.json'

/** The served root of the downloads zone, which external screenshots are published under. */
export const DOWNLOADS_BASE = 'https://dl.numen.md'

/** The environment variable that names a local registry file. */
export const REGISTRY_FILE_VARIABLE = 'NUMEN_PLUGIN_REGISTRY_FILE'

/** The registry shape this build reads. */
const REGISTRY_SCHEMA = 1

type PluginPermission = { id: string; description: string }

type PluginSetting = { key: string; title: string; description: string }

type PluginCommand = { id: string; title: string }

type PluginLicense = { id: string; title: string; description: string }

type PluginScreenshot = { name: string; light: string | null; dark: string | null }

export type PluginChange = { version: string; date: string; changes?: string[]; notes?: string }

type PluginDownloadFile = { platform: string; size: number | null; sha256: string }

type PluginDownload = { id: string; kind: string; version: string; platforms: PluginDownloadFile[] }

type PluginArtifact = {
  name: string
  platform: string
  arch: string | null
  size: number
  sha256: string | null
  url: string
}

type PluginCheckStatus = 'pass' | 'warn' | 'fail' | 'info' | 'skipped'

type PluginCheck = { id: string; status: PluginCheckStatus; detail: string }

export type PluginQualityEntry = {
  target: string | null
  kind: string | null
  mutationScore: number | null
  coverage: number | null
  measuredAt: string | null
}

type PluginQuality = { reportedBy: 'numen' | 'author'; entries: PluginQualityEntry[] }

type PluginSource = {
  kind: 'internal' | 'external'
  repo: string
  url: string
  path?: string
  tag?: string
}

/**
 * A plugin as the registry lists it: manifest fields, README, screenshots, changelog, checks and
 * test numbers.
 */
export type RegistryPlugin = {
  id: string
  name: string
  version: string
  description: string
  category: string
  feature: string
  targets: string[]
  isOfficial: boolean
  isPaid: boolean
  licenses: PluginLicense[]
  permissions: PluginPermission[]
  permissionsAdded: string[]
  settings: PluginSetting[]
  commands: PluginCommand[]
  source: PluginSource
  author: string
  license: string | null
  platforms: string[]
  readme: { intro: string; text: string } | null
  screenshots: PluginScreenshot[]
  changelog: PluginChange[]
  updatedAt: string | null
  downloads: PluginDownload[]
  artifacts: PluginArtifact[]
  checks: PluginCheck[]
  quality: PluginQuality | null
}

/** The registry document. */
type Registry = {
  schema: number
  generatedAt: string
  contentHash: string
  plugins: RegistryPlugin[]
}

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isPlugin = (value: unknown): boolean =>
  isObject(value) &&
  typeof value.id === 'string' &&
  typeof value.name === 'string' &&
  typeof value.version === 'string' &&
  Array.isArray(value.targets) &&
  Array.isArray(value.checks) &&
  isObject(value.source)

/** The registry in `text`; an error names `origin` when the text is not a registry this build reads. */
export function parseRegistry(text: string, origin: string): Registry {
  const value: unknown = JSON.parse(text)
  if (
    !isObject(value) ||
    value.schema !== REGISTRY_SCHEMA ||
    !Array.isArray(value.plugins) ||
    value.plugins.length === 0 ||
    !value.plugins.every(isPlugin)
  ) {
    throw new Error(`${origin} is not a plugin registry of schema ${REGISTRY_SCHEMA}`)
  }
  return value as Registry
}

/**
 * The registry from the file NUMEN_PLUGIN_REGISTRY_FILE names, relative to the working folder, or
 * else from REGISTRY_URL. A failed read or fetch throws.
 */
export async function readRegistry(
  env: Readonly<Record<string, string | undefined>> = process.env,
  fetcher: typeof fetch = fetch,
): Promise<Registry> {
  const file = env[REGISTRY_FILE_VARIABLE]
  if (file) return parseRegistry(await readFile(file, 'utf8'), file)
  const response = await fetcher(REGISTRY_URL)
  if (!response.ok) throw new Error(`${REGISTRY_URL} was answered with ${response.status}`)
  return parseRegistry(await response.text(), REGISTRY_URL)
}
