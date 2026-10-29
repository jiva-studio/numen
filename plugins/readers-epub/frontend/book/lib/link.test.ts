import { describe, expect, it } from 'vitest'
import { isExternalHref, parseBookLink } from './link'

describe('whether a link leads out of the window', () => {
  it('leads out of it where the address names another place', () => {
    expect(isExternalHref('https://example.invalid/away')).toBe(true)
  })

  it('leads nowhere outward, where the address is one the window serves', () => {
    expect(isExternalHref('OEBPS/second.xhtml#alpha')).toBe(false)
    expect(isExternalHref('#alpha')).toBe(false)
  })

  it('leads nowhere outward, where the href is no address at all', () => {
    // An invalid href is resolved against the book itself, which knows the intended
    // location.
    expect(isExternalHref('::')).toBe(false)
  })
})

describe('where a link inside a book points', () => {
  it('names the document and the place in it', () => {
    expect(parseBookLink('OEBPS/first.xhtml#alpha')).toEqual({
      path: 'OEBPS/first.xhtml',
      fragment: 'alpha',
    })
  })

  it('names the document alone, where the link names no place in it', () => {
    expect(parseBookLink('OEBPS/first.xhtml')).toEqual({ path: 'OEBPS/first.xhtml', fragment: '' })
  })

  it('names no document, where the link points inside the one being read', () => {
    expect(parseBookLink('#alpha')).toEqual({ path: '', fragment: 'alpha' })
  })

  it('keeps the place whole, where the name a book gave it holds a hash', () => {
    expect(parseBookLink('OEBPS/first.xhtml#a#b')).toEqual({
      path: 'OEBPS/first.xhtml',
      fragment: 'a#b',
    })
  })
})
