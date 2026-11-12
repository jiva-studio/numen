/**
 * The target of a search hit, and where choosing one navigates the user.
 */
import { describe, expect, it } from 'vitest'
import { COMMAND, ROW } from '@/shared/model/command'
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

/** The first action of a note is the id of the editor command that opens it. */
const OPEN_NOTE = COMMAND.openFile

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

/** The fields of a hit, flat: where it is found, the ids of its actions, its type and selector. */
type HitFields = Partial<SearchHit['spot']> &
  Partial<Pick<SearchHit, 'type' | 'selector'>> & { actionIds?: readonly string[] }

const createHit = ({
  actionIds = [OPEN_NOTE],
  type = 'note',
  selector = null,
  ...spot
}: HitFields = {}): SearchHit => ({
  spot: { path: 'physics/Ontology.md', title: 'Ontology', line: -1, start: 0, length: 0, ...spot },
  actionIds,
  type,
  selector,
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
    expect(createNameItem(createName(), WORDS).hit.actionIds).toStrictEqual([OPEN_NOTE])
    expect(
      createNameItem(createName({ heading: 'Being', line: 7 }), WORDS).hit.actionIds,
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
    expect(row.item.actions).toStrictEqual([{ id: ROW.openDocument, text: WORDS.readDocument }])
    expect(row.hit).toStrictEqual(
      createHit({
        title: '',
        line: -1,
        start: 12,
        length: 4,
        actionIds: [ROW.openDocument],
        type: null,
        selector: { page: 3 },
      }),
    )
  })
})

describe('what choosing an action of an item runs', () => {
  it('is nothing where nothing was found, or the item does not carry that action', () => {
    expect(resolveChoice(undefined, ROW.openDocument)).toBeNull()
    expect(resolveChoice(createHit({ actionIds: [ROW.openDocument] }), 'other')).toBeNull()
  })

  it('is the command that opens the file at the selector of the hit, for a file that is no note', () => {
    const searchHit = createHit({
      type: null,
      actionIds: [ROW.openDocument],
      selector: { t: [12, 14] },
    })

    expect(resolveChoice(searchHit, ROW.openDocument)).toStrictEqual({
      command: 'app:file.open#document',
      over: {
        path: '',
        file: 'physics/Ontology.md',
        title: 'Ontology',
        selector: { t: [12, 14] },
      },
      input: '',
    })
  })

  it('is the command that opens the note at its line, for a note', () => {
    expect(resolveChoice(createHit({ line: 5 }), OPEN_NOTE)).toStrictEqual({
      command: 'app:file.open',
      over: {
        path: 'physics/Ontology.md',
        file: 'physics/Ontology.md',
        title: 'Ontology',
        line: 5,
      },
      input: '',
    })
    expect(resolveChoice(createHit({ line: 0 }), OPEN_NOTE)?.over?.line).toBe(0)
  })

  it('is the old action id of a document for nothing, since the action is a command id now', () => {
    expect(resolveChoice(createHit({ line: 5 }), 'note')).toBeNull()
    expect(resolveChoice(createHit({ type: null, actionIds: ['note'] }), 'note')).toBeNull()
  })

  it('is the file with no selector where a file that is no note remains at none', () => {
    const document = createHit({ type: null, actionIds: [ROW.openDocument], line: -1 })

    expect(resolveChoice(document, ROW.openDocument)?.over).toStrictEqual({
      path: '',
      file: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})
