/**
 * What the catalog pages derive from a registry entry: names of applications and platforms,
 * sizes, dates and the address of a screenshot.
 */
import {
  DOWNLOADS_BASE,
  type PluginChange,
  type PluginQualityEntry,
  type RegistryPlugin,
} from './registry'

/** A registry entry with its README and release notes rendered, as the content collection keeps it. */
export type CatalogPlugin = Omit<RegistryPlugin, 'changelog'> & {
  changelog: (PluginChange & { notesHtml: string | null })[]
}

/** The applications a plugin target names. */
export const APP_NAMES: Readonly<Record<string, string>> = {
  'desktop.editor': 'Numen',
  'desktop.flashcards': 'Numen Flashcards',
}

const PLATFORM_NAMES: Readonly<Record<string, string>> = {
  macos: 'macOS',
  windows: 'Windows',
  linux: 'Linux',
  any: 'Any platform',
}

const KIND_NAMES: Readonly<Record<string, string>> = { go: 'Go', ts: 'TypeScript' }

/** The application names of a plugin's targets, in the order of APP_NAMES. */
export function getAppNames(targets: readonly string[]): string[] {
  return Object.entries(APP_NAMES)
    .filter(([target]) => targets.includes(target))
    .map(([, name]) => name)
}

/** The display name of a platform, `linux/amd64` included. */
export function getPlatformName(platform: string): string {
  const [system = platform, arch] = platform.split('/')
  const name = PLATFORM_NAMES[system === 'darwin' ? 'macos' : system] ?? system
  return arch ? `${name} ${arch}` : name
}

/** A category name with its first letter in upper case. */
export function getCategoryName(category: string): string {
  return category === '' ? 'Other' : category.charAt(0).toUpperCase() + category.slice(1)
}

/** The language a quality entry measured, or an empty string for a plugin measured as a whole. */
export function getQualityKind(entry: PluginQualityEntry): string {
  return entry.kind ? (KIND_NAMES[entry.kind] ?? entry.kind) : ''
}

/** A byte count in KB, MB or GB; an unknown size is an empty string. */
export function formatSize(bytes: number | null): string {
  if (bytes === null) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1000 && unit < units.length - 1) {
    value /= 1000
    unit += 1
  }
  return `${unit === 0 ? value : value.toFixed(value < 10 ? 1 : 0)} ${units[unit] ?? ''}`
}

/** An ISO time as a date in English, in UTC. */
export function formatDate(iso: string | null): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('en', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(iso))
}

/** A percentage with at most one decimal. */
export function formatPercent(value: number | null): string {
  return value === null ? '' : `${Math.round(value * 10) / 10}%`
}

/** The address of a screenshot of an external plugin, which is published in the downloads zone. */
export function getPublishedUrl(path: string): string {
  return `${DOWNLOADS_BASE}/${path}`
}

/** The categories of the plugins, sorted. */
export function listCategories(plugins: readonly { category: string }[]): string[] {
  return [...new Set(plugins.map((one) => one.category))].sort((a, b) => a.localeCompare(b))
}

/** The text a plugin is found by in the catalog search, in lower case. */
export function getSearchText(
  plugin: Pick<RegistryPlugin, 'name' | 'id' | 'author' | 'description' | 'feature' | 'readme'>,
): string {
  const intro = plugin.readme?.intro ?? ''
  return [plugin.name, plugin.id, plugin.author, plugin.description, plugin.feature, intro]
    .join(' ')
    .toLowerCase()
}

/** The path of a plugin's page. */
export function getPluginPath(id: string): string {
  return `/plugins/${id}`
}
