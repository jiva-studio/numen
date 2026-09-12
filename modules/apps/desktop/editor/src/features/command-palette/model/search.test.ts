/**
 * The rules the palette is filled by, without a vault and without a browser.
 *
 * Three questions are in the air at once and come back in whatever order they
 * take, so the ones that matter here are the negatives: an answer to a question
 * nobody is asking any more must not fill a list a person is reading, and a
 * group that failed must not take the other two down with it.
 */
import { describe, expect, it } from 'vitest'
import {
  useSearch,
  type SearchDeps,
  type NameMatch,
  type Passage,
  type SearchMode,
  type Words,
} from './search'
import { later, type Deferred } from '@/testing/later'

const WORDS: Words = {
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  travel: 'Show in plex',
  read: 'Open the note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document',
  noneFound: 'Nothing',
  notAsked: 'The vault could not answer',
  wordsOnly: 'Searching by words only — no model set',
  notEmbedded: 'This vault has not been read for meaning yet',
}

/** A vault that answers when the test says so, and remembers what it was asked. */
function createVault() {
  const names: Deferred<readonly NameMatch[]>[] = []
  const searched: { mode: SearchMode; answer: Deferred<readonly Passage[]> }[] = []
  const queries: string[] = []

  const core: SearchDeps = {
    names: (query) => {
      queries.push(query)
      const one = later<readonly NameMatch[]>()
      names.push(one)
      return one.promise
    },
    search: (query, mode) => {
      queries.push(query)
      const one = later<readonly Passage[]>()
      searched.push({ mode, answer: one })
      return one.promise
    },
  }

  const mode = (which: SearchMode) => searched.find((one) => one.mode === which)?.answer
  return { core, names, searched, queries, mode }
}

/** Nothing waits in a test; the hold is a clock and the clock is handed in. */
const now = async () => {}

/** Let everything already resolved run before the assertion. */
const flushPromises = async () => {
  for (let turn = 0; turn < 6; turn += 1) await Promise.resolve()
}

const createNameMatch = (over: Partial<NameMatch> = {}): NameMatch => ({
  path: 'notes/entropy.md',
  title: 'Entropy',
  heading: '',
  line: -1,
  at: [{ from: 0, to: 3 }],
  type: 'note',
  ...over,
})

const passage = (over: Partial<Passage> = {}): Passage => ({
  path: 'notes/heat.md',
  title: 'Heat engines',
  text: 'no engine beats a reversible one',
  isNote: true,
  type: 'note',
  kind: 'note',
  start: 0,
  length: 0,
  line: 0,
  at: [{ from: 3, to: 9 }],
  ...over,
})

/** The group under one identity, from what the palette is drawing now. */
const groupOf = (groups: readonly { id: string }[], id: string) =>
  groups.find((one) => one.id === id) as
    | { id: string; items: readonly { id: string; title: string }[]; working?: boolean; silence?: string }
    | undefined

describe('asking', () => {
  it('draws no group at all until something is typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    expect(palette.groups.value).toHaveLength(0)
    expect(vault.queries).toHaveLength(0)
  })

  it('asks all three questions at once, about what was typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()

    expect(vault.names).toHaveLength(1)
    expect(vault.searched.map((one) => one.mode)).toEqual(['words', 'meaning'])
    expect(new Set(vault.queries)).toEqual(new Set(['ent']))
  })

  it('asks about the words and not about the spaces around them', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('  ent  ')
    await flushPromises()

    expect(vault.queries).toEqual(['ent', 'ent', 'ent'])
  })

  it('asks nothing at all once what was typed is taken back', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()
    void palette.setTyped('')
    await flushPromises()

    expect(vault.names).toHaveLength(1)
    expect(palette.groups.value).toHaveLength(0)
  })
})

