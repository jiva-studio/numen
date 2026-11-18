import { describe, expect, it } from 'vitest'
import { createLogRow, filterRows, formatEntryProperties } from './format'

describe('createLogRow', () => {
  it('formats the UTC millisecond timestamp as HH:MM:SS.mmm and sorts property keys alphabetically', () => {
    const timeMs = Date.UTC(2026, 10, 18, 9, 5, 3, 42)
    const row = createLogRow(1, {
      time: timeMs,
      level: 'debug',
      source: 'sql',
      message: 'SQL query',
      properties: { sql: '"SELECT 1"', args: '[]', durationMs: '0.5' },
    })

    expect(row.timeText).toBe('09:05:03.042')
    expect(row.keys).toStrictEqual(['args', 'durationMs', 'sql'])
  })
})

describe('formatEntryProperties', () => {
  it('formats sql properties onto separate lines with uppercase keywords and indented AND/OR', () => {
    const rawSql =
      'select id, path, title from notes inner join links on notes.id = links.source_id where notes.vault_id = ? and notes.is_trashed = 0 order by notes.path limit 25'
    const formatted = formatEntryProperties({
      sql: JSON.stringify(rawSql),
    })

    expect(formatted).toStrictEqual([
      {
        key: 'sql',
        kind: 'sql',
        display: [
          'SELECT id, path, title',
          'FROM notes',
          'INNER JOIN links on notes.id = links.source_id',
          'WHERE notes.vault_id = ?',
          '  AND notes.is_trashed = 0',
          'ORDER BY notes.path',
          'LIMIT 25',
        ].join('\n'),
      },
    ])
  })

  it('formats JSON objects as indented JSON, scalars as text, and handles nested or invalid JSON', () => {
    const formatted = formatEntryProperties({
      request: '{"vaultId":"v1","path":"a.md"}',
      durationMs: '1.25',
      pool: '"read"',
      payload: '"{\\"nested\\":true}"',
      raw: 'not-valid-json',
    })

    expect(formatted).toStrictEqual([
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
