/**
 * Query parser and filter for the plugin list.
 *
 * Supports composable `@tag` and `key:value` qualifiers alongside free-text terms (for example
 * `@disabled`, `disabled:true`, `reader:markdown`, `feature:files`, `category:views`,
 * `capability:agent`, `id:notes`). The `feature:` and `category:` qualifiers read what the
 * manifest declares under `contributes`.
 */
import { OTHER_CATEGORY, isInstalled, type ManagedPlugin } from './plugins'

interface FilterQualifier {
  readonly key: string
  readonly value: string
}

export interface ParsedPluginQuery {
  readonly qualifiers: readonly FilterQualifier[]
  readonly terms: readonly string[]
}

export interface PluginFilterContext {
  readonly isEnabled: (pluginId: string) => boolean
}

export interface FlatPluginItem {
  readonly category: string
  readonly categoryLabel: string
  readonly plugin: ManagedPlugin
}

type QualifierMatcher = (
  plugin: ManagedPlugin,
  value: string,
  context: PluginFilterContext,
) => boolean

const TAG_ALIASES: Readonly<Record<string, FilterQualifier>> = {
  '@disabled': { key: 'status', value: 'disabled' },
  '@enabled': { key: 'status', value: 'enabled' },
  '@available': { key: 'status', value: 'available' },
}

const KEY_VALUE_ALIASES: Readonly<Record<string, Readonly<Record<string, FilterQualifier>>>> = {
  disabled: {
    true: { key: 'status', value: 'disabled' },
    false: { key: 'status', value: 'enabled' },
  },
  enabled: {
    true: { key: 'status', value: 'enabled' },
    false: { key: 'status', value: 'disabled' },
  },
}

const SYSTEM_KEYWORDS = ['system', 'theme', 'appearance', 'window', 'scale', 'naming', 'sync']

function toSearchStems(value: string): readonly string[] {
  const clean = value.trim().toLowerCase()
  if (!clean) return []
  const singular = clean.length > 3 && clean.endsWith('s') ? clean.slice(0, -1) : clean
  return singular !== clean ? [clean, singular] : [clean]
}

/** The normalized category key of a plugin: what its manifest declares, or the catch-all. */
export function getPluginCategory(plugin: ManagedPlugin): string {
  return plugin.category === '' ? OTHER_CATEGORY : plugin.category
}

/** Formats a category key into its display label. */
export function formatCategoryLabel(category: string): string {
  return category.charAt(0).toUpperCase() + category.slice(1)
}

function matchesReader(plugin: ManagedPlugin, value: string): boolean {
  const readers = plugin.info?.readers ?? []
  if (!value) return readers.length > 0
  const stems = toSearchStems(value)
  return readers.some((reader) => {
    const readerId = reader.id.toLowerCase()
    return (
      stems.some((stem) => readerId.includes(stem)) ||
      reader.extensions.some((ext) => stems.some((stem) => ext.toLowerCase().includes(stem)))
    )
  })
}

function matchesCapability(plugin: ManagedPlugin, value: string): boolean {
  const capabilities = plugin.info?.capabilities ?? []
  if (!value) return capabilities.length > 0
  const stems = toSearchStems(value)
  return capabilities.some((cap) => stems.some((stem) => cap.toLowerCase().includes(stem)))
}

function matchesFeature(plugin: ManagedPlugin, value: string): boolean {
  if (!value) return plugin.feature !== ''
  const feature = plugin.feature.toLowerCase()
  const kinds = plugin.tabKinds.map((kind) => kind.toLowerCase())
  return toSearchStems(value).some(
    (stem) => feature.includes(stem) || kinds.some((kind) => kind.includes(stem)),
  )
}

function matchesStatus(plugin: ManagedPlugin, value: string, context: PluginFilterContext) {
  if (value === 'available') return !isInstalled(plugin)
  if (!isInstalled(plugin)) return false
  return value === 'disabled' ? !context.isEnabled(plugin.id) : context.isEnabled(plugin.id)
}

function matchesCategory(plugin: ManagedPlugin, value: string): boolean {
  const category = getPluginCategory(plugin)
  return category.includes(value) || formatCategoryLabel(category).toLowerCase().includes(value)
}

