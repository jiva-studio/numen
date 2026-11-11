/** The menu of a search result as the palette draws it and answers a choice from it. */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { SlotCommand, SlotItem, CommandTarget as RunTarget } from '@numen/plugin-loader'
import { COMMAND, ROW, type CommandTarget } from '@/shared/model/command'
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
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
}

const ATLAS = 'org.example.atlas'
const SHOW: SlotCommand = {
  kind: 'command',
  key: `${ATLAS}/command/graph.showNote`,
  pluginId: ATLAS,
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
  shortcut: '',
  hasItems: false,
}
const OTHER: SlotCommand = {
  kind: 'command',
  key: `${ATLAS}/command/other.run`,
  pluginId: ATLAS,
  command: `${ATLAS}:other.run`,
  title: 'Another',
  shortcut: '',
  hasItems: false,
}
const SEPARATOR: SlotItem = { kind: 'separator', key: 'separator/1' }
const SKETCH: SlotItem = {
  kind: 'view',
  key: `${ATLAS}/view/sketch`,
  pluginId: ATLAS,
  view: 'sketch',
  component: {},
}
const openId = COMMAND.openFile
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

  it('asks for the entries of the menu over the note of each result', async () => {
    const targets: RunTarget[] = []
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: {
        getEntries: (target) => {
          targets.push(target)
          return [SHOW]
        },
      },
    })

    await typeQuery(palette)
    expect(getNamesGroup(palette)).toBeDefined()

    expect(targets).toContainEqual({
      path: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: [],
      tab: '',
    })
  })

  it('offers the commands of the entries alone, without separators and views', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW, SEPARATOR, SKETCH, OTHER] },
    })

    await typeQuery(palette)

    expect(getNamesGroup(palette)?.items[0]?.actions?.map((one) => one.id)).toStrictEqual([
      openId,
      showId,
      OTHER.command,
    ])
  })

  it('offers opening the note alone once the plugin that placed the entry is disabled', async () => {
    const entries = ref<readonly SlotItem[]>([SHOW])
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
      command: COMMAND.openFile,
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
      { id: ROW.openDocument, text: 'Open the document' },
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

    expect(palette.chooseResult(item, ROW.openDocument)).toStrictEqual({
      command: 'app:file.open#document',
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

describe('the actions that create a note, in a search', () => {
  const setUp = async (target: CommandTarget, text = '  Entropy  ') => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      getTarget: () => target,
    })
    await palette.setTyped(text)
    await flushPromises()
    return palette
  }

  const getActions = (palette: ReturnType<typeof useSearch>, group = 'names') =>
    palette.groups.value.find((one) => one.id === group)?.items[0]?.actions ?? []

  it('draws no row of its own', async () => {
    const palette = await setUp(front())

    expect(palette.groups.value.map((one) => one.id)).toEqual(['names'])
  })

  it('follows the actions of a row with the command that creates a note and then the three relations to the note in front', async () => {
    const palette = await setUp(front())

    expect(getActions(palette)).toStrictEqual([
      { id: COMMAND.openFile, text: 'Open note' },
      { id: 'app:note.create', text: 'Create note “Entropy”' },
      { id: 'app:note.createChild', text: 'Create as child “Entropy”' },
      { id: 'app:note.createParent', text: 'Create as parent “Entropy”' },
      { id: 'app:note.createJump', text: 'Create as jump “Entropy”' },
    ])
  })

  it('offers the command that creates a note alone where no note is in front', async () => {
    const palette = await setUp(front({ path: '', title: '', file: '' }))

    expect(getActions(palette).map((one) => one.id)).toEqual([COMMAND.openFile, 'app:note.create'])
  })

  it('is offered on a row that found something in the text too', async () => {
    const palette = useSearch(createCore([], [bookPassage]), WORDS, {
      wait: now,
      getTarget: () => front(),
    })
    await palette.setTyped('Entropy')
    await flushPromises()

    expect(getActions(palette, 'text').map((one) => one.id)).toContain('app:note.create')
  })

  it('is offered as soon as a row is there, while the other groups are still working', async () => {
    const searchDeps: SearchDeps = {
      names: async () => [nameMatch],
      search: () => new Promise(() => undefined),
    }
    const palette = useSearch(searchDeps, WORDS, { wait: now, getTarget: () => front() })

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()

    expect(getActions(palette).map((one) => one.id)).toContain('app:note.create')
  })

  it('is offered on no row where nothing was typed', async () => {
    const palette = await setUp(front(), '   ')

    expect(palette.groups.value).toEqual([])
  })

  it('is offered on no row where the window gives no target', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, { wait: now })
    await palette.setTyped('cor')
    await flushPromises()

    expect(getActions(palette).map((one) => one.id)).toEqual([COMMAND.openFile])
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

      expect(palette.chooseResult(nameMatch.path, action)).toStrictEqual({
        command: action,
        over: null,
        input: 'Entropy',
      })
    },
  )

  it('answers an action of no row with nothing', async () => {
    const palette = await setUp(front({ path: '', title: '', file: '' }))

    expect(palette.chooseResult(nameMatch.path, 'app:note.createChild')).toBeNull()
  })

  it('still opens the note on the choice of its own action', async () => {
    const palette = await setUp(front())

    expect(palette.chooseResult(nameMatch.path, COMMAND.openFile)?.over?.path).toBe(nameMatch.path)
  })

  it('answers with the words typed at the moment of the choice', async () => {
    const palette = await setUp(front())
    await palette.setTyped('Enthalpy')
    await flushPromises()

    expect(palette.chooseResult(nameMatch.path, 'app:note.create')?.input).toBe('Enthalpy')
  })
})
