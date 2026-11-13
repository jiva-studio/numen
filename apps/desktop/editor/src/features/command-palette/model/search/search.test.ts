/**
 * The rules the palette is filled by, without a vault and without a browser.
 *
 * Three queries run concurrently and return in any order, so the cases that
 * matter are the negative ones: the response to a superseded query must not
 * fill a list the user is reading, and a failed group must not fail the other
 * two.
 */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND, ROW } from '@/shared/model/command'
import { useSearch, type SearchDeps, type NameMatch, type Passage, type Words } from './search'
import type { SearchMode } from '../../lib/search/types'
import { createDeferred, type Deferred } from '@/testing/deferred'

function rethrow(error: unknown): never {
  throw error
}

const WORDS: Words = {
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  read: 'Open note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document',
  noneFound: 'Nothing',
  noQuery: 'Search failed',
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
}

/** The first action of a note is the id of the editor command that opens it. */
const OPEN_NOTE = COMMAND.openFile

function createVault() {
  const names: Deferred<readonly NameMatch[]>[] = []
  const searched: {
    mode: SearchMode
    answer: Deferred<readonly Passage[]>
    signal: AbortSignal | undefined
  }[] = []
  const queries: string[] = []

  const core: SearchDeps = {
    names: (query) => {
      queries.push(query)
      const one = createDeferred<readonly NameMatch[]>()
      names.push(one)
      return one.promise
    },
    search: (query, mode, _limit, signal) => {
      queries.push(query)
      const one = createDeferred<readonly Passage[]>()
      searched.push({ mode, answer: one, signal })
      return one.promise
    },
  }

  const mode = (which: SearchMode) => searched.find((one) => one.mode === which)?.answer
  return { core, names, searched, queries, mode }
}

/** Nothing waits in a test; the hold is a clock and the clock is handed in. */
const now = vi.fn(() => Promise.resolve())

const flushPromises = async () => {
  for (let turn = 0; turn < 6; turn += 1) await Promise.resolve()
}

const createNameMatch = (override: Partial<NameMatch> = {}): NameMatch => ({
  path: 'notes/entropy.md',
  title: 'Entropy',
  heading: '',
  line: -1,
  at: [{ start: 0, end: 3 }],
  type: 'note',
  ...override,
})

const passage = (override: Partial<Passage> = {}): Passage => ({
  path: 'notes/heat.md',
  title: 'Heat engines',
  text: 'no engine beats a reversible one',
  isNote: true,
  type: 'note',
  selector: null,
  start: 0,
  length: 0,
  line: 0,
  at: [{ start: 3, end: 9 }],
  ...override,
})

const getGroup = (groups: readonly { id: string }[], id: string) =>
  groups.find((one) => one.id === id) as
    | {
        id: string
        items: readonly { id: string; title: string }[]
        isWorking?: boolean
        silence?: string
      }
    | undefined

describe('asking', () => {
  it('draws no group at all until something is typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    expect(palette.groups.value).toHaveLength(0)
    expect(vault.queries).toHaveLength(0)
  })

  it('asks all three questions at once, about what was typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    expect(vault.names).toHaveLength(1)
    expect(vault.searched.map((one) => one.mode)).toEqual(['words', 'meaning'])
    expect(new Set(vault.queries)).toEqual(new Set(['ent']))
  })

  it('asks about the words and not about the spaces around them', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('  ent  ').catch(rethrow)
    await flushPromises()

    expect(vault.queries).toEqual(['ent', 'ent', 'ent'])
  })

  it('asks nothing at all once what was typed is taken back', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('').catch(rethrow)
    await flushPromises()

    expect(vault.names).toHaveLength(1)
    expect(palette.groups.value).toHaveLength(0)
  })
})

