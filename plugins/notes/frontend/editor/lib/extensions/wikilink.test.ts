import { describe, expect, it } from 'vitest'
import { findWikilink, getAddress, getWikilinks } from './wikilink'

describe('getAddress', () => {
  it('reads a name as the name scheme', () => {
    expect(getAddress('Thermodynamics')).toBe('name://Thermodynamics')
    expect(getAddress('Lecture 3: entropy')).toBe('name://Lecture 3: entropy')
    expect(getAddress('note:not-a-scheme')).toBe('name://note:not-a-scheme')
    expect(getAddress('://nothing-before-it')).toBe('name://://nothing-before-it')
  })

  it('reads a scheme before the colons', () => {
    expect(getAddress('note://01J8')).toBe('note://01J8')
  })

  it('leaves out the alias, the fragment and the spaces around the name', () => {
    expect(getAddress(' Entropy#Heat|the other way ')).toBe('name://Entropy')
  })

  it('is empty where only an alias, a fragment or a scheme was written', () => {
    expect(getAddress('|the other way')).toBe('')
    expect(getAddress('#Heading')).toBe('')
    expect(getAddress('note://')).toBe('')
  })
})

describe('getWikilinks', () => {
  it('finds the wikilinks of a text in the order they are written', () => {
    expect(
      getWikilinks('Under [[Thermodynamics]], beside [[Entropy]].').map((one) => one.address),
    ).toStrictEqual(['name://Thermodynamics', 'name://Entropy'])
  })

  it('finds no wikilink in single brackets, a Markdown link or empty brackets', () => {
    expect(getWikilinks('A single [bracket] and [Sundial](https://example.test).')).toStrictEqual(
      [],
    )
    expect(getWikilinks('[[]] and [[   ]] hold nothing.')).toStrictEqual([])
  })
})

describe('findWikilink', () => {
  it('finds the wikilink under a position, and nothing outside it', () => {
    const text = 'Under [[Thermodynamics]] it sits.'

    expect(findWikilink(text, 6)?.address).toBe('name://Thermodynamics')
    expect(findWikilink(text, 24)?.address).toBe('name://Thermodynamics')
    expect(findWikilink(text, 5)).toBeNull()
    expect(findWikilink(text, 25)).toBeNull()
  })
})
