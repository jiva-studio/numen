import { describe, expect, it, vi } from 'vitest'

import { openNotes, type Notes } from './notes'

/** A file's fingerprint, which follows what the file holds. */
const getFingerprint = (body: string): string => `at:${body}`

/** A core that answers reads and writes from what a test puts in it. */
function fake(over: Partial<Notes> = {}) {
  const files = new Map<string, string>()
  const wrote: { path: string; body: string }[] = []
  const core: Notes = {
    read: async (path) =>
      files.has(path)
        ? { body: files.get(path) ?? '', error: null, at: getFingerprint(files.get(path) ?? '') }
        : { body: '', error: 'missing' },
    write: async (path, body, seen) => {
      wrote.push({ path, body })
      const held = files.get(path)
      // A note still holding either the prose or the file that prose came out of
      // is the note this caller read.
      if (seen && held !== undefined && held !== seen.prose && getFingerprint(held) !== seen.at) {
        return { body: '', error: null, changed: true }
      }
      files.set(path, body)
      return { body: '', error: null, at: getFingerprint(body) }
    },
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
    const { core } = fake({ read: async () => ({ body: '', error: 'notText' }) })
    const notes = openNotes(core, { limits: quick })

    notes.open('photo.md')
    await settle()

    expect(notes.getOpenNote('photo.md').state).toBe('stuck')
    expect(notes.getErrorMessage('photo.md')).toBe('this file is not text')
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
    await new Promise((wake) => setTimeout(wake, 20))

    expect(wrote).toHaveLength(0)
  })
})

describe('a change in the vault', () => {
  it('is read again, and the document is left alone where the text is the same', async () => {
    const asked: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'unchanged')
    const notes = openNotes(
      { ...core, read: async (path) => (asked.push(path), core.read(path)) },
      { limits: quick },
    )
    notes.open('Heat.md')
    await settle()

    notes.changed(['Heat.md'])
    await settle()

    expect(asked).toEqual(['Heat.md', 'Heat.md'])
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
    notes.changed(['Heat.md'])
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
    notes.changed([])
    await settle()

    expect(notes.getOpenNote('Heat.md').body).toBe('after')
  })

  it('naming other paths only reaches nobody', async () => {
    const asked: string[] = []
    const { core, files } = fake()
    files.set('Heat.md', 'x')
    const notes = openNotes(
      { ...core, read: async (path) => (asked.push(path), core.read(path)) },
      { limits: quick },
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
    await notes.settle('Heat.md')

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
    expect(notes.getOpenNote('Heat.md').state).toBe('clean')
  })

  it('answers what a write in the air still owes before it says it has settled', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: { quiet: 10_000, bound: 10_000 } })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    notes.save('Heat.md')
    notes.setBody('Heat.md', 'one two three')
    await notes.settle('Heat.md')

    expect(wrote.map((one) => one.body)).toEqual(['one two', 'one two three'])
    expect(files.get('Heat.md')).toBe('one two three')
  })

  it('fires nothing at the path it is leaving once it has settled', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    // A write slower than the interval armed by the keystroke before it.
    const slow: Notes = {
      ...core,
      write: async (path, body, seen) => {
        await new Promise((wake) => setTimeout(wake, 20))
        return core.write(path, body, seen)
      },
    }
    const notes = openNotes(slow, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await notes.settle('Heat.md')
    await new Promise((wake) => setTimeout(wake, 20))

    expect(wrote).toEqual([{ path: 'Heat.md', body: 'one two' }])
  })

  it('answers at once for a note with nothing unsaved, and for one it never held', async () => {
    const { core, files, wrote } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    await notes.settle('Heat.md')
    await notes.settle('Nowhere.md')

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

    await notes.settle('Heat.md')
    notes.changed(['Warmth.md'], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()

    expect(notes.getOpenIds()).toEqual(['Heat.md'])
    expect(notes.getPath('Heat.md')).toBe('Warmth.md')
  })
})