describe('answers arriving', () => {
  it('fills each group on its own, while the others are still out', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([createNameMatch()])
    await flushPromises()

    expect(groupOf(palette.groups.value, 'names')?.items).toHaveLength(1)
    expect(groupOf(palette.groups.value, 'names')?.working).toBe(false)
    expect(groupOf(palette.groups.value, 'text')?.working).toBe(true)
    expect(groupOf(palette.groups.value, 'meaning')?.working).toBe(true)
  })

  it('drops an answer to a question nobody is asking any more', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()
    void palette.setTyped('entr')
    await flushPromises()

    // The first question answers late, and with something else entirely.
    vault.names[0]?.answers([createNameMatch({ path: 'notes/stale.md', title: 'Stale' })])
    await flushPromises()
    expect(groupOf(palette.groups.value, 'names')?.items).toHaveLength(0)

    vault.names[1]?.answers([createNameMatch()])
    await flushPromises()
    expect(groupOf(palette.groups.value, 'names')?.items?.[0]?.title).toBe('Entropy')
  })

  it('says a group could not be asked in the window’s own words, and fills the others', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()

    vault.mode('meaning')?.fails('no model is set')
    vault.names[0]?.answers([createNameMatch()])
    await flushPromises()

    expect(groupOf(palette.groups.value, 'meaning')?.silence).toBe(WORDS.notAsked)
    expect(groupOf(palette.groups.value, 'meaning')?.working).toBe(false)
    expect(groupOf(palette.groups.value, 'names')?.items).toHaveLength(1)
  })

  it('says nothing of a group that was asked and came back with nothing', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    void palette.setTyped('ent')
    await flushPromises()
    vault.names[0]?.answers([])
    await flushPromises()

    const group = groupOf(palette.groups.value, 'names')
    expect(group?.silence).toBe('')
    expect(group?.working).toBe(false)
  })

  it('says the vault holds no vectors, where meaning came back with nothing', async () => {
    const vault = createVault()
    const read = { chunks: 4, embedded: 0, embedding: true }
    const palette = useSearch(vault.core, WORDS, { wait: now, coverage: () => read })

    void palette.setTyped('ent')
    await flushPromises()
    vault.mode('meaning')?.answers([])
    vault.names[0]?.answers([])
    await flushPromises()

    expect(groupOf(palette.groups.value, 'meaning')?.silence).toBe(WORDS.notEmbedded)
    // The other groups are asked of the words, which a vault holding no vector
    // still answers, and one of those that came back with nothing says nothing.
    expect(groupOf(palette.groups.value, 'names')?.silence).toBe('')
  })

  it('says nothing reads the vault for meaning, where nothing is set to', async () => {
    const vault = createVault()
    const read = { chunks: 4, embedded: 0, embedding: false }
    const palette = useSearch(vault.core, WORDS, { wait: now, coverage: () => read })

    void palette.setTyped('ent')
    await flushPromises()
    vault.mode('meaning')?.answers([])
    await flushPromises()

    expect(groupOf(palette.groups.value, 'meaning')?.silence).toBe(WORDS.wordsOnly)
  })

  it('lets go of everything when the palette is put away', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    palette.setOpen(true)
    void palette.setTyped('ent')
    await flushPromises()
    palette.setOpen(false)

    vault.names[0]?.answers([createNameMatch()])
    await flushPromises()

    expect(palette.open.value).toBe(false)
    expect(palette.typed.value).toBe('')
    expect(palette.groups.value).toHaveLength(0)
  })
})

describe('where a thing found takes the person', () => {
  const createFilledPalette = async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([
      createNameMatch(),
      createNameMatch({ path: 'notes/carnot.md', title: 'The Carnot cycle', heading: 'Entropy here', line: 12 }),
    ])
    vault.mode('words')?.answers([passage()])
    await flushPromises()
    return palette
  }

  it('opens a note found by its own name in the plex, and its text on the other key', async () => {
    const palette = await createFilledPalette()
    const item = groupOf(palette.groups.value, 'names')!.items[0]!

    expect(palette.chooseItem(item.id, 'plex')).toEqual({
      at: 'plex',
      path: 'notes/entropy.md',
      title: 'Entropy',
    })
    expect(palette.chooseItem(item.id, 'note')).toEqual({
      at: 'file',
      path: 'notes/entropy.md',
      title: 'Entropy',
    })
  })

  it('opens a note found by a heading inside it on the line that heading stands on', async () => {
    const palette = await createFilledPalette()
    const item = groupOf(palette.groups.value, 'names')!.items[1]!

    expect(palette.chooseItem(item.id, 'note')).toEqual({
      at: 'file',
      path: 'notes/carnot.md',
      title: 'The Carnot cycle',
      line: 12,
    })
    // The plex draws notes and not the lines inside them.
    expect(palette.chooseItem(item.id, 'plex')).toEqual({
      at: 'plex',
      path: 'notes/carnot.md',
      title: 'The Carnot cycle',
    })
  })

  it('opens a passage as the note it was read out of, on the line it stands on', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([createNameMatch()])
    vault.mode('words')?.answers([passage({ line: 12 })])
    await flushPromises()
    const item = groupOf(palette.groups.value, 'text')!.items[0]!

    expect(palette.chooseItem(item.id, 'note')).toEqual({
      at: 'file',
      path: 'notes/heat.md',
      title: 'Heat engines',
      line: 12,
    })
  })

  it('takes nobody anywhere for an item it is not drawing', async () => {
    const palette = await createFilledPalette()
    expect(palette.chooseItem('notes/nowhere.md', 'plex')).toBeNull()
  })
})

describe('what a key reaches, per kind of thing found', () => {
  it('offers the plex first for a name and the note first for a place', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([createNameMatch(), createNameMatch({ heading: 'Entropy here', line: 12 })])
    vault.mode('words')?.answers([passage()])
    await flushPromises()

    const getActions = (group: string, at: number) =>
      (
        palette.groups.value.find((one) => one.id === group)?.items[at]?.actions ?? []
      ).map((one) => one.id)

    expect(getActions('names', 0)).toEqual(['plex', 'note'])
    expect(getActions('names', 1)).toEqual(['note', 'plex'])
    expect(getActions('text', 0)).toEqual(['note', 'plex'])
  })
})

