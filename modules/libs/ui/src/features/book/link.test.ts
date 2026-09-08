import { describe, expect, it } from 'vitest'
import { placeIn } from './link'

describe('where a link inside a book points', () => {
  it('names the document and the place in it', () => {
    expect(placeIn('OEBPS/first.xhtml#alpha')).toEqual({
      path: 'OEBPS/first.xhtml',
      fragment: 'alpha',
    })
  })

  it('names the document alone, where the link names no place in it', () => {
    expect(placeIn('OEBPS/first.xhtml')).toEqual({ path: 'OEBPS/first.xhtml', fragment: '' })
  })

  it('names no document, where the link points inside the one being read', () => {
    expect(placeIn('#alpha')).toEqual({ path: '', fragment: 'alpha' })
  })

  it('keeps the place whole, where the name a book gave it holds a hash', () => {
    expect(placeIn('OEBPS/first.xhtml#a#b')).toEqual({
      path: 'OEBPS/first.xhtml',
      fragment: 'a#b',
    })
  })
})
