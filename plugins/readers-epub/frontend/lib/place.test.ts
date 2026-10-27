import { describe, expect, it } from 'vitest'
import { createCfiSelector, getSelectorCfi } from './place'

describe('the place in a book', () => {
  it('is the CFI a selector made for it names', () => {
    expect(getSelectorCfi(createCfiSelector('epubcfi(/8/4!/4/2/1:12)'))).toBe(
      'epubcfi(/8/4!/4/2/1:12)',
    )
  })

  it('is nothing for a selector without a CFI', () => {
    expect(getSelectorCfi(null)).toBeNull()
    expect(getSelectorCfi({ page: 3 })).toBeNull()
    expect(getSelectorCfi({ cfi: '' })).toBeNull()
  })
})