describe('answers arriving', () => {
  it('fills each group on its own, while the others are still out', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(getGroup(palette.groups.value, 'names')?.items).toHaveLength(1)
    expect(getGroup(palette.groups.value, 'names')?.isWorking).toBe(false)
    expect(palette.groups.value.map((one) => one.id)).toEqual(['names'])
  })

  it('shows what the words found while the meaning question is out, and adds meaning when it answers', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.names[0]?.answer([])
    vault.mode('words')?.answer([passage()])
    await flushPromises()

    expect(palette.groups.value.map((one) => one.id)).toEqual(['text'])
    expect(palette.status.isBusy.value).toBe(true)

    vault.mode('meaning')?.answer([passage({ path: 'notes/carnot.md', title: 'Carnot' })])
    await flushPromises()

    expect(palette.groups.value.map((one) => one.id)).toEqual(['text', 'meaning'])
    expect(palette.status.isBusy.value).toBe(false)
  })

  it('aborts the questions of a query once a newer query is typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    const older = vault.searched.find((one) => one.mode === 'meaning')?.signal
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    expect(older?.aborted).toBe(true)
    expect(vault.searched.at(-1)?.signal?.aborted).toBe(false)
  })

  it('drops a result of a superseded query', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    // The first question answers late, and with something else entirely.
    vault.names[0]?.answer([createNameMatch({ path: 'notes/stale.md', title: 'Stale' })])
    await flushPromises()
    expect(getGroup(palette.groups.value, 'names')?.items).toBeUndefined()

    vault.names[1]?.answer([createNameMatch()])
    await flushPromises()
    expect(getGroup(palette.groups.value, 'names')?.items?.[0]?.title).toBe('Entropy')
  })

  it('says a group could not be requested in the window’s own words, and fills the others', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.mode('meaning')?.fail('no model is set')
    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(getGroup(palette.groups.value, 'meaning')?.silence).toBe(WORDS.noQuery)
    expect(getGroup(palette.groups.value, 'meaning')?.isWorking).toBe(false)
    expect(getGroup(palette.groups.value, 'names')?.items).toHaveLength(1)
  })

  it('says nothing of a group that was requested and returned nothing', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.names[0]?.answer([])
    await flushPromises()

    expect(getGroup(palette.groups.value, 'names')).toBeUndefined()
  })

  it('draws no meaning group when the meaning search returned nothing', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.mode('meaning')?.answer([])
    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(getGroup(palette.groups.value, 'meaning')).toBeUndefined()
    expect(getGroup(palette.groups.value, 'names')?.items).toHaveLength(1)
  })

  it('shows meaning search results with meaning item ids', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.mode('meaning')?.answer([
      {
        path: 'note.md',
        title: 'Note',
        text: 'some snippet',
        at: [],
        isNote: true,
        start: 0,
        length: 10,
        line: 1,
        type: 'note',
        selector: null,
      },
    ])
    await flushPromises()

    const meaningGroup = getGroup(palette.groups.value, 'meaning')
    expect(meaningGroup?.items[0]?.id).toBe('meaning:note.md:0')
  })

  it('shows text search results with text item ids', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.names[0]?.answer([createNameMatch()])
    vault.mode('words')?.answer([passage()])
    await flushPromises()

    const textGroup = getGroup(palette.groups.value, 'text')
    expect(textGroup?.items[0]?.id).toBe('text:notes/heat.md:0')
    const namesItem = getGroup(palette.groups.value, 'names')?.items[0] as
      { id: string; actions?: readonly { id: string; text: string }[] } | undefined
    expect(namesItem?.actions).toStrictEqual([{ id: OPEN_NOTE, text: WORDS.read }])
  })

  it('releases everything when the palette is put away', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setOpen(true)
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setOpen(false)

    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(vault.searched.find((one) => one.mode === 'meaning')?.signal?.aborted).toBe(true)
    expect(palette.open.value).toBe(false)
    expect(palette.typed.value).toBe('')
    expect(palette.groups.value).toHaveLength(0)
  })
})