const QUALIFIER_MATCHERS: Readonly<Record<string, QualifierMatcher>> = {
  status: matchesStatus,
  category: (plugin, value) => matchesCategory(plugin, value),
  reader: (plugin, value) => matchesReader(plugin, value),
  capability: (plugin, value) => matchesCapability(plugin, value),
  feature: (plugin, value) => matchesFeature(plugin, value),
  id: (plugin, value) => plugin.id.toLowerCase().includes(value),
}

function parseToken(rawToken: string): FilterQualifier | string {
  const token = rawToken.toLowerCase()
  const tagAlias = TAG_ALIASES[token]
  if (tagAlias) return tagAlias

  const normalized = token.startsWith('@') ? token.slice(1) : token
  const colonIndex = normalized.indexOf(':')
  if (colonIndex > 0) {
    const key = normalized.slice(0, colonIndex)
    const value = normalized.slice(colonIndex + 1).replace(/^"|"$/g, '')
    const mapped = KEY_VALUE_ALIASES[key]?.[value]
    if (mapped) return mapped
    if (key in QUALIFIER_MATCHERS) {
      return { key, value }
    }
  }

  return token
}

/** Parses a search query into structured qualifiers and free-text terms. */
export function parsePluginQuery(rawQuery: string): ParsedPluginQuery {
  const cleanQuery = rawQuery.trim()
  if (!cleanQuery) return { qualifiers: [], terms: [] }

  const qualifiers: FilterQualifier[] = []
  const terms: string[] = []
  const tokenRegex = /(?:[^\s"]+|"[^"]*")+/g

  for (const match of cleanQuery.matchAll(tokenRegex)) {
    const parsed = parseToken(match[0])
    if (typeof parsed === 'string') {
      terms.push(parsed)
    } else {
      qualifiers.push(parsed)
    }
  }

  return { qualifiers, terms }
}

function matchesFreeText(plugin: ManagedPlugin, term: string): boolean {
  return (
    plugin.name.toLowerCase().includes(term) ||
    plugin.description.toLowerCase().includes(term) ||
    plugin.id.toLowerCase().includes(term) ||
    matchesCategory(plugin, term) ||
    plugin.feature.toLowerCase().includes(term) ||
    matchesCapability(plugin, term) ||
    matchesReader(plugin, term)
  )
}

/** Checks whether a plugin matches all qualifiers and free-text terms in the parsed query. */
export function matchesPluginQuery(
  plugin: ManagedPlugin,
  query: ParsedPluginQuery,
  context: PluginFilterContext,
): boolean {
  const matchesAllQualifiers = query.qualifiers.every((qualifier) => {
    const matcher = QUALIFIER_MATCHERS[qualifier.key]
    return matcher ? matcher(plugin, qualifier.value, context) : true
  })
  if (!matchesAllQualifiers) return false

  return query.terms.every((term) => matchesFreeText(plugin, term))
}

/** Checks whether the System settings row should be shown for the parsed query. */
export function matchesSystemRow(query: ParsedPluginQuery): boolean {
  if (query.qualifiers.length > 0) return false
  if (query.terms.length === 0) return true
  return query.terms.every((term) =>
    SYSTEM_KEYWORDS.some((keyword) => keyword.includes(term) || term.includes(keyword)),
  )
}

/** Orders categories by label with the catch-all last, then plugins by name. */
function compareItems(a: FlatPluginItem, b: FlatPluginItem): number {
  if (a.category !== b.category) {
    if (a.category === OTHER_CATEGORY) return 1
    if (b.category === OTHER_CATEGORY) return -1
    return a.categoryLabel.localeCompare(b.categoryLabel)
  }
  return a.plugin.name.localeCompare(b.plugin.name)
}

/** Filters, categorizes, and sorts plugins for the plugin list. */
export function getFilteredPluginItems(
  plugins: readonly ManagedPlugin[],
  query: ParsedPluginQuery,
  context: PluginFilterContext,
): readonly FlatPluginItem[] {
  const matching =
    query.qualifiers.length > 0 || query.terms.length > 0
      ? plugins.filter((plugin) => matchesPluginQuery(plugin, query, context))
      : plugins

  const items: FlatPluginItem[] = matching.map((plugin) => {
    const category = getPluginCategory(plugin)
    return { category, categoryLabel: formatCategoryLabel(category), plugin }
  })

  return items.sort(compareItems)
}
