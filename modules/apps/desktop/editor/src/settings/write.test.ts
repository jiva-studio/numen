/**
 * A setting written back into the file the person types.
 */
import { describe, expect, it } from 'vitest'
import { write } from './write'

describe('what is written back', () => {
  it('is JSON, laid out to be read', () => {
    expect(write({ model: 'opus' })).toBe('{\n  "model": "opus"\n}')
  })

  it('is read back as what was written', () => {
    expect(JSON.parse(write({ a: [1, 'two'], b: null }))).toStrictEqual({
      a: [1, 'two'],
      b: null,
    })
  })
})
