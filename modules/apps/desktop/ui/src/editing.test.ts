import { describe, expect, it, vi } from 'vitest'

import { editing } from './editing'
import type { Answered, Core } from './showing'

/** A vault that has been read and is doing nothing. */
const idle = {
  name: '',
  ready: true,
  failed: '',
  unwatched: '',
  chunks: 0n,
  embedded: 0n,
  reading: '',
  embedding: false,
  books: 0n,
  booksRead: 0n,
  learning: false,
  busy: false,
}

/** A core that answers reads and writes from what a test puts in it. */
function fake(over: Partial<Core> = {}) {
  const files = new Map<string, string>()
  const wrote: { path: string; body: string }[] = []
  const core: Core = {
    neighbourhood: async () => ({}) as never,
    opening: async () => null,
    state: async () => idle,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    // eslint-disable-next-line require-yield
    quitting: async function* () {},
    flushed: async () => {},
    read: async (path): Promise<Answered> =>
      files.has(path)
        ? { body: files.get(path) ?? '', refusal: null }
        : { body: '', refusal: 'missing' },
    write: async (path, body): Promise<Answered> => {
      wrote.push({ path, body })
      files.set(path, body)
      return { body: '', refusal: null }
    },
    create: async () => ({ path: '', refusal: null }),
    join: async () => null,
    ...over,
  }
  return { core, files, wrote }
}

/** Let every microtask settle. */
const settle = async () => {
  for (let i = 0; i < 8; i++) await Promise.resolve()
}

const quick = { quiet: 1, bound: 5 }

describe('opening a note', () => {
  it('reads it, and shows what came back', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'entropy grows')
    const notes = editing(core, quick)

    notes.open('Heat.md')
    await settle()

    expect(notes.shown('Heat.md').body).toBe('entropy grows')
    expect(notes.shown('Heat.md').state).toBe('clean')
  })

  it('is done once for a note already open', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'first')
    const notes = editing(core, quick)

    notes.open('Heat.md')
    await settle()
    notes.typed('Heat.md', 'mine')
    files.set('Heat.md', 'second')
    notes.open('Heat.md')
    await settle()

    expect(notes.shown('Heat.md').body).toBe('mine')
  })

  it('leaves a note that is not there empty, and the next write makes it', async () => {
    const { core, wrote } = fake()
    const notes = editing(core, quick)

    notes.open('New.md')
    await settle()
    expect(notes.shown('New.md').body).toBe('')

    notes.typed('New.md', 'a first line')
    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    expect(wrote[0]).toEqual({ path: 'New.md', body: 'a first line' })
  })

  it('sticks on a refusal, and says why in words a person reads', async () => {
    const { core } = fake({ read: async () => ({ body: '', refusal: 'notText' }) })
    const notes = editing(core, quick)

    notes.open('photo.md')
    await settle()

    expect(notes.shown('photo.md').state).toBe('stuck')
    expect(notes.saying('photo.md')).toBe('this file is not text')
  })
})

describe('typing', () => {
  it('writes once the text has been still', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    expect(wrote).toHaveLength(0)

    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    expect(wrote[0]?.body).toBe('one two')
  })

  it('writes nothing for a buffer that did not move', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one')
    await new Promise((wake) => setTimeout(wake, 20))

    expect(wrote).toHaveLength(0)
  })
})

describe('a change in the vault', () => {
  it('is read again, and the document is left alone where the text is the same', async () => {
    const asked: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'unchanged')
    const notes = editing(
      { ...core, read: async (path) => (asked.push(path), core.read(path)) },
      quick,
    )
    notes.open('Heat.md')
    await settle()

    notes.changed(['Heat.md'])
    await settle()

    expect(asked).toEqual(['Heat.md', 'Heat.md'])
    expect(notes.shown('Heat.md').body).toBe('unchanged')
  })

  it('reaches a note with nothing unsaved, and passes one with something', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'on disk')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'mine')
    files.set('Heat.md', 'somebody else')
    notes.changed(['Heat.md'])
    await settle()

    expect(notes.shown('Heat.md').body).toBe('mine')
  })

  it('naming no paths at all reaches every note with nothing unsaved', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'before')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    files.set('Heat.md', 'after')
    notes.changed([])
    await settle()

    expect(notes.shown('Heat.md').body).toBe('after')
  })

  it('naming other paths only reaches nobody', async () => {
    const asked: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'x')
    const notes = editing(
      { ...core, read: async (path) => (asked.push(path), core.read(path)) },
      quick,
    )
    notes.open('Heat.md')
    await settle()

    notes.changed(['Other.md'])
    await settle()

    expect(asked).toHaveLength(1)
  })
})

describe('closing', () => {
  it('writes what is owed before the note goes', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await notes.shut('Heat.md')

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
    expect(notes.all()).toEqual([])
  })

  it('takes a note with nothing unsaved straight away', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    await notes.shut('Heat.md')

    expect(wrote).toHaveLength(0)
    expect(notes.all()).toEqual([])
  })

  it('by a flush leaves nothing open and nothing unwritten', async () => {
    const { core, files, wrote } = fake()
    files.set('a.md', 'a')
    files.set('b.md', 'b')
    const notes = editing(core, quick)
    notes.open('a.md')
    notes.open('b.md')
    await settle()

    notes.typed('a.md', 'a changed')
    notes.typed('b.md', 'b changed')
    await notes.flush()

    expect(wrote.map((w) => w.path).sort()).toEqual(['a.md', 'b.md'])
    expect(notes.all()).toEqual([])
  })
})

describe('a core that cannot be reached', () => {
  it('sticks the note instead of leaving it loading for ever', async () => {
    const { core } = fake({
      read: async () => {
        throw new Error('no transport')
      },
    })
    const notes = editing(core, quick)

    notes.open('Heat.md')
    await settle()

    expect(notes.shown('Heat.md').state).toBe('stuck')
  })
})

describe('a save that was refused', () => {
  it('is said in words that outlive the tab it was refused on', async () => {
    const { core, files } = fake({
      write: async () => ({ body: '', refusal: 'bodyRefused' }),
    })
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', '---\nnot a body\n---\n')
    await notes.shut('Heat.md')

    // The tab is gone, and what could not be written is still on screen.
    expect(notes.all()).toEqual([])
    expect(notes.said.value).toBe(
      'a note begins below its frontmatter, and this text begins with one',
    )
  })
})
