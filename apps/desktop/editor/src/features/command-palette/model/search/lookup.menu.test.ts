/** The actions a search result offers: opening the note, then the entries plugins place in its menu. */
import { describe, expect, it } from 'vitest'
import type { PlacedCommand } from '@numen/plugin-host'
import {
  createNameItem,
  createPassageItem,
  MENU_ACTION_PREFIX,
  resolveDestination,
  resolveMenuChoice,
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

const ATLAS = 'org.example.atlas'

const entry = (command: string, title: string, overrides: Partial<PlacedCommand> = {}) => ({
  pluginId: ATLAS,
  command,
  title,
  group: '',
  ...overrides,
})

const SHOW = entry(`${ATLAS}:graph.showNote`, 'Show note in plex', { group: 'navigate', order: 20 })
const OTHER = entry(`${ATLAS}:other.run`, 'Another')
const ENTRIES = [SHOW, OTHER]

const showId = `${MENU_ACTION_PREFIX}${SHOW.command}`
const otherId = `${MENU_ACTION_PREFIX}${OTHER.command}`

const createName = (overrides: Partial<NameMatch> = {}): NameMatch => ({
  path: 'reef/Coral.md',
  title: 'Coral',
  heading: '',
  line: 0,
  at: [],
  type: 'note',
  ...overrides,
})

const createPassage = (overrides: Partial<Passage> = {}): Passage => ({
  path: 'reef/Coral.md',
  title: 'Coral',
  isNote: true,
  type: 'note',
  selector: null,
  text: 'polyps build the reef',
  start: 12,
  length: 4,
  line: 3,
  at: [],
  ...overrides,
})

describe('a note found by its title', () => {
  it('offers opening the note and then each entry in the order it is given', () => {
    const row = createNameItem(createName(), WORDS, ENTRIES)

    expect(row.item.actions).toStrictEqual([
      { id: 'note', text: 'Open note' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
  })

  it('lets the hit accept the entries and opening the note', () => {
    const row = createNameItem(createName(), WORDS, ENTRIES)

    expect(row.hit.offers).toStrictEqual(['note', showId, otherId])
  })

  it('offers opening the note first whatever group the first entry is in', () => {
    const first = entry(`${ATLAS}:first.run`, 'First', { group: 'a', order: 0 })

    const row = createNameItem(createName(), WORDS, [first, SHOW])

    expect(row.item.actions?.[0]).toStrictEqual({ id: 'note', text: 'Open note' })
    expect(row.item.actions?.[1]).toStrictEqual({
      id: `${MENU_ACTION_PREFIX}${first.command}`,
      text: 'First',
    })
  })

  it('offers an entry as its second action', () => {
    const row = createNameItem(createName(), WORDS, [SHOW])

    expect(row.item.actions?.[1]).toStrictEqual({ id: showId, text: 'Show note in plex' })
    expect(row.item.actions).toHaveLength(2)
  })

  it('offers an entry whose command has no icon', () => {
    const iconless = entry(`${ATLAS}:iconless.run`, 'Iconless')

    const row = createNameItem(createName(), WORDS, [iconless])

    expect(row.item.actions?.map((one) => one.text)).toStrictEqual(['Open note', 'Iconless'])
  })

  it('offers opening the note alone where no plugin places an entry', () => {
    expect(createNameItem(createName(), WORDS, []).item.actions).toStrictEqual([
      { id: 'note', text: 'Open note' },
    ])
  })
})

describe('a heading found by its name', () => {
  it('offers opening the note at the heading and then the entries', () => {
    const row = createNameItem(createName({ heading: 'Polyps', line: 7 }), WORDS, ENTRIES)

    expect(row.item.actions).toStrictEqual([
      { id: 'note', text: 'Open at this heading' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
    expect(row.hit.offers).toStrictEqual(['note', showId, otherId])
  })
})

describe('a passage found in a note', () => {
  it('offers opening the note and then the entries', () => {
    const row = createPassageItem('text', createPassage(), WORDS, ENTRIES)

    expect(row.item.actions).toStrictEqual([
      { id: 'note', text: 'Open note' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
    expect(row.hit.offers).toStrictEqual(['note', showId, otherId])
  })
})

describe('a passage found in a file that is no note', () => {
  it('offers opening the document alone', () => {
    const row = createPassageItem(
      'meaning',
      createPassage({ isNote: false, selector: { page: 3 }, title: '', type: 'sample' }),
      WORDS,
      ENTRIES,
    )

    expect(row.item.actions).toStrictEqual([{ id: 'note', text: 'Open the document' }])
    expect(row.hit.offers).toStrictEqual(['note'])
  })
})

describe('the command an action of a result runs', () => {
  const createHit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
    path: 'reef/Coral.md',
    title: 'Coral',
    line: 3,
    start: 12,
    length: 4,
    offers: ['note', showId, otherId],
    type: 'sample',
    selector: null,
    ...overrides,
  })

  it('is the command of the entry over the note of the hit, with the type of the note', () => {
    expect(resolveMenuChoice(createHit(), showId)).toStrictEqual({
      command: SHOW.command,
      path: 'reef/Coral.md',
      title: 'Coral',
      type: 'sample',
    })
  })

  it('is none for the action that opens the note', () => {
    expect(resolveMenuChoice(createHit(), 'note')).toBeNull()
  })

  it('is none where nothing was found or the hit does not accept the action', () => {
    expect(resolveMenuChoice(undefined, showId)).toBeNull()
    expect(resolveMenuChoice(createHit({ offers: ['note'] }), showId)).toBeNull()
  })

  it('is none for a hit in a file that is no note', () => {
    expect(
      resolveMenuChoice(createHit({ type: null, offers: ['note', showId] }), showId),
    ).toBeNull()
  })

  it('leaves the note closed: an entry has no destination', () => {
    expect(resolveDestination(createHit(), showId)).toBeNull()
  })
})
