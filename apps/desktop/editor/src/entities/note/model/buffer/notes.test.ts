import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from './results'
import { computed, watch } from 'vue'

import { openNotes, type Notes } from './notes'

/** A file's file stat, which follows what the file holds. */
const statFile = (body: string): string => `at:${body}`

/** A core that answers reads and writes from what a test puts in it. */
function fake(overrides: Partial<Notes> = {}) {
  const files = new Map<string, string>()
  const wrote: { path: string; body: string }[] = []
  const core: Notes = {
    read: async (path) =>
      files.has(path)
        ? asValue({ body: files.get(path) ?? '', at: statFile(files.get(path) ?? '') })
        : asFailure('missing'),
    write: async (path, body, seen) => {
      wrote.push({ path, body })
      const existing = files.get(path)
      // A note still holding either the prose or the file that prose came out of
      // is the note this caller read.
      if (
        seen &&
        existing !== undefined &&
        existing !== seen.prose &&
        statFile(existing) !== seen.stat
      ) {
        return asFailure('changed' as const)
      }
      files.set(path, body)
      return asValue({ body: '', at: statFile(body) })
    },
    ...overrides,
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
    const notes = openNotes(core, { limits: quick })

    notes.open('Heat.md')
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('entropy grows')
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
  })

  it('is done once for a note already open', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'first')
    const notes = openNotes(core, { limits: quick })

    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'mine')
    files.set('Heat.md', 'second')
    notes.open('Heat.md')
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('mine')
  })

  it('leaves a note that is not there empty, and the next write makes it', async () => {
    const { core, wrote } = fake()
    const notes = openNotes(core, { limits: quick })

    notes.open('New.md')
    await settle()
    expect(notes.getOpenNote('New.md').body).toBe('')

    notes.setBody('New.md', 'a first line')
    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    expect(wrote[0]).toEqual({ path: 'New.md', body: 'a first line' })
  })

  it('sticks on an error, and says why in words a person reads', async () => {
    const { core } = fake({ read: async () => asFailure('notText' as const) })
    const notes = openNotes(core, { limits: quick })

    notes.open('photo.md')
    await settle()

    expect(notes.getOpenNote('photo.md').state).toBe('stuck')
    expect(notes.getErrorMessage('photo.md')).toBe('not a text file')
  })
})

describe('typing', () => {
  it('writes once the text has been still', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    expect(wrote).toHaveLength(0)

    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    expect(wrote[0]?.body).toBe('one two')
  })

  it('writes nothing for a buffer that did not move', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one')
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(wrote).toHaveLength(0)
  })
})

describe('a change in the vault', () => {
  it('is read again, and the document is left alone where the text is the same', async () => {
    const requests: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'unchanged')
    const notes = openNotes(
      { ...core, read: async (path) => (requests.push(path), core.read(path)) },
      { limits: quick },
    )
    notes.open('Heat.md')
    await settle()

    notes.applyPathChanges(['Heat.md'])
    await settle()

    expect(requests).toEqual(['Heat.md', 'Heat.md'])
    expect(notes.getOpenNote('Heat.md').body).toBe('unchanged')
  })

  it('reaches a note with nothing unsaved, and passes one with something', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'on disk')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'mine')
    files.set('Heat.md', 'somebody else')
    notes.applyPathChanges(['Heat.md'])
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('mine')
  })

  it('naming no paths at all reaches every note with nothing unsaved', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'before')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    files.set('Heat.md', 'after')
    notes.applyPathChanges([])
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('after')
  })

  it('naming other paths only reaches nobody', async () => {
    const requests: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'x')
    const notes = openNotes(
      { ...core, read: async (path) => (requests.push(path), core.read(path)) },
      { limits: quick },
    )
    notes.open('Heat.md')
    await settle()

    notes.applyPathChanges(['Other.md'])
    await settle()

    expect(requests).toHaveLength(1)
  })
})

