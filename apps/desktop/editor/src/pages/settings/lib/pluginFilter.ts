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

const FEATURE_ALIASES: Readonly<Record<string, readonly string[]>> = {
  files: ['file.tree', 'file-tree', 'files'],
  plex: ['plex'],
  agent: ['agent.chat', 'agent-chat', 'agent chat'],
  flashcards: ['flashcards'],
  cards: ['flashcards'],
  preset: ['flashcards'],
  recording: ['media.player', 'media-player', 'recording', 'audio'],
  audio: ['media.player', 'media-player', 'audio', 'recording'],
  note: ['notes', 'markdown'],
  markdown: ['notes', 'markdown'],
  document: ['pdf'],
  pdf: ['pdf'],
  book: ['epub'],
  epub: ['epub'],
}

const SYSTEM_KEYWORDS = ['system', 'theme', 'appearance', 'window', 'scale', 'naming', 'sync']

function matchesReader(plugin: PluginInfo, value: string): boolean {
  if (!value) return (plugin.readers ?? []).length > 0
  const hasReader = (plugin.readers ?? []).some(
    (reader) =>
      reader.id.toLowerCase().includes(value) ||
      reader.extensions.some((ext) => ext.toLowerCase().includes(value)),
  )
  if (hasReader) return true
  const aliases = FEATURE_ALIASES[value] ?? [value]
  const id = plugin.id.toLowerCase()
  return aliases.some((alias) => id.includes(alias))
}

function matchesCapability(plugin: PluginInfo, value: string): boolean {
  if (!value) return (plugin.capabilities ?? []).length > 0
  return (plugin.capabilities ?? []).some((cap) => cap.toLowerCase().includes(value))
}

function matchesFeature(plugin: PluginInfo, value: string): boolean {
  if (!value) return true
  const candidates = FEATURE_ALIASES[value] ?? [value]
  const id = plugin.id.toLowerCase()
  const name = plugin.name.toLowerCase()
  const caps = (plugin.capabilities ?? []).map((cap) => cap.toLowerCase())
  return candidates.some(
    (term) =>
      id.includes(term) ||
      name.includes(term) ||
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