describe('where a thing found takes the person', () => {
  const createFilledPalette = async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([
      createNameMatch(),
      createNameMatch({
        path: 'notes/carnot.md',
        title: 'The Carnot cycle',
        heading: 'Entropy here',
        line: 12,
      }),
    ])
    vault.mode('words')?.answer([passage()])
    await flushPromises()
    return palette
  }

  it('opens a note found by its own name from the start', async () => {
    const palette = await createFilledPalette()
    const item = getGroup(palette.groups.value, 'names')!.items[0]!

    expect(palette.chooseResult(item.id, OPEN_NOTE)).toEqual({
      command: COMMAND.openFile,
      over: { path: 'notes/entropy.md', file: 'notes/entropy.md', title: 'Entropy' },
      input: '',
    })
  })

  it('opens a note found by a heading inside it on the line that heading relies on', async () => {
    const palette = await createFilledPalette()
    const item = getGroup(palette.groups.value, 'names')!.items[1]!

    expect(palette.chooseResult(item.id, OPEN_NOTE)).toEqual({
      command: COMMAND.openFile,
      over: {
        path: 'notes/carnot.md',
        file: 'notes/carnot.md',
        title: 'The Carnot cycle',
        line: 12,
      },
      input: '',
    })
  })

  it('opens a passage as the note it was read out of, on the line it relies on', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([createNameMatch()])
    vault.mode('words')?.answer([passage({ line: 12 })])
    await flushPromises()
    const item = getGroup(palette.groups.value, 'text')!.items[0]!

    expect(palette.chooseResult(item.id, OPEN_NOTE)).toEqual({
      command: COMMAND.openFile,
      over: { path: 'notes/heat.md', file: 'notes/heat.md', title: 'Heat engines', line: 12 },
      input: '',
    })
  })

  it('navigates nowhere for an item it is not drawing', async () => {
    const palette = await createFilledPalette()
    expect(palette.chooseResult('notes/nowhere.md', OPEN_NOTE)).toBeNull()
  })
})

describe('what a key reaches, per kind of thing found', () => {
  it('offers opening the note for a name, a heading and a passage', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([
      createNameMatch(),
      createNameMatch({ heading: 'Entropy here', line: 12 }),
    ])
    vault.mode('words')?.answer([passage()])
    await flushPromises()

    const getActions = (group: string, pos: number) =>
      (palette.groups.value.find((one) => one.id === group)?.items[pos]?.actions ?? []).map(
        (one) => one.id,
      )

    expect(getActions('names', 0)).toEqual([OPEN_NOTE])
    expect(getActions('names', 1)).toEqual([OPEN_NOTE])
    expect(getActions('text', 0)).toEqual([OPEN_NOTE])
  })
})

describe('a passage from something that is not a note', () => {
  it('offers the document, opened where the words were found', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('war').catch(rethrow)
    await flushPromises()

    vault.mode('words')?.answer([
      passage({
        path: 'library/mahabharata.epub',
        title: '',
        isNote: false,
        selector: { cfi: 'epubcfi(/6/4!/4/2)' },
        start: 40_512,
        length: 31,
      }),
    ])
    await flushPromises()

    const item = getGroup(palette.groups.value, 'text')!.items[0]! as {
      id: string
      title: string
      actions?: readonly { id: string; text: string }[]
    }

    expect(item.title).toBe('library/mahabharata.epub')
    expect(item.actions?.map((one) => one.text)).toEqual([WORDS.readDocument])

    // The document opens where the words stand in its own text.
    expect(palette.chooseResult(item.id, ROW.openDocument)).toEqual({
      command: 'app:file.open#document',
      over: {
        path: '',
        file: 'library/mahabharata.epub',
        title: '',
        selector: { cfi: 'epubcfi(/6/4!/4/2)' },
      },
      input: '',
    })
  })
})

