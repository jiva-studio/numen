import { describe, expect, it } from 'vitest'
import { readConnectionRows } from './connectionRows'

describe('readConnectionRows', () => {
  it('returns the rows of the setting in the order of use', () => {
    const rows = readConnectionRows([
      { connection: 'c1f0a9e20', model: 'google/gemini-2.5-flash', reasoning: 'low' },
      { connection: 'c77b01d40', model: 'qwen2.5:14b' },
    ])

    expect(rows).toEqual({
      ok: true,
      value: [
        { connection: 'c1f0a9e20', model: 'google/gemini-2.5-flash', reasoning: 'low' },
        { connection: 'c77b01d40', model: 'qwen2.5:14b', reasoning: '' },
      ],
    })
  })

  it('returns no row for a setting that was not delivered', () => {
    expect(readConnectionRows(undefined)).toEqual({ ok: true, value: [] })
  })

  it('returns no row for an empty list', () => {
    expect(readConnectionRows([])).toEqual({ ok: true, value: [] })
  })

  it.each([
    ['a text', 'c1f0a9e20'],
    ['an object', { connection: 'c1f0a9e20', model: 'm' }],
    ['a row that is a text', ['c1f0a9e20']],
    ['a row with no model', [{ connection: 'c1f0a9e20' }]],
    ['a row with a number as its connection', [{ connection: 7, model: 'm' }]],
  ])('refuses %s', (_, value) => {
    expect(readConnectionRows(value)).toEqual({ ok: false, error: 'invalid_rows' })
  })
})