describe('closing', () => {
  it('writes the unsaved changes before the note goes', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await notes.close('Heat.md')

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
    expect(notes.getOpenIds()).toEqual([])
  })

  it('takes a note with nothing unsaved straight away', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    await notes.close('Heat.md')

    expect(wrote).toHaveLength(0)
    expect(notes.getOpenIds()).toEqual([])
  })

  it('by a flush leaves nothing open and nothing unwritten', async () => {
    const { core, files, wrote } = fake()
    files.set('a.md', 'a')
    files.set('b.md', 'b')
    const notes = openNotes(core, { limits: quick })
    notes.open('a.md')
    notes.open('b.md')
    await settle()

    notes.setBody('a.md', 'a changed')
    notes.setBody('b.md', 'b changed')
    await notes.flush()

    expect(wrote.map((w) => w.path).sort()).toEqual(['a.md', 'b.md'])
    expect(notes.getOpenIds()).toEqual([])
  })
})

describe('a note whose file is about to be renamed or removed', () => {
  it('writes what is unsaved before it answers', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: { quiet: 10_000, bound: 10_000 } })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await notes.flushPending('Heat.md')

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
  })

  it('answers the changes a write in the air still has pending before it says it has flushed', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: { quiet: 10_000, bound: 10_000 } })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    notes.save('Heat.md')
    notes.setBody('Heat.md', 'one two three')
    await notes.flushPending('Heat.md')

    expect(wrote.map((one) => one.body)).toEqual(['one two', 'one two three'])
    expect(files.get('Heat.md')).toBe('one two three')
  })

  it('fires nothing at the path it is leaving once it has flushed', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    // A write slower than the interval armed by the keystroke before it.
    const slow: Notes = {
      ...core,
      write: async (path, body, seen) => {
        await new Promise((resolve) => setTimeout(resolve, 20))
        return core.write(path, body, seen)
      },
    }
    const notes = openNotes(slow, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await notes.flushPending('Heat.md')
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
  })

  it('answers at once for a note with nothing unsaved, and for one it never retained', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    await notes.flushPending('Heat.md')
    await notes.flushPending('Nowhere.md')

    expect(wrote).toEqual([])
    expect(notes.getOpenIds()).toEqual(['Heat.md'])
  })

  it('leaves the tab open, so the note is followed wherever it went', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    files.set('Warmth.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    await notes.flushPending('Heat.md')
    notes.applyPathChanges(['Warmth.md'], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()

    expect(notes.getOpenIds()).toEqual(['Heat.md'])
    expect(notes.getPath('Heat.md')).toBe('Warmth.md')
  })
})

describe('where a note remains', () => {
  it('is the file its tab opened with while nothing has moved it', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    expect(notes.getPath('Heat.md')).toBe('Heat.md')
    expect(notes.getPath('Nowhere.md')).toBe('Nowhere.md')
  })

  it('follows each rename, so a note moved twice remains at the last of them', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.applyPathChanges([], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()
    notes.applyPathChanges([], [{ from: 'Warmth.md', to: 'Entropy.md' }])
    await settle()

    expect(notes.getPath('Heat.md')).toBe('Entropy.md')
  })

  /**
   * Whether the window holds the note at all is checked separately.
   */
  it('is answered for only while the window holds the note', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('member', 'Heat.md')
    await settle()

    expect(notes.has('member')).toBe(true)
    expect(notes.has('never opened')).toBe(false)

    await notes.close('member')

    expect(notes.has('member')).toBe(false)
  })
})

describe('a save requested for now', () => {
  it('writes what is unsaved without waiting for the text to be still', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: { quiet: 10_000, bound: 10_000 } })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    notes.save('Heat.md')
    await settle()

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
  })

  it('writes nothing for a note nothing was typed into', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.save('Heat.md')
    await settle()

    expect(wrote).toEqual([])
  })
})

describe('a note that changed on disk under a save', () => {
  /** A note open and typed into, whose file moved before the write landed. */
  const createCaughtSave = async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'mine')
    files.set('Heat.md', 'theirs')
    notes.save('Heat.md')
    await settle()
    return { notes, files, wrote }
  }

  it('is put to the person, and nothing more is written', async () => {
    const { notes, wrote } = await createCaughtSave()

    expect(notes.getOpenNote('Heat.md').state).toBe('stale')
    expect(notes.stale('Heat.md')?.says).toBe('saving paused: the note changed on disk')
    expect(notes.getOpenNote('Heat.md').body).toBe('mine')

    notes.setBody('Heat.md', 'mine and more')
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(wrote).toHaveLength(1)
    expect(notes.getOpenNote('Heat.md').state).toBe('stale')
  })

  it('keeps what the person has, and the file takes it', async () => {
    const { notes, files } = await createCaughtSave()

    notes.keep('Heat.md')
    await settle()

    expect(files.get('Heat.md')).toBe('mine')
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
    expect(notes.stale('Heat.md')).toBeNull()
  })

  it("takes the file's, and what it holds replaces what was typed", async () => {
    const { notes } = await createCaughtSave()

    notes.take('Heat.md')
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('theirs')
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
    expect(notes.stale('Heat.md')).toBeNull()
  })

  it('does not stop a note whose own two saves follow each other', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(wrote).toHaveLength(1))
    notes.setBody('Heat.md', 'one two three')
    await vi.waitFor(() => expect(wrote).toHaveLength(2))

    expect(files.get('Heat.md')).toBe('one two three')
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
  })
})

