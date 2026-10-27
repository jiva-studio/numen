import { describe, expect, it } from 'vitest'
import { createCfiSelector, findAnchorCfi, findAnchorOffset, getSelectorCfi } from './place'

const anchors = [
  { offset: 0, cfi: 'epubcfi(/6/2!/4)' },
  { offset: 0, cfi: 'epubcfi(/6/2!/4/2)' },
  { offset: 12, cfi: 'epubcfi(/6/2!/4/4)' },
  { offset: 40, cfi: 'epubcfi(/6/4!/4/2)' },
]

describe('the place in a book', () => {
  it('is the CFI a selector made for it names', () => {
    expect(getSelectorCfi(createCfiSelector('epubcfi(/6/4!/4/2)'))).toBe('epubcfi(/6/4!/4/2)')
  })

  it('is nothing for a selector without a CFI or with one that is no CFI', () => {
    expect(getSelectorCfi(null)).toBeNull()
    expect(getSelectorCfi({ page: 3 })).toBeNull()
    expect(getSelectorCfi({ cfi: '/6/4' })).toBeNull()
  })

  it('finds the offset of the innermost block a CFI names a place inside', () => {
    expect(findAnchorOffset(anchors, 'epubcfi(/6/2!/4/4)')).toBe(12)
    expect(findAnchorOffset(anchors, 'epubcfi(/6/2!/4/4/1:7)')).toBe(12)
    expect(findAnchorOffset(anchors, 'epubcfi(/6/8!/4/2)')).toBeNull()
  })

  it('finds the CFI of the block an offset falls in', () => {
    expect(findAnchorCfi(anchors, 20)).toBe('epubcfi(/6/2!/4/4)')
    expect(findAnchorCfi(anchors, 40)).toBe('epubcfi(/6/4!/4/2)')
    expect(findAnchorCfi([], 3)).toBeNull()
  })
})