describe('where a note stands', () => {
  it('is the file its tab opened with while nothing has moved it', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    expect(notes.getPath('Heat.md')).toBe('Heat.md')
    expect(notes.getPath('Nowhere.md')).toBe('Nowhere.md')
  })

  it('follows each rename, so a note moved twice stands at the last of them', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.changed([], [{ from: 'Heat.md', to: 'Warmth.md' }])
    await settle()
    notes.changed([], [{ from: 'Warmth.md', to: 'Entropy.md' }])
    await settle()

    expect(notes.getPath('Heat.md')).toBe('Entropy.md')
  })

  /**
   * A note stands at the name of its identity while the window has none open
   * under it, so whether the window has one at all is asked apart.
   */
  it('is answered for only while the window holds the note', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('held', 'Heat.md')
    await settle()

    expect(notes.has('held')).toBe(true)
    expect(notes.has('never opened')).toBe(false)

    await notes.close('held')

    expect(notes.has('held')).toBe(false)
  })
})

describe('a save asked for now', () => {
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
    expect(notes.stale('Heat.md')?.says).toBe('this note changed on disk, and saving stopped')
    expect(notes.getOpenNote('Heat.md').body).toBe('mine')

    notes.setBody('Heat.md', 'mine and more')
    await new Promise((wake) => setTimeout(wake, 20))

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
  it('sticks the note instead of leaving it loading for ever', async () => {
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

describe('a save that was refused', () => {
  it('says why in the tab it was refused on, for as long as that tab is open', async () => {
    const { core, files } = fake({
      write: async () => ({ body: '', error: 'bodyUnwritable' }),
    })
    files.set('Heat.md', 'one')
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', '---\nnot a body\n---\n')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    expect(notes.getErrorMessage('Heat.md')).toBe(
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
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    expect(notes.getErrorMessage('Heat.md')).toBe(
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
    const notes = openNotes(core, { limits: quick })
    notes.open('Heat.md')
    await settle()

    notes.setBody('Heat.md', 'one two')
    await vi.waitFor(() => expect(notes.getOpenNote('Heat.md').state).toBe('stuck'))

    await notes.close('Heat.md')
    expect(notes.getOpenIds()).toEqual(['Heat.md'])
    expect(notes.getErrorMessage('Heat.md')).toBe(
      'the vault could not be reached, so this note was not written',
    )

    await notes.close('Heat.md')
    expect(notes.getOpenIds()).toEqual([])
  })

  it('lets a note it could never read go the first time', async () => {
    const { core } = fake({ read: async () => ({ body: '', error: 'notText' }) })
    const notes = openNotes(core, { limits: quick })
    notes.open('photo.md')
    await settle()

    await notes.close('photo.md')

    expect(notes.getOpenIds()).toEqual([])
  })
})

describe('a note that points somewhere', () => {
  const linked = {
    body: 'What I made of it.',
    error: null,
    at: getFingerprint('What I made of it.'),
    link: {
      url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
      embed: 'https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ?enablejsapi=1',
    },
  }

  it('carries the link, beside the prose it was read with', async () => {
    const { core } = fake({ read: async () => linked })
    const notes = openNotes(core, { limits: quick })

    notes.open('Entropy.md')
    await settle()

    expect(notes.link('Entropy.md')).toStrictEqual(linked.link)
    expect(notes.getOpenNote('Entropy.md').body).toBe('What I made of it.')
  })

  it('points nowhere once a read says it points nowhere', async () => {
    let link: (typeof linked)['link'] | undefined = linked.link
    const { core } = fake({
      read: async () => ({
        body: '',
        error: null,
        at: getFingerprint(''),
        ...(link ? { link } : {}),
      }),
    })
    const notes = openNotes(core, { limits: quick })

    notes.open('Entropy.md')
    await settle()
    expect(notes.link('Entropy.md')).not.toBeNull()

    link = undefined
    notes.changed(['Entropy.md'])
    await settle()

    expect(notes.link('Entropy.md')).toBeNull()
  })
})

describe('a note that points nowhere', () => {
  it('says so, which is what draws no player over its prose', async () => {
    const { core, files } = fake()
    files.set('Heat.md', 'entropy grows')
    const notes = openNotes(core, { limits: quick })

    notes.open('Heat.md')
    await settle()

    expect(notes.link('Heat.md')).toBeNull()
  })
})
