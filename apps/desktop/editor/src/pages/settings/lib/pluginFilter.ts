/**
 * Extensible query parser and filter engine for the settings plugin list.
 *
 * Supports composable `@tag` and `key:value` qualifiers alongside free-text
 * terms (for example `@disabled`, `disabled:true`, `reader:markdown`,
 * `feature:files`, `category:views`, `capability:agent`, `id:notes`).
 */
import type { PluginInfo } from '@/entities/settings'

export interface FilterQualifier {
  readonly key: string
  readonly value: string
}

export interface ParsedPluginQuery {
  readonly qualifiers: readonly FilterQualifier[]
  readonly terms: readonly string[]
}

export interface PluginFilterContext {
  readonly isEnabled: (pluginId: string) => boolean
  readonly getCategory: (plugin: PluginInfo) => string
  readonly getCategoryLabel: (category: string) => string
}

export interface FlatPluginItem {
  readonly category: string
  readonly categoryLabel: string
  readonly plugin: PluginInfo
}

const CATEGORY_ORDER = ['agents', 'editors', 'engines', 'readers', 'tools', 'views']

type QualifierMatcher = (
  plugin: PluginInfo,
  value: string,
  context: PluginFilterContext,
) => boolean

const TAG_ALIASES: Readonly<Record<string, FilterQualifier>> = {
  '@disabled': { key: 'status', value: 'disabled' },
  '@enabled': { key: 'status', value: 'enabled' },
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

function matchesReader(plugin: PluginInfo, value: string): boolean {
  if (!value) return (plugin.readers ?? []).length > 0
  const stems = toSearchStems(value)
  const hasReader = (plugin.readers ?? []).some((reader) => {
    const readerId = reader.id.toLowerCase()
    return (
      stems.some((stem) => readerId.includes(stem)) ||
      reader.extensions.some((ext) => stems.some((stem) => ext.toLowerCase().includes(stem)))
    )
  })
  if (hasReader) return true
  const id = plugin.id.toLowerCase()
  return stems.some((stem) => id.includes(stem))
}

function matchesCapability(plugin: PluginInfo, value: string): boolean {
  if (!value) return (plugin.capabilities ?? []).length > 0
  const stems = toSearchStems(value)
  return (plugin.capabilities ?? []).some((cap) =>
    stems.some((stem) => cap.toLowerCase().includes(stem)),
  )
}

function matchesFeature(plugin: PluginInfo, value: string): boolean {
  if (!value) return true
  const stems = toSearchStems(value)
  const id = plugin.id.toLowerCase()
  const name = plugin.name.toLowerCase()
  const description = (plugin.description ?? '').toLowerCase()
  const caps = (plugin.capabilities ?? []).map((cap) => cap.toLowerCase())
  return stems.some(
    (term) =>
      id.includes(term) ||
      name.includes(term) ||
      description.includes(term) ||
      caps.some((cap) => cap.includes(term)) ||
      matchesReader(plugin, term),
  )
}

const QUALIFIER_MATCHERS: Readonly<Record<string, QualifierMatcher>> = {
  status: (plugin, value, context) =>
    value === 'disabled' ? !context.isEnabled(plugin.id) : context.isEnabled(plugin.id),
  category: (plugin, value, context) => {
    const category = context.getCategory(plugin)
    const label = context.getCategoryLabel(category).toLowerCase()
    return category.includes(value) || label.includes(value)
  },
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

function matchesFreeText(
  plugin: PluginInfo,
  term: string,
  context: PluginFilterContext,
): boolean {
  const category = context.getCategory(plugin)
  const categoryLabel = context.getCategoryLabel(category).toLowerCase()
  return (
    plugin.name.toLowerCase().includes(term) ||
    (plugin.description ?? '').toLowerCase().includes(term) ||
    plugin.id.toLowerCase().includes(term) ||
    category.includes(term) ||
    categoryLabel.includes(term) ||
    matchesCapability(plugin, term) ||
    matchesReader(plugin, term)
  )
}

/** Checks whether a plugin matches all qualifiers and free-text terms in the parsed query. */
export function matchesPluginQuery(
  plugin: PluginInfo,
  query: ParsedPluginQuery,
  context: PluginFilterContext,
): boolean {
  const matchesAllQualifiers = query.qualifiers.every((qualifier) => {
    const matcher = QUALIFIER_MATCHERS[qualifier.key]
    return matcher ? matcher(plugin, qualifier.value, context) : true
  })
  if (!matchesAllQualifiers) return false

  return query.terms.every((term) => matchesFreeText(plugin, term, context))
}

/** Checks whether the System settings row should be shown for the parsed query. */
export function matchesSystemRow(query: ParsedPluginQuery): boolean {
  if (query.qualifiers.length > 0) return false
  if (query.terms.length === 0) return true
  return query.terms.every((term) =>
    SYSTEM_KEYWORDS.some((keyword) => keyword.includes(term) || term.includes(keyword)),
  )
}

/** Resolves the normalized category key for a plugin. */
export function getPluginCategory(plugin: PluginInfo): string {
  if ('category' in plugin && typeof plugin.category === 'string' && plugin.category.length > 0) {
    return plugin.category.toLowerCase()
  }
  const parts = plugin.id.split('.')
  if (parts.length >= 3 && parts[0] === 'md' && parts[1] === 'numen' && parts[2]) {
    return parts[2].toLowerCase()
  }
  if (parts.length >= 2 && parts[0]) {
    return parts[0].toLowerCase()
  }
  return 'other'
}

/** Formats a category key into its display label using localized category names. */
export function formatCategoryLabel(
  category: string,
  labels: Readonly<Record<string, string>> = {},
): string {
  return labels[category] ?? category.charAt(0).toUpperCase() + category.slice(1)
}

/** Filters, categorizes, and sorts plugins for the settings plugin list. */
export function getFilteredPluginItems(
  plugins: readonly PluginInfo[],
  query: ParsedPluginQuery,
  context: PluginFilterContext,
): readonly FlatPluginItem[] {
  const matching =
    query.qualifiers.length > 0 || query.terms.length > 0
      ? plugins.filter((plugin) => matchesPluginQuery(plugin, query, context))
      : plugins

  const items: FlatPluginItem[] = matching.map((plugin) => {
    const category = context.getCategory(plugin)
    return {
      category,
      categoryLabel: context.getCategoryLabel(category),
      plugin,
    }
  })

  return items.sort((a, b) => {
    const idxA = CATEGORY_ORDER.indexOf(a.category)
    const idxB = CATEGORY_ORDER.indexOf(b.category)
    if (idxA >= 0 && idxB >= 0 && idxA !== idxB) return idxA - idxB
    if (idxA >= 0 && idxB < 0) return -1
    if (idxB >= 0 && idxA < 0) return 1
    const catComp = a.category.localeCompare(b.category)
    if (catComp !== 0) return catComp
    return a.plugin.name.localeCompare(b.plugin.name)
  })
}
