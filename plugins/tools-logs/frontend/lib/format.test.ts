import { describe, expect, it } from 'vitest'
import { createLogRow, filterRows, formatEntryProperties } from './format'

describe('createLogRow', () => {
  it('formats the UTC millisecond timestamp as HH:MM:SS.mmm', () => {
    const timeMs = Date.UTC(2026, 10, 18, 9, 5, 3, 42)
    const row = createLogRow(1, {
      time: timeMs,
      level: 'debug',
      source: 'sql',
      message: 'SQL query',
      properties: { sql: '"SELECT 1"', args: '[]', durationMs: '0.5' },
    })

    expect(row.timeText).toBe('09:05:03.042')
  })
})

describe('formatEntryProperties', () => {
  it('formats sql properties onto separate lines with uppercase keywords, indented AND/OR, and syntax tokens', () => {
    const rawSql =
      'select id, path, title from notes inner join links on notes.id = links.source_id where notes.vault_id = ? and notes.is_trashed = 0 order by notes.path limit 25'
    const formatted = formatEntryProperties({
      sql: JSON.stringify(rawSql),
    })

    expect(formatted).toHaveLength(1)
    expect(formatted[0]?.key).toBe('sql')
    expect(formatted[0]?.kind).toBe('sql')
    expect(formatted[0]?.display).toBe(
      [
        'SELECT id, path, title',
        'FROM notes',
        'INNER JOIN links on notes.id = links.source_id',
        'WHERE notes.vault_id = ?',
        '  AND notes.is_trashed = 0',
        'ORDER BY notes.path',
        'LIMIT 25',
      ].join('\n'),
    )
    expect(formatted[0]?.tokens.map((t) => t.text).join('')).toBe(formatted[0]?.display)
    expect(formatted[0]?.tokens.some((t) => t.kind === 'keyword' && t.text === 'SELECT')).toBe(true)
    expect(formatted[0]?.tokens.some((t) => t.kind === 'number' && t.text === '25')).toBe(true)
  })

  it('strips SQL line comments and formats only top-level clauses onto separate lines', () => {
    const rawSql = [
      '-- What the notes at these paths are.',
      '--',
      '-- The paths arrive as a JSON array, so a folder is one question whatever it',
      '-- holds. A path the index holds no note at is absent from the answer.',
      'SELECT s.path, n.type',
      'FROM sources s',
      'JOIN notes n ON n.source_id = s.id',
      'WHERE s.vault_id = ? AND s.path IN (SELECT value FROM json_each(?));',
    ].join('\n')
    const formatted = formatEntryProperties({
      sql: JSON.stringify(rawSql),
    })

    expect(formatted[0]?.display).toBe(
      [
        'SELECT s.path, n.type',
        'FROM sources s',
        'JOIN notes n ON n.source_id = s.id',
        'WHERE s.vault_id = ?',
        '  AND s.path IN (SELECT value FROM json_each(?));',
      ].join('\n'),
    )
    expect(formatted[0]?.tokens.some((t) => t.kind === 'keyword' && t.text === 'SELECT')).toBe(true)
  })

  it('unwraps JSON-encoded strings nested inside JSON arrays or objects', () => {
    const formatted = formatEntryProperties({
      args: JSON.stringify([1, '["a.md","b.md"]']),
    })

    expect(formatted[0]?.kind).toBe('json')
    expect(formatted[0]?.display).toBe('[\n  1,\n  [\n    "a.md",\n    "b.md"\n  ]\n]')
  })

  it('formats JSON objects as indented syntax-highlighted JSON, scalars as text, and handles nested or invalid JSON', () => {
    const formatted = formatEntryProperties({
      request: '{"vaultId":"v1","path":"a.md"}',
      durationMs: '1.25',
      pool: '"read"',
      payload: '"{\\"nested\\":true}"',
      raw: 'not-valid-json',
    })

    expect(
      formatted.map(({ key, kind, display }) => ({
        key,
        kind,
        display,
      })),
    ).toStrictEqual([
      { key: 'durationMs', kind: 'scalar', display: '1.25' },
      {
        key: 'payload',
        kind: 'json',
        display: '{\n  "nested": true\n}',
      },
      { key: 'pool', kind: 'scalar', display: 'read' },
      { key: 'raw', kind: 'scalar', display: 'not-valid-json' },
      {
        key: 'request',
        kind: 'json',
        display: '{\n  "vaultId": "v1",\n  "path": "a.md"\n}',
      },
    ])
    const requestProp = formatted.find((p) => p.key === 'request')
    expect(requestProp?.tokens.map((t) => t.text).join('')).toBe(requestProp?.display)
    expect(requestProp?.tokens.some((t) => t.kind === 'key' && t.text === '"vaultId"')).toBe(true)
    expect(requestProp?.tokens.some((t) => t.kind === 'string' && t.text === '"v1"')).toBe(true)
  })

  it('pretty-prints and syntax-highlights truncated JSON payloads', () => {
    const formatted = formatEntryProperties({
      request: JSON.stringify('{"tabs":[{"id":"logs:logs"... [truncated]'),
    })
    expect(formatted).toHaveLength(1)
    expect(formatted[0]?.kind).toBe('json')
    expect(formatted[0]?.display).toContain('{\n  "tabs": [\n    {\n      "id": "logs:logs"')
    expect(formatted[0]?.tokens.some((t) => t.kind === 'key' && t.text === '"tabs"')).toBe(true)
  })
})

describe('filterRows', () => {
  const rows = [
    createLogRow(1, {
      time: 1000,
      level: 'debug',
      source: 'sql',
      message: 'SQL query',
      properties: { sql: '"SELECT 1"' },
    }),
    createLogRow(2, {
      time: 2000,
      level: 'info',
      source: 'rpc',
      message: 'NoteService/CreateNote',
      properties: {},
    }),
    createLogRow(3, {
      time: 3000,
      level: 'error',
      source: 'md.numen.notes',
      message: 'Failed to save note',
      properties: {},
    }),
  ]

  it('filters by level and case-insensitive message or source text', () => {
    expect(filterRows(rows, 'all', '').map((r) => r.id)).toStrictEqual([1, 2, 3])
    expect(filterRows(rows, 'error', '').map((r) => r.id)).toStrictEqual([3])
    expect(filterRows(rows, 'all', 'note').map((r) => r.id)).toStrictEqual([2, 3])
    expect(filterRows(rows, 'debug', 'sql').map((r) => r.id)).toStrictEqual([1])
  })
})
