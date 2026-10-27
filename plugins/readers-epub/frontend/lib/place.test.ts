import { describe, expect, it } from 'vitest'
import { createOffsetSelector, getSelectorOffset } from './place'

describe('the place in a book', () => {
  it('is the offset a selector made for it names', () => {
    expect(getSelectorOffset(createOffsetSelector(1204))).toBe(1204)
  })

  it('is nothing for a selector without an offset or with one that is no byte offset', () => {
    expect(getSelectorOffset(null)).toBeNull()
    expect(getSelectorOffset({ page: 3 })).toBeNull()
    expect(getSelectorOffset(createOffsetSelector(-1))).toBeNull()
    expect(getSelectorOffset({ 'md.numen.readers.epub/offset': '12' })).toBeNull()
  })
})
