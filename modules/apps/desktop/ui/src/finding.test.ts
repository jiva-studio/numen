/**
 * The rules the palette is filled by, without a vault and without a browser.
 *
 * Three questions are in the air at once and come back in whatever order they
 * take, so the ones that matter here are the negatives: an answer to a question
 * nobody is asking any more must not fill a list a person is reading, and a
 * band that failed must not take the other two down with it.
 */
import { describe, expect, it } from 'vitest'
import { finding, type Asking, type Way, type Named, type Passage, type Words } from './finding'

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

/** An answer the test hands over when it chooses to. */
interface Later<T> {
  promise: Promise<T>
  answers: (value: T) => void
  fails: (why: string) => void
}

function later<T>(): Later<T> {
  let answers!: (value: T) => void
  let fails!: (why: unknown) => void
  const promise = new Promise<T>((resolve, reject) => {
    answers = resolve
    fails = reject
  })
  return { promise, answers, fails: (why: string) => fails(new Error(why)) }
}

/** A vault that answers when the test says so, and remembers what it was asked. */
function asking() {
  const names: Later<readonly Named[]>[] = []
  const searched: { way: Way; answer: Later<readonly Passage[]> }[] = []
  const queries: string[] = []

  const core: Asking = {
    names: (query) => {
      queries.push(query)
      const one = later<readonly Named[]>()
      names.push(one)
      return one.promise
    },
    search: (query, way) => {
      queries.push(query)
      const one = later<readonly Passage[]>()
      searched.push({ way, answer: one })
      return one.promise
    },
  }

  const way = (which: Way) => searched.find((one) => one.way === which)?.answer
  return { core, names, searched, queries, way }
}

/** Nothing waits in a test; the hold is a clock and the clock is handed in. */
const now = async () => {}

/** Let everything already resolved run before the assertion. */
const settled = async () => {
  for (let turn = 0; turn < 6; turn += 1) await Promise.resolve()
}

const named = (over: Partial<Named> = {}): Named => ({
  path: 'notes/entropy.md',
  title: 'Entropy',
  heading: '',
  line: -1,
  at: [{ from: 0, to: 3 }],
  ...over,
})

const passage = (over: Partial<Passage> = {}): Passage => ({
  path: 'notes/heat.md',
  title: 'Heat engines',
  text: 'no engine beats a reversible one',
  isNote: true,
  start: 0,
  length: 0,
  line: 0,
  at: [{ from: 3, to: 9 }],
  ...over,
})

/** The band under one identity, from what the palette is drawing now. */
const bandOf = (bands: readonly { id: string }[], id: string) =>
  bands.find((one) => one.id === id) as
    | { id: string; items: readonly { id: string; title: string }[]; working?: boolean; silence?: string }
    | undefined

describe('asking', () => {
  it('draws no band at all until something is typed', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    expect(palette.bands.value).toHaveLength(0)
    expect(vault.queries).toHaveLength(0)
  })

  it('asks all three questions at once, about what was typed', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()

    expect(vault.names).toHaveLength(1)
    expect(vault.searched.map((one) => one.way)).toEqual(['words', 'meaning'])
    expect(new Set(vault.queries)).toEqual(new Set(['ent']))
  })

  it('asks about the words and not about the spaces around them', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('  ent  ')
    await settled()

    expect(vault.queries).toEqual(['ent', 'ent', 'ent'])
  })

  it('asks nothing at all once what was typed is taken back', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()
    void palette.typing('')
    await settled()

    expect(vault.names).toHaveLength(1)
    expect(palette.bands.value).toHaveLength(0)
  })
})

