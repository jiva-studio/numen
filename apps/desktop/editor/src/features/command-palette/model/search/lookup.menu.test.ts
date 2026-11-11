/** The actions a search result actionIds: opening the note, then the entries plugins place in its menu. */
import { describe, expect, it } from 'vitest'
import { COMMAND, ROW } from '@/shared/model/command'
import type { SlotCommand } from '@numen/plugin-loader'
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
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
}

const ATLAS = 'org.example.atlas'

const entry = (
  command: string,
  title: string,
  overrides: Partial<SlotCommand> = {},
): SlotCommand => ({
  kind: 'command',
  key: `${ATLAS}/command/${command}`,
  pluginId: ATLAS,
  command,
  title,
  shortcut: '',
  hasItems: false,
  ...overrides,
})

const SHOW = entry(`${ATLAS}:graph.showNote`, 'Show note in plex')
const OTHER = entry(`${ATLAS}:other.run`, 'Another')
const ENTRIES = [SHOW, OTHER]

/** The first action of a note is the id of the editor command that opens it, and an entry is the id of its command. */
const openId = COMMAND.openFile
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
    const first = entry(`${ATLAS}:first.run`, 'First')

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

    expect(row.item.actions).toStrictEqual([{ id: ROW.openDocument, text: 'Open the document' }])
    expect(row.hit.actionIds).toStrictEqual([ROW.openDocument])
  })
})

describe('the command an action of a result runs', () => {
  /** The fields of a hit, flat: where it is found, the ids of its actions, its type and selector. */
  type HitFields = Partial<SearchHit['spot']> &
    Partial<Pick<SearchHit, 'type' | 'selector'>> & { actionIds?: readonly string[] }

  const createHit = ({
    actionIds = [openId, showId, otherId],
    type = 'sample',
    selector = null,
    ...spot
  }: HitFields = {}): SearchHit => ({
    spot: { path: 'reef/Coral.md', title: 'Coral', line: 3, start: 12, length: 4, ...spot },
    actionIds,
    type,
    selector,
  })

  const createDocument = (overrides: HitFields = {}): SearchHit =>
    createHit({
      path: 'library/a.bin',
      title: '',
      line: -1,
      type: null,
      actionIds: [ROW.openDocument],
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
      command: COMMAND.openFile,
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
    expect(resolveChoice(createDocument(), ROW.openDocument)).toStrictEqual({
      command: 'app:file.open#document',
      over: { path: '', file: 'library/a.bin', title: '', selector: { page: 3 } },
      input: '',
    })
  })

  it('is the command that opens a document, over the file alone where the hit has no selector', () => {
    const place = resolveChoice(createDocument({ selector: null }), ROW.openDocument)?.over

    expect(place).toStrictEqual({ path: '', file: 'library/a.bin', title: '' })
  })

  it('is none where nothing was found or the hit does not carry the action', () => {
    expect(resolveChoice(undefined, showId)).toBeNull()
    expect(resolveChoice(createHit({ actionIds: [openId] }), showId)).toBeNull()
    expect(resolveChoice(createDocument(), 'other')).toBeNull()
  })

  it('is none for an entry of the menu over a hit in a file that is no note', () => {
    expect(
      resolveChoice(createDocument({ actionIds: [ROW.openDocument, showId] }), showId),
    ).toBeNull()
  })

  it('is none for the action id the document had before it was a command', () => {
    expect(resolveChoice(createDocument({ actionIds: ['note'] }), 'note')).toBeNull()
  })
})

describe('an entry that holds values', () => {
  const WITH_VALUES = entry(`${ATLAS}:graph.show`, 'Show', { values: { where: 'plex' } })

  const getActionId = (row: ReturnType<typeof createNameItem>, text: string): string =>
    row.item.actions?.find((one) => one.text === text)?.id ?? ''

  it('gives the values to the choice of its action, and none to the choice of another', () => {
    const row = createNameItem(createName(), WORDS, [WITH_VALUES, OTHER])

    const choice = resolveChoice(row.hit, getActionId(row, 'Show'))

    expect(choice?.command).toBe(WITH_VALUES.command)
    expect(choice?.values).toStrictEqual({ where: 'plex' })
    expect(resolveChoice(row.hit, OTHER.command)).not.toHaveProperty('values')
  })

  it('keeps the plain open of a note free of the values of an entry for the same command', () => {
    const splitEntry = entry(COMMAND.openFile, 'Open in split', { values: { placement: 'split' } })
    const row = createNameItem(createName(), WORDS, [splitEntry])

    expect(resolveChoice(row.hit, COMMAND.openFile)).not.toHaveProperty('values')
    expect(resolveChoice(row.hit, getActionId(row, 'Open in split'))).toMatchObject({
      command: COMMAND.openFile,
      values: { placement: 'split' },
    })
  })

  it('gives the values to the choice on a passage of a note', () => {
    const row = createPassageItem('text', createPassage(), WORDS, [WITH_VALUES])

    expect(resolveChoice(row.hit, getActionId(row, 'Show'))?.values).toStrictEqual({
      where: 'plex',
    })
  })

  it('leaves the hit without values where no entry holds any', () => {
    const row = createNameItem(createName(), WORDS, ENTRIES)

    expect(row.hit).not.toHaveProperty('actionValues')
  })
})
