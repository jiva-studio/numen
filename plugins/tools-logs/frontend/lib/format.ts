/**
 * Pure formatting and filtering helpers for structured log entries: timestamp formatting, SQL
 * query pretty-printing, JSON property formatting, key extraction, and level/message filtering.
 */
import type { FormattedProperty, LevelFilter, LogEntry, LogRow } from '../types'

const pad2 = (n: number): string => String(n).padStart(2, '0')
const pad3 = (n: number): string => String(n).padStart(3, '0')

function formatTimestamp(timeMs: number): string {
  const date = new Date(timeMs)
  return `${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())}:${pad2(date.getUTCSeconds())}.${pad3(date.getUTCMilliseconds())}`
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

const MAJOR_SQL_PATTERN = new RegExp(
  `\\b(${MAJOR_SQL_CLAUSES.map((clause) => clause.replace(/ /g, '\\s+')).join('|')})\\b`,
  'gi',
)

const MINOR_SQL_PATTERN = new RegExp(`\\b(${MINOR_SQL_CLAUSES.join('|')})\\b`, 'gi')

function formatSql(rawSql: string): string {
  const collapsed = rawSql.replace(/\s+/g, ' ').trim()
  if (!collapsed) return ''
  const result = collapsed
    .replace(MAJOR_SQL_PATTERN, (match) => `\n${match.replace(/\s+/g, ' ').toUpperCase()}`)
    .replace(MINOR_SQL_PATTERN, (match) => `\n  ${match.toUpperCase()}`)
  return result
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.trim().length > 0)
    .join('\n')
}

function parseJsonString(
  raw: string,
):
  | { readonly ok: true; readonly parsed: string | number | boolean | object | null }
  | { readonly ok: false } {
  try {
    return { ok: true, parsed: JSON.parse(raw) as string | number | boolean | object | null }
  } catch {
    return { ok: false }
  }
}

function formatStringValue(key: string, text: string): FormattedProperty {
  if (key.toLowerCase() === 'sql') {
    return { key, kind: 'sql', display: formatSql(text) }
  }
  const nested = parseJsonString(text)
  if (nested.ok && typeof nested.parsed === 'object' && nested.parsed !== null) {
    return { key, kind: 'json', display: JSON.stringify(nested.parsed, null, 2) }
  }
  return { key, kind: 'scalar', display: text }
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
    return { key, kind: 'json', display: JSON.stringify(parsed, null, 2) }
  }
  return { key, kind: 'scalar', display: String(parsed) }
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
    keys: getPropertyKeys(entry.properties),
  }
}
