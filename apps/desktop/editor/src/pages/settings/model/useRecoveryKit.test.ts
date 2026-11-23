import { describe, expect, it } from 'vitest'
import { splitSheetText } from './useRecoveryKit'

describe('splitSheetText', () => {
  it('splits a sheet on any whitespace', () => {
    expect(splitSheetText('academic acid\tacne\nacquire\r\nactress   adapt')).toStrictEqual([
      'academic',
      'acid',
      'acne',
      'acquire',
      'actress',
      'adapt',
    ])
  })

  it('trims the text and lower-cases each word', () => {
    expect(splitSheetText('  Academic ACID  \n')).toStrictEqual(['academic', 'acid'])
  })

  it('gives no word for an empty text or for blanks', () => {
    expect(splitSheetText('')).toStrictEqual([])
    expect(splitSheetText(' \n\t ')).toStrictEqual([])
  })
})
