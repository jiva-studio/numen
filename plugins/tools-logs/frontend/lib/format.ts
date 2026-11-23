/**
 * Pure formatting and filtering helpers for structured log entries: timestamp formatting, SQL
 * query pretty-printing, JSON property formatting, key extraction, and level/message filtering.
 */
import type { JsonValue } from '@numen/plugin-sdk'
import type { FormattedProperty, LevelFilter, LogEntry, LogRow } from '../types'
import { tokenizeJson, tokenizeSql } from './tokenize'

const padToTwo = (n: number): string => String(n).padStart(2, '0')
const padToThree = (n: number): string => String(n).padStart(3, '0')

function formatTimestamp(timeMs: number): string {
  const date = new Date(timeMs)
  return `${padToTwo(date.getUTCHours())}:${padToTwo(date.getUTCMinutes())}:${padToTwo(date.getUTCSeconds())}.${padToThree(date.getUTCMilliseconds())}`
}

function getPropertyKeys(properties: Readonly<Record<string, string>>): readonly string[] {
  return Object.keys(properties).sort()
}

const MAJOR_SQL_CLAUSES = [
  'LEFT OUTER JOIN',
  'INNER JOIN',
  'LEFT JOIN',
  'RIGHT JOIN',
  'CROSS JOIN',
  'JOIN',
  'INSERT INTO',
  'DELETE FROM',
  'ORDER BY',
  'GROUP BY',
  'SELECT',
  'FROM',
  'WHERE',
  'VALUES',
  'UPDATE',
  'SET',
  'HAVING',
  'LIMIT',
  'OFFSET',
  'ON CONFLICT',
  'RETURNING',
]

const MINOR_SQL_CLAUSES = ['AND', 'OR']

const SQL_CLAUSE_TOKEN = new RegExp(
  `'[^']*'|"(?:""|[^"])*"|[()]|\\b(?:${[...MAJOR_SQL_CLAUSES, ...MINOR_SQL_CLAUSES]
    .map((clause) => clause.replace(/ /g, '\\s+'))
    .join('|')})\\b`,
  'gi',
)

function formatSqlStatement(statement: string): string {
  const collapsed = statement.replace(/\s+/g, ' ').trim()
  if (!collapsed) return ''
  let depth = 0
  const result = collapsed.replace(SQL_CLAUSE_TOKEN, (match) => {
    if (match === '(') {
      depth += 1
      return match
    }
    if (match === ')') {
      depth = Math.max(0, depth - 1)
      return match
    }
    if (match.startsWith("'") || match.startsWith('"')) {
      return match
    }
    const upper = match.replace(/\s+/g, ' ').toUpperCase()
    if (depth > 0) {
      return upper
    }
    if (MINOR_SQL_CLAUSES.includes(upper)) {
      return `\n  ${upper}`
    }
    return `\n${upper}`
  })
  return result
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.trim().length > 0)
    .join('\n')
}

function formatSql(rawSql: string): string {
  const bodyLines: string[] = []
  for (const line of rawSql.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('--')) continue
    bodyLines.push(trimmed)
  }
  return formatSqlStatement(bodyLines.join(' '))
}

function parseJsonString(
  raw: string,
): { readonly ok: true; readonly parsed: JsonValue } | { readonly ok: false } {
  try {
    return { ok: true, parsed: JSON.parse(raw) as JsonValue }
  } catch {
    return { ok: false }
  }
}

function createSqlProperty(key: string, rawSql: string): FormattedProperty {
  const display = formatSql(rawSql)
  return { key, kind: 'sql', display, tokens: tokenizeSql(display) }
}

function tryParseNestedJson(value: JsonValue): JsonValue {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (
      (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
      (trimmed.startsWith('[') && trimmed.endsWith(']'))
    ) {
      const nested = parseJsonString(trimmed)
      if (nested.ok && typeof nested.parsed === 'object' && nested.parsed !== null) {
        return unwrapNestedJson(nested.parsed)
      }
    }
    return value
  }
  return unwrapNestedJson(value)
}

function unwrapNestedJson(value: JsonValue): JsonValue {
  if (Array.isArray(value)) {
    return value.map((item) => tryParseNestedJson(item))
  }
  if (typeof value === 'object' && value !== null) {
    return Object.fromEntries(
      Object.entries(value).map(([k, v]) => [k, tryParseNestedJson(v as JsonValue)]),
    )
  }
  return value
}

