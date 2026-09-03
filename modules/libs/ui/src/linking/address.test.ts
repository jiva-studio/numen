/**
 * A link to a note, read the same wherever it is written.
 *
 * The same brackets stand in an answer and in a note, and both are read here.
 */
import { describe, expect, it } from 'vitest'
import { addressOf, pointsAtNote, stated, wikilinkAt, wikilinksIn } from './address'

describe('an address', () => {
  it('is the name itself where no scheme is written', () => {
    expect(addressOf('Thermodynamics')).toStrictEqual({ scheme: 'name', value: 'Thermodynamics' })
  })

  it('is read out of the brackets it stands in', () => {
    expect(addressOf('[[Thermodynamics]]')).toStrictEqual({
      scheme: 'name',
      value: 'Thermodynamics',
    })
  })

  it('carries the scheme that was written', () => {
    expect(addressOf('note://01J8')).toStrictEqual({ scheme: 'note', value: '01J8' })
  })

  it('drops the alias and the fragment, which are not part of it', () => {
    expect(addressOf('notes/Entropy#Later|the other way')).toStrictEqual({
      scheme: 'name',
      value: 'notes/Entropy',
    })
  })

  it('reads a colon in a title as a colon', () => {
    expect(addressOf('Lecture 3: entropy')).toStrictEqual({
      scheme: 'name',
      value: 'Lecture 3: entropy',
    })
  })

  it('is one string, which is how it is handed on and read back', () => {
    expect(stated(addressOf('[[Entropy]]'))).toBe('name://Entropy')
    expect(addressOf(stated(addressOf('[[Entropy]]')))).toStrictEqual(addressOf('Entropy'))
  })
})

describe('what points at a note', () => {
  it('is a name and an identifier', () => {
    expect(pointsAtNote('name://Entropy')).toBe(true)
    expect(pointsAtNote('note://01J8')).toBe(true)
  })

  it('is not a place in a source, and not a page on the web', () => {
    expect(pointsAtNote('numen:book.pdf?start=1&length=2')).toBe(false)
    expect(pointsAtNote('https://example.com')).toBe(false)
  })
})

describe('the wikilinks in a text', () => {
  it('are found in the order they are written', () => {
    const found = wikilinksIn('Under [[Thermodynamics]], beside [[Entropy]].')

    expect(found.map((one) => one.address)).toStrictEqual([
      'name://Thermodynamics',
      'name://Entropy',
    ])
  })

  it('are read in the sentence by their alias, where one was written', () => {
    expect(wikilinksIn('[[Entropy|the other way]]')[0]?.text).toBe('the other way')
    expect(wikilinksIn('[[Entropy]]')[0]?.text).toBe('Entropy')
  })

  it('carry the name whole, dots, dashes and diacritics alike', () => {
    const name = 'Seminar 1.2–1.3 — Lisbon, 9 July 1973'

    expect(wikilinksIn(`Under [[${name}]].`).map((one) => one.address)).toStrictEqual([
      `name://${name}`,
    ])
  })

  it('are not brackets holding nothing', () => {
    expect(wikilinksIn('[[]] and [[ ]]')).toStrictEqual([])
  })

  it('are found under a position in the text, and nowhere else', () => {
    const text = 'Under [[Thermodynamics]] it sits.'

    expect(wikilinkAt(text, 10)?.address).toBe('name://Thermodynamics')
    expect(wikilinkAt(text, 2)).toBeNull()
    expect(wikilinkAt(text, 30)).toBeNull()
  })
})
