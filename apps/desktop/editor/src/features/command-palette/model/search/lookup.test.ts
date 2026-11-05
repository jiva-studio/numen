/**
 * The target of a search hit, and where choosing one navigates the user.
 */
import { describe, expect, it } from 'vitest'
import { COMMAND } from '@/shared/commandIds'
import {
  createNameItem,
  createPassageItem,
  MENU_ACTION_PREFIX,
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
  noQuery: 'Search failed',
  wordsOnly: 'Words only',
  notEmbedded: 'Not read for meaning yet',
}

/** The first action of a note: the editor command that opens it. */
const OPEN_NOTE = `${MENU_ACTION_PREFIX}${COMMAND.openNote}`

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
  selector: null,
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
  offers: [OPEN_NOTE],
  type: 'note',
  selector: null,
  ...overrides,
})

describe('a note found by its title', () => {
  it('is placed under its path, offers opening it, and is a note', () => {
    const row = createNameItem(createName(), WORDS)

    expect(row.item.id).toBe('physics/Ontology.md')
    expect(row.item.actions).toStrictEqual([{ id: OPEN_NOTE, text: WORDS.read }])
    expect(row.hit).toStrictEqual(createHit({ line: -1 }))
  })
})

describe('a note found by name, heading or passage', () => {
  it('offers the note alone', () => {
    const read = [{ id: OPEN_NOTE, text: WORDS.read }]

    expect(createNameItem(createName(), WORDS).item.actions).toStrictEqual(read)
    expect(createNameItem(createName(), WORDS).hit.offers).toStrictEqual([OPEN_NOTE])
    expect(
      createNameItem(createName({ heading: 'Being', line: 7 }), WORDS).hit.offers,
    ).toStrictEqual([OPEN_NOTE])
    expect(createPassageItem('text', createPassage(), WORDS).item.actions).toStrictEqual(read)
  })
})

describe('a heading found by its name', () => {
  it('is placed under its path and line, offers the note at the heading, and is a note', () => {
    const row = createNameItem(createName({ heading: 'Being', line: 7, type: 'sample' }), WORDS)

    expect(row.item.id).toBe('physics/Ontology.md#7')
    expect(row.item.title).toBe('Being')
    expect(row.item.detail).toBe('Ontology')
    expect(row.item.actions).toStrictEqual([{ id: OPEN_NOTE, text: WORDS.readAt }])
    expect(row.hit).toStrictEqual(createHit({ line: 7, type: 'sample' }))
  })
})

describe('a passage found in a note', () => {
  it('offers the note, and remains at its line', () => {
    const row = createPassageItem('text', createPassage(), WORDS)

    expect(row.item.id).toBe('text:physics/Ontology.md:12')
    expect(row.item.actions).toStrictEqual([{ id: OPEN_NOTE, text: WORDS.read }])
    expect(row.hit).toStrictEqual(createHit({ line: 3, start: 12, length: 4 }))
  })
})

describe('a passage found in a file that is no note', () => {
  it('offers opening the file at its selector, remains at no line, and is of no note type', () => {
    const row = createPassageItem(
      'meaning',
      createPassage({ isNote: false, selector: { page: 3 }, title: '', type: 'sample' }),
      WORDS,
    )

    expect(row.item.title).toBe('physics/Ontology.md')
    expect(row.item.actions).toStrictEqual([{ id: 'note', text: WORDS.readDocument }])
    expect(row.hit).toStrictEqual(
      createHit({
        title: '',
        line: -1,
        start: 12,
        length: 4,
        offers: ['note'],
        type: null,
        selector: { page: 3 },
      }),
    )
  })
})

describe('where choosing an item takes the person', () => {
  it('is nowhere where nothing was found, or the item does not offer that', () => {
    expect(resolveDestination(undefined, 'note')).toBeNull()
    expect(resolveDestination(createHit({ offers: ['note'] }), 'other')).toBeNull()
  })

  it('is the file at the selector of the hit, for a file that is no note', () => {
    const searchHit = createHit({ type: null, offers: ['note'], selector: { t: [12, 14] } })

    expect(resolveDestination(searchHit, 'note')).toStrictEqual({
      at: 'file',
      path: 'physics/Ontology.md',
      title: 'Ontology',
      selector: { t: [12, 14] },
    })
  })

  it('is nowhere for a note, which the command that opens it takes to its line', () => {
    expect(resolveDestination(createHit({ line: 0 }), 'note')).toBeNull()
    expect(resolveDestination(createHit({ line: 5 }), 'note')).toBeNull()
    expect(resolveDestination(createHit({ line: 5 }), OPEN_NOTE)).toBeNull()
  })

  it('is the file with no line where a file that is no note remains at none', () => {
    const document = createHit({ type: null, offers: ['note'], line: -1 })

    expect(resolveDestination(document, 'note')).toStrictEqual({
      at: 'file',
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})
