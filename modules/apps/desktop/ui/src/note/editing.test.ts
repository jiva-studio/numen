import { describe, expect, it, vi } from 'vitest'

import { editing, type Notes } from './editing'
import type { Core } from '../core'

/** A vault that has been read and is doing nothing. */
const idle = {
  name: '',
  path: '',
  ready: true,
  failed: '',
  unwatched: '',
  unreachable: '',
  chunks: 0n,
  embedded: 0n,
  embedding: false,
}

/** A file's fingerprint, which follows what the file holds. */
const marked = (body: string): string => `at:${body}`

/** Everything a tab asks of the core, and the rest of what a window asks. */
type Faked = Omit<Core, 'read' | 'write'> & Notes

/** A core that answers reads and writes from what a test puts in it. */
function fake(over: Partial<Faked> = {}) {
  const files = new Map<string, string>()
  const wrote: { path: string; body: string }[] = []
  const core: Faked = {
    neighbourhood: async () => ({}) as never,
    headings: async () => new Map(),
    standing: async () => new Map(),
    opening: async () => null,
    state: async () => idle,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    attending: async () => {},
    // eslint-disable-next-line require-yield
    editing: async function* () {},
    tasks: async function* () {},
    // eslint-disable-next-line require-yield
    quitting: async function* () {},
    flushed: async () => {},
    read: async (path) =>
      files.has(path)
        ? { body: files.get(path) ?? '', refusal: null, at: marked(files.get(path) ?? '') }
        : { body: '', refusal: 'missing' },
    write: async (path, body, seen) => {
      wrote.push({ path, body })
      const held = files.get(path)
      // A note still holding either the prose or the file that prose came out of
      // is the note this caller read.
      if (seen && held !== undefined && held !== seen.prose && marked(held) !== seen.at) {
        return { body: '', refusal: null, changed: true }
      }
      files.set(path, body)
      return { body: '', refusal: null, at: marked(body) }
    },
    create: async () => ({ path: '', refusal: null }),
    join: async () => null,
    rename: async (path) => ({
      path,
      title: '',
      frontmatter: false,
      moved: null,
      refusal: null,
      changed: false,
    }),
    remove: async () => ({ trashed: '', dangling: [], refusal: null }),
    list: async () => [],
    move: async () => ({ moved: null, refusal: null }),
    makeFolder: async () => null,
    syncing: async () => true,
    hanging: async () => ({ hangs: true, parts: 6 }),
    choosesSyncing: async () => null,
    choosesHanging: async () => null,
    reviewing: async () => '04:00',
    choosesReviewing: async () => null,
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

describe('a note whose file is about to be renamed or removed', () => {
  it('writes what is unsaved before it answers', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, { quiet: 10_000, bound: 10_000 })
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await notes.settles('Heat.md')

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
    expect(notes.shown('Heat.md').state).toBe('clean')
  })

  it('answers what a write in the air still owes before it says it has settled', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, { quiet: 10_000, bound: 10_000 })
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    notes.save('Heat.md')
    notes.typed('Heat.md', 'one two three')
    await notes.settles('Heat.md')

    expect(wrote.map((one) => one.body)).toEqual(['one two', 'one two three'])
    expect(files.get('Heat.md')).toBe('one two three')
  })

  it('fires nothing at the path it is leaving once it has settled', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    // A write slower than the interval armed by the keystroke before it.
    const slow: Faked = {
      ...core,
      write: async (path, body, seen) => {
        await new Promise((wake) => setTimeout(wake, 20))
        return core.write(path, body, seen)
      },
    }
    const notes = editing(slow, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await notes.settles('Heat.md')
    await new Promise((wake) => setTimeout(wake, 20))

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
  })

  it('answers at once for a note with nothing unsaved, and for one it never held', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    await notes.settles('Heat.md')
    await notes.settles('Nowhere.md')

    expect(wrote).toEqual([])
    expect(notes.all()).toEqual(['Heat.md'])
  })

  it('leaves the tab open, so the note is followed wherever it went', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    files.set('Warmth.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    await notes.settles('Heat.md')
    notes.changed(['Warmth.md'], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()

    expect(notes.all()).toEqual(['Heat.md'])
    expect(notes.where('Heat.md')).toBe('Warmth.md')
  })
})

