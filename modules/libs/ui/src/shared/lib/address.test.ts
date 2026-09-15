/**
 * A link to a note, read the same wherever it is written.
 *
 * The same brackets stand in an answer and in a note, and both are read here.
 */
import { describe, expect, it } from 'vitest'
import { addressOf, isNoteAddress, writeAddress, wikilinkAt, wikilinksIn } from './address'

/**
 * The addresses in a note travel on the wire as they were written, so the core
 * reads them too. Both read this one corpus, and neither owns it.
 */
import corpus from '../../../../protocol/testdata/addresses.json'

/**
 * The brackets are read here and again in the core, which finds the links a
 * note carries and the stencil a card is cut by. Both read this one corpus,
 * and neither owns it.
 */
import brackets from '../../../../protocol/testdata/wikilinks.json'

describe('an address', () => {
  it('is read the way the schema says it is', () => {
    expect(corpus.length).toBeGreaterThan(0)
    for (const { written, scheme, value } of corpus) {
      expect({ written, ...addressOf(written) }).toStrictEqual({ written, scheme, value })
    }
  })

  it('is one string, which is how it is handed on and read back', () => {
    expect(writeAddress(addressOf('[[Entropy]]'))).toBe('name://Entropy')
    expect(addressOf(writeAddress(addressOf('[[Entropy]]')))).toStrictEqual(addressOf('Entropy'))
  })
})

describe('what points at a note', () => {
  it('is a name and an identifier', () => {
    expect(isNoteAddress('name://Entropy')).toBe(true)
    expect(isNoteAddress('note://01J8')).toBe(true)
  })

  it('is not a place in a source, and not a page on the web', () => {
    expect(isNoteAddress('numen:book.pdf?start=1&length=2')).toBe(false)
    expect(isNoteAddress('https://example.com')).toBe(false)
  })
})

describe('the wikilinks in a text', () => {
  it('are the ones the schema says are there', () => {
    expect(brackets.length).toBeGreaterThan(0)
    for (const { text, found } of brackets) {
      expect({ text, found: wikilinksIn(text).map((one) => one.address) }).toStrictEqual({
        text,
        found,
      })
    }
  })

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
