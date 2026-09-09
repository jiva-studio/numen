import { describe, expect, it } from 'vitest'
import { placeIn, pointsAway } from './link'

describe('whether a link leads out of the window', () => {
  it('leads out of it where the address names another place', () => {
    expect(pointsAway('https://example.invalid/away')).toBe(true)
  })

  it('leads nowhere outward, where the address is one the window serves', () => {
    expect(pointsAway('OEBPS/second.xhtml#alpha')).toBe(false)
    expect(pointsAway('#alpha')).toBe(false)
  })

  it('leads nowhere outward, where the href is no address at all', () => {
    // An href a book wrote badly is asked of the book itself, which is what
    // knows the place it meant.
    expect(pointsAway('::')).toBe(false)
  })
})

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
