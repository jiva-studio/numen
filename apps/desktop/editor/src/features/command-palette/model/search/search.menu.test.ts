/** The menu of a search result as the palette draws it and answers a choice from it. */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { SearchMenuEntry } from './lookup'
import { COMMAND } from '@/shared/commandIds'
import { CREATE_NOTE_ROW } from '../../lib/entries/entries'
import type { CommandTarget } from '../../types'
import { useSearch, type NameMatch, type Passage, type SearchDeps, type Words } from './search'

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
const SHOW: SearchMenuEntry = {
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
}
const OTHER: SearchMenuEntry = {
  command: `${ATLAS}:other.run`,
  title: 'Another',
}
const openId = COMMAND.openNote
const showId = SHOW.command

const nameMatch: NameMatch = {
  path: 'reef/Coral.md',
  title: 'Coral',
  heading: '',
  line: -1,
  at: [{ start: 0, end: 3 }],
  type: 'sample',
}

const bookPassage: Passage = {
  path: 'books/Atlas.pdf',
  title: '',
  text: 'a map of the reef',
  isNote: false,
  type: 'note',
  selector: { page: 3 },
  start: 0,
  length: 0,
  line: 0,
  at: [],
}

const createCore = (names: readonly NameMatch[], texts: readonly Passage[] = []): SearchDeps => ({
  names: async () => names,
  search: async (_query, mode) => (mode === 'words' ? texts : []),
})

const flushPromises = async () => {
  for (let turn = 0; turn < 6; turn += 1) await Promise.resolve()
}

const now = () => Promise.resolve()

async function typeQuery(palette: ReturnType<typeof useSearch>) {
  await palette.setTyped('cor')
  await flushPromises()
}

const getNamesGroup = (palette: ReturnType<typeof useSearch>) =>
  palette.groups.value.find((one) => one.id === 'names')

describe('a note found by name in the search', () => {
  it('offers opening the note and then the entries of the menu in the order they are given', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW, OTHER] },
    })

    await typeQuery(palette)

    expect(getNamesGroup(palette)?.items[0]?.actions).toStrictEqual([
      { id: openId, text: 'Open note' },
      { id: showId, text: 'Show note in plex' },
      { id: OTHER.command, text: 'Another' },
    ])
  })

  it('offers opening the note alone where the search has no menu', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, { wait: now })

    await typeQuery(palette)

    expect(getNamesGroup(palette)?.items[0]?.actions).toStrictEqual([
      { id: openId, text: 'Open note' },
    ])
  })

  it('offers opening the note alone once the plugin that placed the entry is disabled', async () => {
    const entries = ref<readonly SearchMenuEntry[]>([SHOW])
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => entries.value },
    })
    await typeQuery(palette)
    expect(getNamesGroup(palette)?.items[0]?.actions).toHaveLength(2)

    entries.value = []

    expect(getNamesGroup(palette)?.items[0]?.actions).toStrictEqual([
      { id: openId, text: 'Open note' },
    ])
  })

  it('answers the choice of an entry with its command over the note of the result', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })
    await typeQuery(palette)
    const item = getNamesGroup(palette)?.items[0]?.id ?? ''

    expect(palette.chooseResult(item, showId)).toStrictEqual({
      command: SHOW.command,
      over: { path: 'reef/Coral.md', file: 'reef/Coral.md', title: 'Coral' },
      input: '',
    })
  })

  it('runs the command that opens the note over the result on the choice of the first action', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })
    await typeQuery(palette)
    const item = getNamesGroup(palette)?.items[0]?.id ?? ''

    expect(palette.chooseResult(item, openId)).toStrictEqual({
      command: COMMAND.openNote,
      over: { path: 'reef/Coral.md', file: 'reef/Coral.md', title: 'Coral' },
      input: '',
    })
    expect(palette.chooseResult(item, 'note')).toBeNull()
  })

  it('answers an action the row does not carry with nothing', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })
    await typeQuery(palette)
    const item = getNamesGroup(palette)?.items[0]?.id ?? ''

    expect(palette.chooseResult(item, OTHER.command)).toBeNull()
    expect(palette.chooseResult('no such row', showId)).toBeNull()
  })
})

describe('a passage of a file that is no note in the search', () => {
  it('offers opening the document and no entry of the menu', async () => {
    const palette = useSearch(createCore([], [bookPassage]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })

    await typeQuery(palette)

    const text = palette.groups.value.find((one) => one.id === 'text')
    expect(text?.items[0]?.actions).toStrictEqual([
      { id: COMMAND.openFile, text: 'Open the document' },
    ])
    expect(palette.chooseResult(text?.items[0]?.id ?? '', showId)).toBeNull()
    expect(palette.chooseResult(text?.items[0]?.id ?? '', 'note')).toBeNull()
  })
})

