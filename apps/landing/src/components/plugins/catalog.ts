/**
 * What the catalog pages derive from the registry entries: names of applications and platforms,
 * sizes, dates, the address of a screenshot, the featured and related plugins and issue links.
 */
import {
  DOWNLOADS_BASE,
  type PluginChange,
  type PluginQualityEntry,
  type RegistryPlugin,
} from './registry.ts'

/**
 * A registry entry with its release notes rendered and its place in the featured list, as the
 * content collection keeps it.
 */
export type CatalogPlugin = Omit<RegistryPlugin, 'changelog'> & {
  changelog: (PluginChange & { notesHtml: string | null })[]
  featuredRank: number | null
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

/** The applications of a plugin's targets, in the order of APP_NAMES. */
export function listApps(targets: readonly string[]): { target: string; name: string }[] {
  return Object.entries(APP_NAMES)
    .filter(([target]) => targets.includes(target))
    .map(([target, name]) => ({ target, name }))
}

/** The operating system of a platform: `macos`, `windows`, `linux` or `any`. */
export function getPlatformSystem(platform: string): string {
  const system = platform.split('/')[0] ?? platform
  return system === 'darwin' ? 'macos' : system
}

/** The display name of a platform, `linux/amd64` included. */
export function getPlatformName(platform: string): string {
  const arch = platform.split('/')[1]
  const system = getPlatformSystem(platform)
  const name = PLATFORM_NAMES[system] ?? system
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
  return `${unit === 0 ? value : value.toFixed(value < 9.95 ? 1 : 0)} ${units[unit] ?? ''}`
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

/** How many featured plugins the catalog shows as cards. */
export const FEATURED_CARDS = 3

/** How many featured plugins the catalog shows as icons above its title. */
export const HERO_ICONS = 7

/** The featured plugins, in the order of the registry's featured list. */
export function listFeatured(plugins: readonly CatalogPlugin[]): CatalogPlugin[] {
  return plugins
    .filter((one) => one.featuredRank !== null)
    .sort((a, b) => (a.featuredRank ?? 0) - (b.featuredRank ?? 0))
}

/**
 * Up to `count` plugins to show beside `plugin`: those of its category by name, then the featured
 * plugins of other categories.
 */
export function listRelated(
  plugin: CatalogPlugin,
  plugins: readonly CatalogPlugin[],
  count = 3,
): CatalogPlugin[] {
  const others = plugins.filter((one) => one.id !== plugin.id)
  const sameCategory = others
    .filter((one) => one.category === plugin.category)
    .sort((a, b) => a.name.localeCompare(b.name))
  const featured = listFeatured(others).filter((one) => one.category !== plugin.category)
  return [...sameCategory, ...featured].slice(0, count)
}

/** What a contributor is shown as: their GitHub login, else their name. */
export function getContributorName(person: { name: string; login: string | null }): string {
  return person.login ?? person.name
}

const GITHUB_LOGIN = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$/

/** The GitHub profile of a contributor, or null when they have no valid login. */
export function getContributorUrl(person: { login: string | null }): string | null {
  return person.login && GITHUB_LOGIN.test(person.login)
    ? `https://github.com/${person.login}`
    : null
}

const AVATAR_HOST = 'avatars.githubusercontent.com'

/**
 * The GitHub avatar of a contributor at `size` pixels, or null when they have none or it is not
 * served from GitHub's avatar host.
 */
export function getAvatarUrl(person: { avatarUrl: string | null }, size: number): string | null {
  if (!person.avatarUrl) return null
  let url: URL
  try {
    url = new URL(person.avatarUrl)
  } catch {
    return null
  }
  if (url.protocol !== 'https:' || url.hostname !== AVATAR_HOST) return null
  url.searchParams.set('s', String(size))
  return url.toString()
}

/** The first letter of a name in upper case, for an avatar. */
export function getInitial(name: string): string {
  return [...name.trim()][0]?.toUpperCase() ?? '?'
}

const getIssueUrl = (plugin: CatalogPlugin, template: string | null): string => {
  const query = new URLSearchParams()
  if (plugin.source.kind === 'internal' && template !== null) query.set('template', template)
  query.set('title', `${plugin.name}: `)
  return `https://github.com/${plugin.source.repo}/issues/new?${query.toString()}`
}

/** A new GitHub issue reporting a bug of the plugin, titled with its name. */
export function getBugReportUrl(plugin: CatalogPlugin): string {
  return getIssueUrl(plugin, 'bug.yml')
}

/** A new GitHub issue asking for a feature of the plugin, titled with its name. */
export function getFeatureRequestUrl(plugin: CatalogPlugin): string {
  return getIssueUrl(plugin, 'task.yml')
}