describe('what a row is drawn as', () => {
  it('says what kind each name found is', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([
      createNameMatch(),
      createNameMatch({
        path: 'samples/words.md',
        title: 'Words to learn',
        type: 'sample',
        tabKind: 'sample',
      }),
      createNameMatch({ path: 'forms/animal.md', title: 'Animal', type: 'form', tabKind: 'form' }),
      createNameMatch({
        path: 'sheets/daily.md',
        title: 'Every day',
        type: 'sheet',
        tabKind: 'sheet',
      }),
    ])
    await flushPromises()

    const items = getGroup(palette.groups.value, 'names')!.items
    expect(items.map((one) => palette.reads.findTabKind(one.id))).toEqual([
      '',
      'sample',
      'form',
      'sheet',
    ])
  })

  it('draws a heading as the note it remains in', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([
      createNameMatch({
        path: 'samples/words.md',
        title: 'Words to learn',
        heading: 'Entropy',
        line: 12,
        type: 'sample',
        tabKind: 'sample',
      }),
    ])
    await flushPromises()

    const item = getGroup(palette.groups.value, 'names')!.items[0]!
    expect(palette.reads.findTabKind(item.id)).toBe('sample')
  })

  it('draws a passage as the note it was read out of', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.mode('words')?.answer([
      passage({ path: 'sheets/daily.md', type: 'sheet', tabKind: 'sheet' }),
      passage({
        path: 'library/mahabharata.epub',
        title: '',
        isNote: false,
        selector: { page: 3 },
        start: 40_512,
      }),
    ])
    await flushPromises()

    const items = getGroup(palette.groups.value, 'text')!.items
    expect(items.map((one) => palette.reads.findTabKind(one.id))).toEqual(['sheet', null])
  })

  it('names the file of a passage read out of a file that is no note', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault
      .mode('words')
      ?.answer([
        passage({ path: 'notes/heat.md' }),
        passage({ path: 'library/a.bin', isNote: false, selector: { page: 3 } }),
        passage({ path: 'talks/b.bin', isNote: false, selector: { t: [12, 14] } }),
      ])
    await flushPromises()

    const items = getGroup(palette.groups.value, 'text')!.items
    expect(items.map((one) => palette.reads.findHitFile(one.id))).toEqual([
      null,
      'library/a.bin',
      'talks/b.bin',
    ])
  })

  it('says a name remains in a note, whatever kind of note it is', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([
      createNameMatch(),
      createNameMatch({ path: 'samples/words.md', type: 'sample', tabKind: 'sample' }),
    ])
    await flushPromises()

    const items = getGroup(palette.groups.value, 'names')!.items
    expect(items.map((one) => palette.reads.findHitFile(one.id))).toEqual([null, null])
  })

  it('says nothing about an item it is not drawing', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    expect(palette.reads.findTabKind('notes/entropy.md')).toBeNull()
    expect(palette.reads.findHitFile('notes/entropy.md')).toBeNull()
  })
})

describe('a group landing under the keyboard', () => {
  it('keys a passage by its vault position, not by its list position', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('war').catch(rethrow)
    await flushPromises()

    vault.mode('words')?.answer([passage({ path: 'notes/heat.md' })])
    await flushPromises()
    const before = getGroup(palette.groups.value, 'text')!.items[0]!.id

    // The same passage, now second in its group because a better one arrived.
    palette.setTyped('war ').catch(rethrow)
    await flushPromises()
    vault.searched
      .filter((one) => one.mode === 'words')
      .at(-1)
      ?.answer.answer([
        passage({ path: 'notes/fire.md', title: 'Fire' }),
        passage({ path: 'notes/heat.md' }),
      ])
    await flushPromises()

    const after = getGroup(palette.groups.value, 'text')!.items[1]!.id
    expect(after).toBe(before)
  })
})

function createHolds() {
  const holds: Deferred<void>[] = []
  const wait = () => {
    const one = createDeferred<void>()
    holds.push(one)
    return one.promise
  }
  return { holds, wait }
}

describe('the hold before a question', () => {
  it('draws no group and no spinner while the first keystroke waits', async () => {
    const vault = createVault()
    const { wait } = createHolds()
    const palette = useSearch(vault.core, WORDS, { wait })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    expect(palette.groups.value).toStrictEqual([])
    expect(palette.status.isBusy.value).toBe(false)
  })

  it('shows the spinner and draws no group once the hold is out and nothing has answered', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(true)
    expect(palette.groups.value).toStrictEqual([])
  })

  it('asks nothing about a keystroke another one overtook while it waited', async () => {
    const vault = createVault()
    const { holds, wait } = createHolds()
    const palette = useSearch(vault.core, WORDS, { wait })

    palette.setTyped('e').catch(rethrow)
    palette.setTyped('en').catch(rethrow)
    holds[0]?.answer()
    await flushPromises()

    expect(vault.queries).toStrictEqual([])

    holds[1]?.answer()
    await flushPromises()

    expect(vault.queries).toStrictEqual(['en', 'en', 'en'])
  })
})

