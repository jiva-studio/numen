/** The actions a search result actionIds: opening the note, then the entries plugins place in its menu. */
import { describe, expect, it } from 'vitest'
import { COMMAND } from '@/shared/commandIds'
import type { PlacedCommand } from '@numen/plugin-host'
import {
  createNameItem,
  createPassageItem,
  resolveChoice,
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
  creating: 'No results',
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
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

/** The first action of a note is the id of the editor command that opens it, and an entry is the id of its command. */
const openId = COMMAND.openNote
const showId = SHOW.command
const otherId = OTHER.command

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
      { id: openId, text: 'Open note' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
  })

  it('lets the hit accept the entries and opening the note', () => {
    const row = createNameItem(createName(), WORDS, ENTRIES)

    expect(row.hit.actionIds).toStrictEqual([openId, showId, otherId])
  })

  it('offers opening the note first whatever group the first entry is in', () => {
    const first = entry(`${ATLAS}:first.run`, 'First', { group: 'a', order: 0 })

    const row = createNameItem(createName(), WORDS, [first, SHOW])

    expect(row.item.actions?.[0]).toStrictEqual({ id: openId, text: 'Open note' })
    expect(row.item.actions?.[1]).toStrictEqual({
      id: first.command,
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
      { id: openId, text: 'Open note' },
    ])
  })
})

describe('a heading found by its name', () => {
  it('offers opening the note at the heading and then the entries', () => {
    const row = createNameItem(createName({ heading: 'Polyps', line: 7 }), WORDS, ENTRIES)

    expect(row.item.actions).toStrictEqual([
      { id: openId, text: 'Open at this heading' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
    expect(row.hit.actionIds).toStrictEqual([openId, showId, otherId])
  })
})

describe('a passage found in a note', () => {
  it('offers opening the note and then the entries', () => {
    const row = createPassageItem('text', createPassage(), WORDS, ENTRIES)

    expect(row.item.actions).toStrictEqual([
      { id: openId, text: 'Open note' },
      { id: showId, text: 'Show note in plex' },
      { id: otherId, text: 'Another' },
    ])
    expect(row.hit.actionIds).toStrictEqual([openId, showId, otherId])
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

    expect(row.item.actions).toStrictEqual([{ id: COMMAND.openFile, text: 'Open the document' }])
    expect(row.hit.actionIds).toStrictEqual([COMMAND.openFile])
  })
})

describe('the command an action of a result runs', () => {
  const createHit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
    path: 'reef/Coral.md',
    title: 'Coral',
    line: 3,
    start: 12,
    length: 4,
    actionIds: [openId, showId, otherId],
    type: 'sample',
    selector: null,
    ...overrides,
  })

  const createDocument = (overrides: Partial<SearchHit> = {}): SearchHit =>
    createHit({
      path: 'library/a.bin',
      title: '',
      line: -1,
      type: null,
      actionIds: [COMMAND.openFile],
      selector: { page: 3 },
      ...overrides,
    })

  it('is the command of the entry over the note of the hit', () => {
    expect(resolveChoice(createHit(), showId)).toStrictEqual({
      command: SHOW.command,
      over: { path: 'reef/Coral.md', file: 'reef/Coral.md', title: 'Coral', line: 3 },
      input: '',
    })
  })

  it('is the command that opens the note, over the note of the hit', () => {
    expect(resolveChoice(createHit(), openId)).toStrictEqual({
      command: COMMAND.openNote,
      over: { path: 'reef/Coral.md', file: 'reef/Coral.md', title: 'Coral', line: 3 },
      input: '',
    })
  })

  it('is the line of a heading or a passage for the note to open at, and no line for a name', () => {
    expect(resolveChoice(createHit({ line: 7 }), openId)?.over?.line).toBe(7)
    expect(resolveChoice(createHit({ line: 0 }), openId)?.over?.line).toBe(0)
    expect(resolveChoice(createHit({ line: -1 }), openId)?.over).not.toHaveProperty('line')
  })

  it('is the command that opens a document, over its file and the selector of the hit', () => {
    expect(resolveChoice(createDocument(), COMMAND.openFile)).toStrictEqual({
      command: 'app:file.open',
      over: { path: '', file: 'library/a.bin', title: '', selector: { page: 3 } },
      input: '',
    })
  })

  it('is the command that opens a document, over the file alone where the hit has no selector', () => {
    const place = resolveChoice(createDocument({ selector: null }), COMMAND.openFile)?.over

    expect(place).toStrictEqual({ path: '', file: 'library/a.bin', title: '' })
  })

  it('is none where nothing was found or the hit does not carry the action', () => {
    expect(resolveChoice(undefined, showId)).toBeNull()
    expect(resolveChoice(createHit({ actionIds: [openId] }), showId)).toBeNull()
    expect(resolveChoice(createDocument(), 'other')).toBeNull()
  })

  it('is none for an entry of the menu over a hit in a file that is no note', () => {
    expect(
      resolveChoice(createDocument({ actionIds: [COMMAND.openFile, showId] }), showId),
    ).toBeNull()
  })

  it('is none for the action id the document had before it was a command', () => {
    expect(resolveChoice(createDocument({ actionIds: ['note'] }), 'note')).toBeNull()
  })
})