describe('answers arriving', () => {
  it('fills each band on its own, while the others are still out', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()

    vault.names[0]?.answers([named()])
    await settled()

    expect(bandOf(palette.bands.value, 'names')?.items).toHaveLength(1)
    expect(bandOf(palette.bands.value, 'names')?.working).toBe(false)
    expect(bandOf(palette.bands.value, 'text')?.working).toBe(true)
    expect(bandOf(palette.bands.value, 'meaning')?.working).toBe(true)
  })

  it('drops an answer to a question nobody is asking any more', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()
    void palette.typing('entr')
    await settled()

    // The first question answers late, and with something else entirely.
    vault.names[0]?.answers([named({ path: 'notes/stale.md', title: 'Stale' })])
    await settled()
    expect(bandOf(palette.bands.value, 'names')?.items).toHaveLength(0)

    vault.names[1]?.answers([named()])
    await settled()
    expect(bandOf(palette.bands.value, 'names')?.items?.[0]?.title).toBe('Entropy')
  })

  it('says a band could not be asked in the window’s own words, and fills the others', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()

    vault.way('meaning')?.fails('no model is set')
    vault.names[0]?.answers([named()])
    await settled()

    expect(bandOf(palette.bands.value, 'meaning')?.silence).toBe(WORDS.notAsked)
    expect(bandOf(palette.bands.value, 'meaning')?.working).toBe(false)
    expect(bandOf(palette.bands.value, 'names')?.items).toHaveLength(1)
  })

  it('says a band came back with nothing in the window’s own words', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    void palette.typing('ent')
    await settled()
    vault.names[0]?.answers([])
    await settled()

    expect(bandOf(palette.bands.value, 'names')?.silence).toBe('Nothing')
  })

  it('says the vault holds no vectors, where meaning came back with nothing', async () => {
    const vault = asking()
    const read = { chunks: 4, embedded: 0, embedding: true }
    const palette = finding(vault.core, WORDS, now, () => read)

    void palette.typing('ent')
    await settled()
    vault.way('meaning')?.answers([])
    vault.names[0]?.answers([])
    await settled()

    expect(bandOf(palette.bands.value, 'meaning')?.silence).toBe(WORDS.notEmbedded)
    // The other bands are asked of the words, which a vault holding no vector
    // still answers.
    expect(bandOf(palette.bands.value, 'names')?.silence).toBe(WORDS.noneFound)
  })

  it('says nothing reads the vault for meaning, where nothing is set to', async () => {
    const vault = asking()
    const read = { chunks: 4, embedded: 0, embedding: false }
    const palette = finding(vault.core, WORDS, now, () => read)

    void palette.typing('ent')
    await settled()
    vault.way('meaning')?.answers([])
    await settled()

    expect(bandOf(palette.bands.value, 'meaning')?.silence).toBe(WORDS.wordsOnly)
  })

  it('lets go of everything when the palette is put away', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)

    palette.shows(true)
    void palette.typing('ent')
    await settled()
    palette.shows(false)

    vault.names[0]?.answers([named()])
    await settled()

    expect(palette.open.value).toBe(false)
    expect(palette.typed.value).toBe('')
    expect(palette.bands.value).toHaveLength(0)
  })
})