describe('what was typed taken back', () => {
  it('draws no group of what was found before anything else is typed', async () => {
    const vault = createVault()
    const { holds, wait } = createHolds()
    const palette = useSearch(vault.core, WORDS, { wait })
    palette.setTyped('ent').catch(rethrow)
    holds[0]?.answer()
    await flushPromises()
    vault.names[0]?.answer([createNameMatch()])
    vault.mode('words')?.answer([passage()])
    vault.mode('meaning')?.fail('no model is set')
    await flushPromises()

    palette.setTyped('').catch(rethrow)
    palette.setTyped('heat').catch(rethrow)
    await flushPromises()

    expect(palette.groups.value).toStrictEqual([])
  })

  it('draws no group at all where only spaces are typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })

    palette.setTyped('   ').catch(rethrow)
    await flushPromises()

    expect(palette.groups.value).toHaveLength(0)
  })
})

describe('answers to a question overtaken', () => {
  it('draws no failure of a superseded query', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    vault.names[1]?.answer([createNameMatch()])
    vault.names[0]?.fail('the vault went away')
    await flushPromises()

    expect(getGroup(palette.groups.value, 'names')?.items).toHaveLength(1)
    expect(getGroup(palette.groups.value, 'names')?.silence ?? '').toBe('')
  })

  it('keeps the spinner where only the overtaken question has answered', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(true)
    expect(palette.groups.value).toStrictEqual([])
  })

  it('holds nothing in a group whose question failed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.fail('the vault went away')
    await flushPromises()

    expect(getGroup(palette.groups.value, 'names')?.items).toStrictEqual([])
  })

  it('draws no group of an earlier question as soon as another question is typed', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    vault.names[0]?.answer([createNameMatch()])
    vault.mode('words')?.answer([passage()])
    vault.mode('meaning')?.answer([passage({ path: 'notes/heat.md', title: 'Heat' })])
    await flushPromises()

    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    expect(palette.groups.value).toStrictEqual([])
  })
})

describe('the palette opened and put away', () => {
  it('keeps what was typed and what answered when it is opened while open', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, {
      wait: now,
    })
    palette.setOpen(true)
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    palette.setOpen(true)
    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    expect(palette.typed.value).toBe('ent')
    expect(getGroup(palette.groups.value, 'names')?.items).toHaveLength(1)
  })

  it('draws nothing from before it was put away once it is typed into again', async () => {
    const vault = createVault()
    const { holds, wait } = createHolds()
    const palette = useSearch(vault.core, WORDS, { wait })
    palette.setOpen(true)
    palette.setTyped('ent').catch(rethrow)
    holds[0]?.answer()
    await flushPromises()
    vault.names[0]?.answer([createNameMatch()])
    await flushPromises()

    palette.setOpen(false)
    vault.mode('words')?.answer([passage()])
    await flushPromises()
    palette.setOpen(true)
    palette.setTyped('heat').catch(rethrow)
    await flushPromises()

    expect(palette.groups.value).toStrictEqual([])
  })
})

describe('the spinner of the field', () => {
  const answerAll = (vault: ReturnType<typeof createVault>, query: number) => {
    vault.names[query]?.answer([])
    for (const which of ['words', 'meaning'] as const)
      vault.searched.filter((one) => one.mode === which)[query]?.answer.answer([])
  }

  it('stays while one question of the query is out and goes once the last one fails', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    vault.names[0]?.answer([createNameMatch()])
    vault.mode('words')?.fail('the index is closed')
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(true)

    vault.mode('meaning')?.fail('no model is set')
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(false)
  })

  it('goes when the query is cleared, and the late answers of that query do not bring it back', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    palette.setTyped('').catch(rethrow)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(false)

    answerAll(vault, 0)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(false)
  })

  it('goes when the palette is put away with questions out', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    palette.setOpen(true)
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()

    palette.setOpen(false)

    expect(palette.status.isBusy.value).toBe(false)
  })

  it('stays gone when the overtaken query answers after the newer query has answered', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    answerAll(vault, 1)
    await flushPromises()
    expect(palette.status.isBusy.value).toBe(false)

    answerAll(vault, 0)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(false)
  })

  it('stays while the newer query waits for its meaning, though the overtaken query has answered', async () => {
    const vault = createVault()
    const palette = useSearch(vault.core, WORDS, { wait: now })
    palette.setTyped('ent').catch(rethrow)
    await flushPromises()
    palette.setTyped('entr').catch(rethrow)
    await flushPromises()

    vault.names[1]?.answer([])
    vault.searched.filter((one) => one.mode === 'words')[1]?.answer.answer([])
    answerAll(vault, 0)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(true)
  })
})