describe('choosing the action of a passage of a file that is no note', () => {
  it('runs the command that opens the document over its file and the selector of the passage', async () => {
    const palette = useSearch(createCore([], [bookPassage]), WORDS, { wait: now })
    await typeQuery(palette)
    const item = palette.groups.value.find((one) => one.id === 'text')?.items[0]?.id ?? ''

    expect(palette.chooseResult(item, COMMAND.openFile)).toStrictEqual({
      command: 'app:file.open',
      over: { path: '', file: 'books/Atlas.pdf', title: '', selector: { page: 3 } },
      input: '',
    })
  })
})

describe('a search that finds nothing', () => {
  it('lists no row and so offers no entry', async () => {
    const palette = useSearch(createCore([]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })

    await typeQuery(palette)

    expect(palette.groups.value.flatMap((one) => one.items)).toEqual([])
  })
})

/** What is in front: a note, or no note. */
const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Being.md',
  title: 'Being',
  file: 'physics/Being.md',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

describe('the row that creates a note, in a search that found nothing', () => {
  const setUp = async (target: CommandTarget, text = '  Entropy  ') => {
    const palette = useSearch(createCore([]), WORDS, { wait: now, getTarget: () => target })
    await palette.setTyped(text)
    await flushPromises()
    return palette
  }

  const getRow = (palette: ReturnType<typeof useSearch>) =>
    palette.groups.value.find((one) => one.id === CREATE_NOTE_ROW)

  it('is the last group, and holds one item named after the words typed', async () => {
    const palette = await setUp(front())

    expect(palette.groups.value.at(-1)?.id).toBe(CREATE_NOTE_ROW)
    expect(palette.groups.value.at(-1)?.title).toBe('No results')
    expect(getRow(palette)?.items).toHaveLength(1)
    expect(getRow(palette)?.items[0]).toMatchObject({
      id: CREATE_NOTE_ROW,
      title: 'Create note “Entropy”',
      detail: 'Being',
    })
  })

  it('offers the command that creates a note and then the three relations to the note in front', async () => {
    const palette = await setUp(front())

    expect(getRow(palette)?.items[0]?.actions).toStrictEqual([
      { id: 'app:note.create', text: 'Create note' },
      { id: 'app:note.createChild', text: 'Create as child' },
      { id: 'app:note.createParent', text: 'Create as parent' },
      { id: 'app:note.createJump', text: 'Create as jump' },
    ])
  })

  it('offers the command that creates a note alone where no note is in front', async () => {
    const palette = await setUp(front({ path: '', title: '', file: '' }))

    expect(getRow(palette)?.items[0]?.actions).toStrictEqual([
      { id: 'app:note.create', text: 'Create note' },
    ])
    expect(getRow(palette)?.items[0]?.detail).toBeUndefined()
  })

  it('has an action whose id is the id of the row for none of its actions', async () => {
    const palette = await setUp(front())

    const ids = getRow(palette)?.items[0]?.actions?.map((one) => one.id) ?? []
    expect(ids.length).toBeGreaterThan(0)
    expect(ids).not.toContain(CREATE_NOTE_ROW)
  })

  it('is absent while a group is still working', async () => {
    const pending: SearchDeps = {
      names: () => new Promise(() => undefined),
      search: () => new Promise(() => undefined),
    }
    const palette = useSearch(pending, WORDS, { wait: now, getTarget: () => front() })

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()

    expect(palette.groups.value.some((one) => one.isWorking)).toBe(true)
    expect(getRow(palette)).toBeUndefined()
  })

  it('is absent where a group found something', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      getTarget: () => front(),
    })
    await palette.setTyped('cor')
    await flushPromises()

    expect(getRow(palette)).toBeUndefined()
  })

  it('is absent where nothing was typed', async () => {
    const palette = await setUp(front(), '   ')

    expect(getRow(palette)).toBeUndefined()
  })

  it.each([
    'app:note.create',
    'app:note.createChild',
    'app:note.createParent',
    'app:note.createJump',
  ])(
    'answers the choice of %s with that command, over the window, with the trimmed words as input',
    async (action) => {
      const palette = await setUp(front())

      expect(palette.chooseResult(CREATE_NOTE_ROW, action)).toStrictEqual({
        command: action,
        over: null,
        input: 'Entropy',
      })
    },
  )

  it('answers an action the row does not carry with nothing', async () => {
    const palette = await setUp(front({ path: '', title: '', file: '' }))

    expect(palette.chooseResult(CREATE_NOTE_ROW, 'app:note.createChild')).toBeNull()
    expect(palette.chooseResult(CREATE_NOTE_ROW, CREATE_NOTE_ROW)).toBeNull()
  })

  it('answers with the words typed at the moment of the choice', async () => {
    const palette = await setUp(front())
    await palette.setTyped('Enthalpy')
    await flushPromises()

    expect(palette.chooseResult(CREATE_NOTE_ROW, 'app:note.create')?.input).toBe('Enthalpy')
  })
})
