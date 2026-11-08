import { describe, expect, it } from 'vitest'
import {
  assertPaletteStep,
  assertPlace,
  assertSingleTab,
  getStem,
  parseReference,
  parseTime,
  pickByTitle,
} from './rules.js'

describe('parseReference', () => {
  it('takes at and titled', () => {
    expect(parseReference('at', 'notes/A.md')).toEqual({ how: 'at', value: 'notes/A.md' })
    expect(parseReference('titled', 'A')).toEqual({ how: 'titled', value: 'A' })
  })

  it('throws for any other word', () => {
    expect(() => parseReference('named', 'A')).toThrow(
      '"named" does not name a tab or a note: use "at" or "titled"',
    )
  })
})

describe('assertPlace', () => {
  it('passes the place of each kind of tab', () => {
    expect(() => assertPlace('PDF Reader', 'page', 'a.pdf')).not.toThrow()
    expect(() => assertPlace('EPUB Reader', 'page', 'a.epub')).not.toThrow()
    expect(() => assertPlace('Audio Player', 'time', 'a.mp3')).not.toThrow()
  })

  it('throws for a place of the wrong kind', () => {
    expect(() => assertPlace('PDF Reader', 'time', 'a.pdf')).toThrow(
      'the tab "a.pdf" is a PDF Reader tab: its place is a page, not a time',
    )
    expect(() => assertPlace('Audio Player', 'page', 'a.mp3')).toThrow(
      'its place is a time, not a page',
    )
  })
})

describe('parseTime', () => {
  it('reads minutes and seconds', () => {
    expect(parseTime('2:00')).toBe(120)
    expect(parseTime('0:45')).toBe(45)
  })

  it('throws for text that is not a time', () => {
    expect(() => parseTime('2')).toThrow('"2" is not a time: write minutes and seconds, like "2:00"')
    expect(() => parseTime('2:75')).toThrow('is not a time')
  })
})

describe('assertPaletteStep', () => {
  it('passes the name', () => {
    expect(() => assertPaletteStep('name')).not.toThrow()
  })

  it('throws for a step that is not built', () => {
    expect(() => assertPaletteStep('vault')).toThrow('the palette step "vault" is not built yet')
  })
})

describe('pickByTitle', () => {
  it('finds the one path with the title', () => {
    expect(getStem('assets/shore.pdf')).toBe('shore')
    expect(pickByTitle('shore', ['assets/shore.pdf', 'assets/talk.mp3'])).toBe('assets/shore.pdf')
  })

  it('throws when no path has the title', () => {
    expect(() => pickByTitle('shore', ['assets/talk.mp3'])).toThrow(
      'no tab is titled "shore": use "at" with its path',
    )
  })

  it('throws when two paths have the title', () => {
    expect(() => pickByTitle('shore', ['a/shore.pdf', 'b/shore.md'])).toThrow(
      'more than one tab is titled "shore": use "at" with its path',
    )
  })
})

describe('assertSingleTab', () => {
  it('passes one tab and throws for two', () => {
    expect(() => assertSingleTab('Shopping list', 1)).not.toThrow()
    expect(() => assertSingleTab('Shopping list', 2)).toThrow(
      '2 tabs are titled "Shopping list": use "at" with the path',
    )
  })
})