describe('where a note stands', () => {
  it('is the file its tab opened with while nothing has moved it', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    expect(notes.where('Heat.md')).toBe('Heat.md')
    expect(notes.where('Nowhere.md')).toBe('Nowhere.md')
  })

  it('follows each rename, so a note moved twice stands at the last of them', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.changed([], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()
    notes.changed([], [{ from: 'Warmth.md', to: 'Entropy.md' }])
    await settle()

    expect(notes.where('Heat.md')).toBe('Entropy.md')
  })

  /**
   * A note stands at the name of its identity while the window has none open
   * under it, so whether the window has one at all is asked apart.
   */
  it('is answered for only while the window holds the note', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('held', 'Heat.md')
    await settle()

    expect(notes.has('held')).toBe(true)
    expect(notes.has('never opened')).toBe(false)

    await notes.shut('held')

    expect(notes.has('held')).toBe(false)
  })
})

describe('a save asked for now', () => {
  it('writes what is unsaved without waiting for the text to be still', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, { quiet: 10_000, bound: 10_000 })
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    notes.save('Heat.md')
    await settle()

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
  })

  it('writes nothing for a note nothing was typed into', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.save('Heat.md')
    await settle()

    expect(wrote).toEqual([])
  })
})

describe('a note that changed on disk under a save', () => {
  /** A note open and typed into, whose file moved before the write landed. */
  const caught = async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'mine')
    files.set('Heat.md', 'theirs')
    notes.save('Heat.md')
    await settle()
    return { notes, files, wrote }
  }

  it('is put to the person, and nothing more is written', async () => {
    const { notes, wrote } = await caught()

    expect(notes.shown('Heat.md').state).toBe('overtaken')
    expect(notes.overtaken('Heat.md')?.says).toBe('this note changed on disk, and saving stopped')
    expect(notes.shown('Heat.md').body).toBe('mine')

    notes.typed('Heat.md', 'mine and more')
    await new Promise((wake) => setTimeout(wake, 20))

    expect(wrote).toHaveLength(1)
    expect(notes.shown('Heat.md').state).toBe('overtaken')
  })

  it('keeps what the person has, and the file takes it', async () => {
    const { notes, files } = await caught()

    notes.keep('Heat.md')
    await settle()

    expect(files.get('Heat.md')).toBe('mine')
    expect(notes.shown('Heat.md').state).toBe('clean')
    expect(notes.overtaken('Heat.md')).toBeNull()
  })

  it("takes the file's, and what it holds replaces what was typed", async () => {
    const { notes } = await caught()

    notes.take('Heat.md')
    await settle()

    expect(notes.shown('Heat.md').body).toBe('theirs')
    expect(notes.shown('Heat.md').state).toBe('clean')
    expect(notes.overtaken('Heat.md')).toBeNull()
  })

  it('does not stop a note whose own two saves follow each other', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    notes.typed('Heat.md', 'one two three')
    await vi.waitFor(() => expect(wrote).toHaveLength(2))

    expect(files.get('Heat.md')).toBe('one two three')
    expect(notes.shown('Heat.md').state).toBe('clean')
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
  it('says why in the tab it was refused on, for as long as that tab is open', async () => {
    const { core, files } = fake({
      write: async () => ({ body: '', refusal: 'bodyRefused' }),
    })
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', '---\nnot a body\n---\n')
    await vi.waitFor(() => expect(notes.shown('Heat.md').state).toBe('stuck'))

    expect(notes.saying('Heat.md')).toBe(
      'a note begins below its frontmatter, and this text begins with one',
    )
  })

  it('says the vault was out of reach when the core did not answer', async () => {
    const { core, files } = fake({
      write: async () => {
        throw new Error('no transport')
      },
    })
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.shown('Heat.md').state).toBe('stuck'))

    expect(notes.saying('Heat.md')).toBe(
      'the vault could not be reached, so this note was not written',
    )
  })
})

describe('closing a note that could not be written', () => {
  it('holds it once and says why, and lets it go when it is asked again', async () => {
    const { core, files } = fake({
      write: async () => {
        throw new Error('no transport')
      },
    })
    files.set('Heat.md', 'one')
    const notes = editing(core, quick)
    notes.open('Heat.md')
    await settle()

    notes.typed('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.shown('Heat.md').state).toBe('stuck'))

    await notes.shut('Heat.md')
    expect(notes.all()).toEqual(['Heat.md'])
    expect(notes.saying('Heat.md')).toBe(
      'the vault could not be reached, so this note was not written',
    )

    await notes.shut('Heat.md')
    expect(notes.all()).toEqual([])
  })

  it('lets a note it could never read go the first time', async () => {
    const { core } = fake({ read: async () => ({ body: '', refusal: 'notText' }) })
    const notes = editing(core, quick)
    notes.open('photo.md')
    await settle()

    await notes.shut('photo.md')

    expect(notes.all()).toEqual([])
  })
})