describe('a passage from something that is not a note', () => {
  it('offers the document, opened where the words were found', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('war')
    await flushPromises()

    vault.mode('words')?.answers([
      passage({
        path: 'library/mahabharata.epub',
        title: '',
        isNote: false,
        start: 40_512,
        length: 31,
      }),
    ])
    await flushPromises()

    const item = groupOf(palette.groups.value, 'text')!.items[0]! as {
      id: string
      title: string
      actions?: readonly { id: string; text: string }[]
    }

    expect(item.title).toBe('library/mahabharata.epub')
    expect(item.actions?.map((one) => one.text)).toEqual([WORDS.readDocument])

    // The document opens where the words stand in its own text.
    expect(palette.chooseItem(item.id, 'document')).toEqual({
      at: 'document',
      path: 'library/mahabharata.epub',
      title: '',
      start: 40_512,
      length: 31,
    })
    // It is neither a note nor a node, and neither is answered for.
    expect(palette.chooseItem(item.id, 'note')).toBeNull()
    expect(palette.chooseItem(item.id, 'plex')).toBeNull()
  })
})

describe('what a row is drawn as', () => {
  it('says what kind each name found is', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([
      createNameMatch(),
      createNameMatch({ path: 'decks/words.md', title: 'Words to learn', type: 'deck' }),
      createNameMatch({ path: 'stencils/animal.md', title: 'Animal', type: 'stencil' }),
      createNameMatch({ path: 'presets/daily.md', title: 'Every day', type: 'preset' }),
    ])
    await flushPromises()

    const items = groupOf(palette.groups.value, 'names')!.items
    expect(items.map((one) => palette.typeOf(one.id))).toEqual([
      'note',
      'deck',
      'stencil',
      'preset',
    ])
  })

  it('draws a heading as the note it stands in', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([
      createNameMatch({ path: 'decks/words.md', title: 'Words to learn', heading: 'Entropy', line: 12, type: 'deck' }),
    ])
    await flushPromises()

    const item = groupOf(palette.groups.value, 'names')!.items[0]!
    expect(palette.typeOf(item.id)).toBe('deck')
  })

  it('draws a passage as the note it was read out of', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.mode('words')?.answers([
      passage({ path: 'presets/daily.md', type: 'preset' }),
      passage({
        path: 'library/mahabharata.epub',
        title: '',
        isNote: false,
        kind: 'book',
        start: 40_512,
      }),
    ])
    await flushPromises()

    const items = groupOf(palette.groups.value, 'text')!.items
    expect(items.map((one) => palette.typeOf(one.id))).toEqual(['preset', null])
  })

  it('says what the vault holds where a passage was read out of', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.mode('words')?.answers([
      passage({ path: 'notes/heat.md' }),
      passage({ path: 'library/mahabharata.epub', isNote: false, kind: 'book', start: 40_512 }),
      passage({ path: 'talks/730709BG.LON.mp3', isNote: false, kind: 'recording', start: 12 }),
    ])
    await flushPromises()

    const items = groupOf(palette.groups.value, 'text')!.items
    expect(items.map((one) => palette.kindOf(one.id))).toEqual(['note', 'book', 'recording'])
  })

  it('says a name stands in a note, whatever kind of note it is', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('ent')
    await flushPromises()

    vault.names[0]?.answers([createNameMatch(), createNameMatch({ path: 'decks/words.md', type: 'deck' })])
    await flushPromises()

    const items = groupOf(palette.groups.value, 'names')!.items
    expect(items.map((one) => palette.kindOf(one.id))).toEqual(['note', 'note'])
  })

  it('says nothing about an item it is not drawing', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    expect(palette.typeOf('notes/entropy.md')).toBeNull()
    expect(palette.kindOf('notes/entropy.md')).toBeNull()
  })
})

describe('a group landing under the keyboard', () => {
  it('names a passage by where it stands in the vault, not by where it stands in the list', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    void palette.setTyped('war')
    await flushPromises()

    vault.mode('words')?.answers([passage({ path: 'notes/heat.md' })])
    await flushPromises()
    const before = groupOf(palette.groups.value, 'text')!.items[0]!.id

    // The same passage, now second in its group because a better one arrived.
    void palette.setTyped('war ')
    await flushPromises()
    vault.searched
      .filter((one) => one.mode === 'words')
      .at(-1)
      ?.answer.answers([
        passage({ path: 'notes/fire.md', title: 'Fire' }),
        passage({ path: 'notes/heat.md' }),
      ])
    await flushPromises()

    const after = groupOf(palette.groups.value, 'text')!.items[1]!.id
    expect(after).toBe(before)
  })
})
