import { describe, expect, it } from 'vitest'
import { getSharedNotePath, parseReader } from './shared-reader.js'

describe('parseReader', () => {
  it('names the reader and the second reader', () => {
    expect(parseReader('the reader')).toBe('reader')
    expect(parseReader('the second reader')).toBe('second reader')
  })

  it('fails on a reader the scenarios do not know', () => {
    expect(() => parseReader('the third reader')).toThrow('does not name a reader')
  })
})

describe('getSharedNotePath', () => {
  it('adds the extension of a note to its title', () => {
    expect(getSharedNotePath('Shopping list')).toBe('Shopping list.md')
  })
})
