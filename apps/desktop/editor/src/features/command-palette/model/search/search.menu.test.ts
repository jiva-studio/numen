/** The menu of a search result as the palette draws it and answers a choice from it. */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { PlacedCommand } from '@numen/plugin-host'
import { COMMAND } from '@/shared/commandIds'
import { MENU_ACTION_PREFIX } from './lookup'
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
}

const ATLAS = 'org.example.atlas'
const SHOW: PlacedCommand = {
  pluginId: ATLAS,
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
  group: 'navigate',
  order: 20,
}
const OTHER: PlacedCommand = {
  pluginId: ATLAS,
  command: `${ATLAS}:other.run`,
  title: 'Another',
  group: '',
}
const openId = `${MENU_ACTION_PREFIX}${COMMAND.openNote}`
const showId = `${MENU_ACTION_PREFIX}${SHOW.command}`

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
      { id: `${MENU_ACTION_PREFIX}${OTHER.command}`, text: 'Another' },
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
    const entries = ref<readonly PlacedCommand[]>([SHOW])
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

    expect(palette.chooseEntry(item, showId)).toStrictEqual({
      command: SHOW.command,
      path: 'reef/Coral.md',
      title: 'Coral',
      type: 'sample',
      line: -1,
    })
  })

  it('runs the command that opens the note over the result on the choice of the first action', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })
    await typeQuery(palette)
    const item = getNamesGroup(palette)?.items[0]?.id ?? ''

    expect(palette.chooseEntry(item, openId)).toStrictEqual({
      command: COMMAND.openNote,
      path: 'reef/Coral.md',
      title: 'Coral',
      type: 'sample',
      line: -1,
    })
    expect(palette.chooseItem(item, openId)).toBeNull()
    expect(palette.chooseItem(item, 'note')).toBeNull()
  })

  it('opens nothing on the choice of an entry', async () => {
    const palette = useSearch(createCore([nameMatch]), WORDS, {
      wait: now,
      menu: { getEntries: () => [SHOW] },
    })
    await typeQuery(palette)
    const item = getNamesGroup(palette)?.items[0]?.id ?? ''

    expect(palette.chooseItem(item, showId)).toBeNull()
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
    expect(text?.items[0]?.actions).toStrictEqual([{ id: 'note', text: 'Open the document' }])
    expect(palette.chooseEntry(text?.items[0]?.id ?? '', showId)).toBeNull()
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