describe('a core that cannot be reached', () => {
  it('sticks the note and not leaving it loading for ever', async () => {
    const { core } = fake({
      read: async () => {
        throw new Error('no transport')
      },
    })
    const notes = openNotes(core, { limits: quick })

    notes.open('Heat.md')
    await settle()

    expect(notes.getOpenNote('Heat.md').state).toBe('stuck')
  })
})

describe('a save that failed', () => {
  it('says why in the tab it failed on, for as long as that tab is open', async () => {
    const { core, files } = fake({
      write: async () => asFailure('bodyUnwritable' as const),
    })
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', '---\nnot a body\n---\n')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    expect(notes.getErrorMessage('Heat.md')).toBe('text cannot start with frontmatter')
  })

  it('says the vault was out of reach when the core did not answer', async () => {
    const { core, files } = fake({
      write: async () => {
        throw new Error('no transport')
      },
    })
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    expect(notes.getErrorMessage('Heat.md')).toBe('vault unreachable, note not saved')
  })
})

describe('closing a note that could not be written', () => {
  it('holds it once and says why, and lets it go when it is requested again', async () => {
    const { core, files } = fake({
      write: async () => {
        throw new Error('no transport')
      },
    })
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    await notes.close('Heat.md')
    expect(notes.getOpenIds()).toEqual(['Heat.md'])
    expect(notes.getErrorMessage('Heat.md')).toBe('vault unreachable, note not saved')

    await notes.close('Heat.md')
    expect(notes.getOpenIds()).toEqual([])
  })

  it('lets a note it could never read go the first time', async () => {
    const { core } = fake({ read: async () => asFailure('notText' as const) })
    const notes = openNotes(core, { limits: quick })
    notes.open('photo.md')
    await settle()

    await notes.close('photo.md')

    expect(notes.getOpenIds()).toEqual([])
  })
})

describe('reloading a note', () => {
  it('clears previous body and reloads new path', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'first body')
    files.set('Cold.md', 'second body')
    const notes = openNotes(core, { limits: quick })

    notes.open('Heat.md')
    await settle()
    expect(notes.getOpenNote('Heat.md').body).toBe('first body')

    notes.reload('Heat.md', 'Cold.md')
    await settle()
    expect(notes.getOpenNote('Heat.md').body).toBe('second body')
  })
})

describe('a reader of one open note', () => {
  /** Two notes open, each with a computed that counts how often it is evaluated. */
  const createReaders = async () => {
    const { core, files } = fake()
    files.set('A.md', 'a')
    files.set('B.md', 'b')
    const notes = openNotes(core, { limits: { quiet: 1000, bound: 5000 } })
    notes.open('A.md')
    notes.open('B.md')
    await settle()

    const runs = { A: 0, B: 0, ids: 0 }
    const noteA = computed(() => (runs.A++, notes.getOpenNote('A.md')))
    const noteB = computed(() => (runs.B++, notes.getOpenNote('B.md')))
    const ids = computed(() => (runs.ids++, notes.getOpenIds()))
    const read = () => [noteA.value, noteB.value, ids.value] as const
    read()
    return { notes, files, runs, read, noteA, noteB, ids }
  }

  it('is not evaluated again when another note is typed into', async () => {
    const { notes, runs, read, noteA, noteB } = await createReaders()
    const before = { ...runs }

    notes.setBody('A.md', 'a and more')
    read()

    expect(noteA.value.body).toBe('a and more')
    expect(noteB.value.body).toBe('b')
    expect(runs.A).toBe(before.A + 1)
    expect(runs.B).toBe(before.B)
    expect(runs.ids).toBe(before.ids)
  })

  it('is not evaluated again when another note is replaced by its file', async () => {
    const { notes, files, runs, read } = await createReaders()
    const before = { ...runs }

    files.set('A.md', 'changed on disk')
    notes.applyPathChanges(['A.md'])
    await settle()
    read()

    expect(notes.getOpenNote('A.md').body).toBe('changed on disk')
    expect(runs.B).toBe(before.B)
    expect(runs.ids).toBe(before.ids)
  })
})

