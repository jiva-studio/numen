/**
 * What a hit of the search stands for, and where choosing one takes the person.
 */
import { describe, expect, it } from 'vitest'
import {
  createNameItem,
  createPassageItem,
  resolveDestination,
  type NameMatch,
  type Passage,
  type SearchHit,
} from './lookup'
import type { Words } from './search'

const WORDS: Words = {
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  read: 'Open note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document',
  noneFound: 'Nothing',
  notAsked: 'Search failed',
  wordsOnly: 'Words only',
  notEmbedded: 'Not read for meaning yet',
}

/** The words of the command a plugin declares to put a note in front. */
const SHOW = 'Show in the graph'

const createName = (overrides: Partial<NameMatch> = {}): NameMatch => ({
  path: 'physics/Ontology.md',
  title: 'Ontology',
  heading: '',
  line: 0,
  at: [],
  type: 'note',
  ...overrides,
})

const createPassage = (overrides: Partial<Passage> = {}): Passage => ({
  path: 'physics/Ontology.md',
  title: 'Ontology',
  isNote: true,
  type: 'note',
  kind: 'note',
  text: 'what is',
  start: 12,
  length: 4,
  line: 3,
  at: [],
  ...overrides,
})

const createHit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
  path: 'physics/Ontology.md',
  title: 'Ontology',
  line: -1,
  start: 0,
  length: 0,
  offers: ['note', 'travel', 'document'],
  type: 'note',
  kind: 'note',
  ...overrides,
})

describe('a note found by its title', () => {
  it('is placed under its path, offers putting it in front first, and is a note', () => {
    const row = createNameItem(createName(), WORDS, SHOW)

    expect(row.item.id).toBe('physics/Ontology.md')
    expect(row.item.actions).toStrictEqual([
      { id: 'travel', text: SHOW },
      { id: 'note', text: WORDS.read },
    ])
    expect(row.hit).toStrictEqual(createHit({ line: -1, offers: ['travel', 'note'], kind: 'note' }))
  })
})

describe('a note found where no plugin puts a note in front', () => {
  it('offers the note alone, by name, heading and passage', () => {
    const read = [{ id: 'note', text: WORDS.read }]

    expect(createNameItem(createName(), WORDS, '').item.actions).toStrictEqual(read)
    expect(createNameItem(createName(), WORDS, '').hit.offers).toStrictEqual(['note'])
    expect(
      createNameItem(createName({ heading: 'Being', line: 7 }), WORDS, '').hit.offers,
    ).toStrictEqual(['note'])
    expect(createPassageItem('text', createPassage(), WORDS, '').item.actions).toStrictEqual(read)
  })
})

describe('a heading found by its name', () => {
  it('is placed under its path and line, offers the note first, and is a note', () => {
    const row = createNameItem(
      createName({ heading: 'Being', line: 7, type: 'sample' }),
      WORDS,
      SHOW,
    )

    expect(row.item.id).toBe('physics/Ontology.md#7')
    expect(row.item.title).toBe('Being')
    expect(row.item.detail).toBe('Ontology')
    expect(row.item.actions).toStrictEqual([
      { id: 'note', text: WORDS.readAt },
      { id: 'travel', text: SHOW },
    ])
    expect(row.hit).toStrictEqual(
      createHit({ line: 7, offers: ['note', 'travel'], type: 'sample', kind: 'note' }),
    )
  })
})

describe('a passage found in a note', () => {
  it('offers the note and putting it in front, and remains at its line', () => {
    const row = createPassageItem('text', createPassage(), WORDS, SHOW)

    expect(row.item.id).toBe('text:physics/Ontology.md:12')
    expect(row.item.actions).toStrictEqual([
      { id: 'note', text: WORDS.read },
      { id: 'travel', text: SHOW },
    ])
    expect(row.hit).toStrictEqual(
      createHit({ line: 3, start: 12, length: 4, offers: ['note', 'travel'] }),
    )
  })
})

describe('a passage found in a source that is not a note', () => {
  it('offers the document, remains at no line, and is of no note type', () => {
    const row = createPassageItem(
      'meaning',
      createPassage({ isNote: false, kind: 'book', title: '', type: 'sample' }),
      WORDS,
      SHOW,
    )

    expect(row.item.title).toBe('physics/Ontology.md')
    expect(row.item.actions).toStrictEqual([{ id: 'document', text: WORDS.readDocument }])
    expect(row.hit).toStrictEqual(
      createHit({
        title: '',
        line: -1,
        start: 12,
        length: 4,
        offers: ['document'],
        type: null,
        kind: 'book',
      }),
    )
  })
})

describe('where choosing an item takes the person', () => {
  it('is nowhere where nothing was found, or the item does not offer that', () => {
    expect(resolveDestination(undefined, 'note')).toBeNull()
    expect(resolveDestination(createHit({ offers: ['note'] }), 'travel')).toBeNull()
  })

  it('is the note put in front, by the note it remains in', () => {
    expect(resolveDestination(createHit(), 'travel')).toStrictEqual({
      at: 'travel',
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })

  it('is the span of the document it was found in', () => {
    expect(resolveDestination(createHit({ start: 12, length: 4 }), 'document')).toStrictEqual({
      at: 'document',
      path: 'physics/Ontology.md',
      title: 'Ontology',
      start: 12,
      length: 4,
    })
  })

  it('is the file at its line, the first line included', () => {
    expect(resolveDestination(createHit({ line: 0 }), 'note')).toStrictEqual({
      at: 'file',
      path: 'physics/Ontology.md',
      title: 'Ontology',
      line: 0,
    })
    expect(resolveDestination(createHit({ line: 5 }), 'note')).toMatchObject({ line: 5 })
  })

  it('is the file with no line where the item remains at none', () => {
    expect(resolveDestination(createHit({ line: -1 }), 'note')).toStrictEqual({
      at: 'file',
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})