function createJsonProperty(key: string, value: JsonValue): FormattedProperty {
  const unwrapped = unwrapNestedJson(value)
  const display = JSON.stringify(unwrapped, null, 2)
  return { key, kind: 'json', display, tokens: tokenizeJson(display) }
}

function createScalarProperty(key: string, display: string): FormattedProperty {
  return { key, kind: 'scalar', display, tokens: [] }
}

const TRUNCATED_SUFFIX = '... [truncated]'

function isTruncatedJson(text: string): boolean {
  const trimmed = text.trimStart()
  return (trimmed.startsWith('{') || trimmed.startsWith('[')) && trimmed.endsWith(TRUNCATED_SUFFIX)
}

function stepInString(
  ch: string,
  escaped: boolean,
): { readonly inString: boolean; readonly escaped: boolean } {
  if (escaped) return { inString: true, escaped: false }
  if (ch === '\\') return { inString: true, escaped: true }
  return { inString: ch !== '"', escaped: false }
}

function appendJsonStructureChar(out: string[], ch: string, indent: number): number {
  if (ch === '{' || ch === '[') {
    const next = indent + 1
    out.push(ch, '\n', '  '.repeat(next))
    return next
  }
  if (ch === '}' || ch === ']') {
    const next = Math.max(0, indent - 1)
    out.push('\n', '  '.repeat(next), ch)
    return next
  }
  if (ch === ',') {
    out.push(',\n', '  '.repeat(indent))
    return indent
  }
  if (ch === ':') {
    out.push(': ')
    return indent
  }
  if (!/\s/.test(ch)) {
    out.push(ch)
  }
  return indent
}

function formatTruncatedJson(raw: string): string {
  const out: string[] = []
  let indent = 0
  let inString = false
  let escaped = false
  for (const ch of raw) {
    if (inString) {
      out.push(ch)
      const next = stepInString(ch, escaped)
      inString = next.inString
      escaped = next.escaped
      continue
    }
    if (ch === '"') {
      inString = true
      out.push(ch)
      continue
    }
    indent = appendJsonStructureChar(out, ch, indent)
  }
  return out.join('')
}

function createTruncatedJsonProperty(key: string, raw: string): FormattedProperty {
  const display = formatTruncatedJson(raw)
  return { key, kind: 'json', display, tokens: tokenizeJson(display) }
}

function formatStringValue(key: string, text: string): FormattedProperty {
  if (key.toLowerCase() === 'sql') {
    return createSqlProperty(key, text)
  }
  const nested = parseJsonString(text)
  if (nested.ok && typeof nested.parsed === 'object' && nested.parsed !== null) {
    return createJsonProperty(key, nested.parsed)
  }
  if (isTruncatedJson(text)) {
    return createTruncatedJsonProperty(key, text)
  }
  return createScalarProperty(key, text)
}

function formatProperty(key: string, rawJson: string): FormattedProperty {
  const parsedResult = parseJsonString(rawJson)
  if (!parsedResult.ok) {
    return formatStringValue(key, rawJson)
  }
  const { parsed } = parsedResult
  if (typeof parsed === 'string') {
    return formatStringValue(key, parsed)
  }
  if (typeof parsed === 'object' && parsed !== null) {
    return createJsonProperty(key, parsed)
  }
  return createScalarProperty(key, String(parsed))
}

/** Formats all properties of an entry in alphabetical key order. */
export function formatEntryProperties(
  properties: Readonly<Record<string, string>>,
): readonly FormattedProperty[] {
  return getPropertyKeys(properties).map((key) => formatProperty(key, properties[key] ?? ''))
}

/** Filters log rows by selected level (`'all'` or an exact level) and case-insensitive message query. */
export function filterRows(
  rows: readonly LogRow[],
  levelFilter: LevelFilter,
  searchQuery: string,
): readonly LogRow[] {
  const needle = searchQuery.trim().toLowerCase()
  return rows.filter((row) => {
    if (levelFilter !== 'all' && row.entry.level !== levelFilter) {
      return false
    }
    if (!needle) {
      return true
    }
    return (
      row.entry.message.toLowerCase().includes(needle) ||
      row.entry.source.toLowerCase().includes(needle)
    )
  })
}

/** Wraps a LogEntry in a LogRow with a sequential identifier. */
export function createLogRow(id: number, entry: LogEntry): LogRow {
  return {
    id,
    entry,
    timeText: formatTimestamp(entry.time),
  }
}