describe('where a thing found takes the person', () => {
  const filled = async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)
    void palette.typing('ent')
    await settled()

    vault.names[0]?.answers([
      named(),
      named({ path: 'notes/carnot.md', title: 'The Carnot cycle', heading: 'Entropy here', line: 12 }),
    ])
    vault.way('words')?.answers([passage()])
    await settled()
    return palette
  }

  it('opens a note found by its own name in the plex, and its text on the other key', async () => {
    const palette = await filled()
    const item = bandOf(palette.bands.value, 'names')!.items[0]!

    expect(palette.chose(item.id, 'plex')).toEqual({
      at: 'plex',
      path: 'notes/entropy.md',
      title: 'Entropy',
    })
    expect(palette.chose(item.id, 'note')).toEqual({
      at: 'note',
      path: 'notes/entropy.md',
      title: 'Entropy',
    })
  })

  it('opens a note found by a heading inside it on the line that heading stands on', async () => {
    const palette = await filled()
    const item = bandOf(palette.bands.value, 'names')!.items[1]!

    expect(palette.chose(item.id, 'note')).toEqual({
      at: 'note',
      path: 'notes/carnot.md',
      title: 'The Carnot cycle',
      line: 12,
    })
    // The plex draws notes and not the lines inside them.
    expect(palette.chose(item.id, 'plex')).toEqual({
      at: 'plex',
      path: 'notes/carnot.md',
      title: 'The Carnot cycle',
    })
  })

  it('opens a passage as the note it was read out of, on the line it stands on', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)
    void palette.typing('ent')
    await settled()

    vault.names[0]?.answers([named()])
    vault.way('words')?.answers([passage({ line: 12 })])
    await settled()
    const item = bandOf(palette.bands.value, 'text')!.items[0]!

    expect(palette.chose(item.id, 'note')).toEqual({
      at: 'note',
      path: 'notes/heat.md',
      title: 'Heat engines',
      line: 12,
    })
  })

  it('takes nobody anywhere for an item it is not drawing', async () => {
    const palette = await filled()
    expect(palette.chose('notes/nowhere.md', 'plex')).toBeNull()
  })
})

describe('what a key reaches, per kind of thing found', () => {
  it('offers the plex first for a name and the note first for a place', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)
    void palette.typing('ent')
    await settled()

    vault.names[0]?.answers([named(), named({ heading: 'Entropy here', line: 12 })])
    vault.way('words')?.answers([passage()])
    await settled()

    const acts = (band: string, at: number) =>
      (
        palette.bands.value.find((one) => one.id === band)?.items[at]?.actions ?? []
      ).map((one) => one.id)

    expect(acts('names', 0)).toEqual(['plex', 'note'])
    expect(acts('names', 1)).toEqual(['note', 'plex'])
    expect(acts('text', 0)).toEqual(['note', 'plex'])
  })
})

describe('a passage from something that is not a note', () => {
  it('offers the document, opened where the words were found', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)
    void palette.typing('war')
    await settled()

    vault.way('words')?.answers([
      passage({
        path: 'library/mahabharata.epub',
        title: '',
        isNote: false,
        start: 40_512,
        length: 31,
      }),
    ])
    await settled()

    const item = bandOf(palette.bands.value, 'text')!.items[0]! as {
      id: string
      title: string
      actions?: readonly { id: string; text: string }[]
    }

    expect(item.title).toBe('library/mahabharata.epub')
    expect(item.actions?.map((one) => one.text)).toEqual([WORDS.readDocument])

    // The document opens where the words stand in its own text.
    expect(palette.chose(item.id, 'document')).toEqual({
      at: 'document',
      path: 'library/mahabharata.epub',
      title: '',
      start: 40_512,
      length: 31,
    })
    // It is neither a note nor a node, and neither is answered for.
    expect(palette.chose(item.id, 'note')).toBeNull()
    expect(palette.chose(item.id, 'plex')).toBeNull()
  })
})

describe('a band landing under the keyboard', () => {
  it('names a passage by where it stands in the vault, not by where it stands in the list', async () => {
    const vault = asking()
    const palette = finding(vault.core, WORDS, now)
    void palette.typing('war')
    await settled()

    vault.way('words')?.answers([passage({ path: 'notes/heat.md' })])
    await settled()
    const before = bandOf(palette.bands.value, 'text')!.items[0]!.id

    // The same passage, now second in its band because a better one arrived.
    void palette.typing('war ')
    await settled()
    vault.searched
      .filter((one) => one.way === 'words')
      .at(-1)
      ?.answer.answers([
        passage({ path: 'notes/fire.md', title: 'Fire' }),
        passage({ path: 'notes/heat.md' }),
      ])
    await settled()

    const after = bandOf(palette.bands.value, 'text')!.items[1]!.id
    expect(after).toBe(before)
  })
})