describe('the ids of the open notes', () => {
  it('are read again when a note is opened or closed, and only then', async () => {
    const { core, files } = fake()
    files.set('A.md', 'a')
    files.set('B.md', 'b')
    const notes = openNotes(core, { limits: quick })
    const observed: string[][] = []
    const stop = watch(
      () => [...notes.getOpenIds()],
      (ids) => observed.push(ids),
      { flush: 'sync' },
    )

    notes.open('A.md')
    await settle()
    notes.setBody('A.md', 'a2')
    notes.open('B.md')
    await settle()
    notes.setBody('B.md', 'b2')
    expect(observed.at(-1)).toEqual(['A.md', 'B.md'])
    const afterOpen = observed.length

    notes.setBody('A.md', 'a3')
    expect(observed).toHaveLength(afterOpen)

    await notes.close('A.md')
    expect(observed.at(-1)).toEqual(['B.md'])
    stop()
  })

  it('keep the order of opening and the history of a note across typing', async () => {
    const { core, files, wrote } = fake()
    files.set('A.md', 'a')
    const notes = openNotes(core, { limits: quick })
    notes.open('A.md')
    await settle()

    notes.setBody('A.md', 'a1')
    notes.setBody('A.md', 'a12')
    expect(notes.getOpenNote('A.md')).toMatchObject({ body: 'a12', state: 'unsaved' })
    expect(notes.getOpenIds()).toEqual(['A.md'])

    await vi.waitFor(() => expect(wrote.at(-1)?.body).toBe('a12'))
    expect(notes.getOpenNote('A.md').state).toBe('clean')
  })
})

describe('the limits a store is given', () => {
  it('set how long the text is still before it is written', async () => {
    vi.useFakeTimers()
    try {
      const { core, files, wrote } = fake()
      files.set('Heat.md', 'one')
      const notes = openNotes(core, { limits: { quiet: 50, bound: 500 } })
      notes.open('Heat.md')
      await settle()

      notes.setBody('Heat.md', 'one two')
      vi.advanceTimersByTime(60)
      await settle()

      expect(wrote).toHaveLength(1)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('an id the store does not hold', () => {
  it('is turned by nothing that is told to it', () => {
    const { core } = fake()
    const notes = openNotes(core, { limits: quick })

    expect(() => {
      notes.setBody('ghost.md', 'text')
      notes.save('ghost.md')
      notes.keep('ghost.md')
      notes.take('ghost.md')
    }).not.toThrow()
    expect(notes.getOpenIds()).toEqual([])
  })

  it('is closed at once', async () => {
    const { core } = fake()
    const notes = openNotes(core, { limits: quick })

    expect(await notes.close('ghost.md')).toBe(true)
  })

  it('is a note still loading, standing at no file, with nothing to say', () => {
    const { core } = fake()
    const notes = openNotes(core, { limits: quick })

    expect(notes.getOpenNote('ghost.md')).toEqual({
      path: 'ghost.md',
      body: '',
      state: 'loading',
      error: null,
    })
    expect(notes.getFilePath('ghost.md')).toBe('')
    expect(notes.getErrorMessage('ghost.md')).toBe('')
  })
})

describe('a note with nothing wrong', () => {
  it('has no error message', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    expect(notes.getErrorMessage('Heat.md')).toBe('')
  })
})

describe('a note replaced by what its file holds', () => {
  it('is announced by the path it remains at', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'entropy grows')
    const onReplaced = vi.fn<(path: string) => void>()
    const notes = openNotes(core, { limits: quick, onReplaced })

    notes.open('tab-1', 'Heat.md')
    await settle()

    expect(onReplaced).toHaveBeenCalledTimes(1)
    expect(onReplaced).toHaveBeenCalledWith('Heat.md')
  })
})

describe('reloading a note that retained something', () => {
  it('shows an empty file as empty', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'first body')
    files.set('Cold.md', '')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.reload('Heat.md', 'Cold.md')
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('')
  })

  it('leaves no timer running for the text typed before it', async () => {
    vi.useFakeTimers()
    try {
      const { core, files } = fake()
      files.set('Heat.md', 'one')
      const notes = openNotes(core, { limits: { quiet: 50, bound: 500 } })
      notes.open('Heat.md')
      await settle()
      notes.setBody('Heat.md', 'one two')
      expect(vi.getTimerCount()).toBe(1)

      notes.reload('Heat.md')

      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('closing a note', () => {
  it('leaves no timer running for the text it wrote', async () => {
    vi.useFakeTimers()
    try {
      const { core, files } = fake()
      files.set('Heat.md', 'one')
      const notes = openNotes(core, { limits: { quiet: 50, bound: 500 } })
      notes.open('Heat.md')
      await settle()
      notes.setBody('Heat.md', 'one two')

      expect(await notes.close('Heat.md')).toBe(true)

      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })

  it('leaves no text behind', async () => {
    const { core } = fake({
      read: async () =>
        asValue({
          body: 'What I made of it.',
          at: statFile('What I made of it.'),
        }),
    })
    const notes = openNotes(core, { limits: quick })
    notes.open('Entropy.md')
    await settle()

    await notes.close('Entropy.md')

    expect(notes.getOpenNote('Entropy.md').body).toBe('')
  })

  it('is not answered while a write in the air is still being waited on', async () => {
    const base = fake()
    base.files.set('Heat.md', 'one')
    let release: () => void = () => {}
    const gate = new Promise<void>((done) => {
      release = done
    })
    let calls = 0
    const core: Notes = {
      ...base.core,
      write: async (path, body) => {
        calls += 1
        if (calls === 1) await gate
        base.files.set(path, body)
        return asValue({ body: '', at: statFile(body) })
      },
    }
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'one two')
    const answers: boolean[] = []
    void notes.close('Heat.md').then((closedResult) => answers.push(closedResult))

    notes.setBody('Heat.md', 'one two three')
    await settle()
    expect(answers).toEqual([])

    release()
    await vi.waitFor(() => expect(answers).toEqual([true]))
    expect(base.files.get('Heat.md')).toBe('one two three')
  })

  it('answers that a stale note was kept open, every time it is requested', async () => {
    const base = fake()
    base.files.set('Heat.md', 'one')
    const notes = openNotes(base.core, { limits: quick })
    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'mine')
    base.files.set('Heat.md', 'theirs')
    notes.save('Heat.md')
    await settle()
    expect(notes.getOpenNote('Heat.md').state).toBe('stale')

    expect(await notes.close('Heat.md')).toBe(false)
    expect(await notes.close('Heat.md')).toBe(false)
    expect(notes.has('Heat.md')).toBe(true)
  })
})

describe('a note that could not be written, closed, and opened again', () => {
  /** A core whose writes fail while `down.on` is set and succeed when it is cleared. */
  const createFlakyCore = () => {
    const base = fake()
    base.files.set('Heat.md', 'one')
    const down = { on: true }
    const core: Notes = {
      ...base.core,
      write: async (path, body) => {
        if (down.on) throw new Error('no transport')
        base.files.set(path, body)
        return asValue({ body: '', at: statFile(body) })
      },
    }
    return { core, down }
  }

  it('is retained again, and is not closed by the writes of the note it was', async () => {
    const { core, down } = createFlakyCore()
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))
    expect(await notes.close('Heat.md')).toBe(false)
    expect(await notes.close('Heat.md')).toBe(true)

    down.on = false
    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'three')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('clean'))
    expect(notes.has('Heat.md')).toBe(true)

    down.on = true
    notes.setBody('Heat.md', 'four')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))
    expect(await notes.close('Heat.md')).toBe(false)
  })

  it('is retained again once it is reloaded and fails again', async () => {
    const { core } = createFlakyCore()
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))
    expect(await notes.close('Heat.md')).toBe(false)

    notes.reload('Heat.md')
    await settle()
    notes.setBody('Heat.md', 'three')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    expect(await notes.close('Heat.md')).toBe(false)
  })
})
