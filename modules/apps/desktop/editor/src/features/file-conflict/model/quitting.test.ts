/**
 * The route by which quitting reaches a buffer only the page holds.
 *
 * The failure this guards against is silent: the window goes, the application
 * answers that everything landed, and the last seconds of typing are gone.
 */
import { describe, expect, it } from 'vitest'
import { asFailure, asValue } from '@numen/wire'

import { openNotes, type Notes } from '@/entities/note'
import { useFileFlush, type Conflict, type FlushDeps, type FlushResult } from './flush'
import type { NoteResult } from '@/entities/note'

/** What the application sends over the quit stream during tests. */
function createMockStream() {
  const events: { token: string; flush: boolean }[] = []
  let wake: (() => void) | null = null
  let isOver = false
  return {
    send(message: { token: string; flush: boolean }) {
      events.push(message)
      wake?.()
    },
    /** The stream drops. The next one a page opens is a stream again. */
    end() {
      isOver = true
      wake?.()
    },
    async *read() {
      for (;;) {
        while (events.length > 0) yield events.shift() as { token: string; flush: boolean }
        if (isOver) {
          isOver = false
          return
        }
        await new Promise<void>((woken) => {
          wake = woken
        })
      }
    },
  }
}

/** A mock core whose reads and writes a test drives, and whose quit it speaks for. */
function createFakeCore(getQuits: () => AsyncIterable<{ token: string; flush: boolean }>) {
  const files = new Map<string, string>()
  const writtenFiles: { path: string; body: string }[] = []
  const reportedResults: { token: string; result: FlushResult }[] = []
  /** Writes wait here until a test lets them through. */
  let pendingWriteResolve: (() => void) | null = null

  const core: Notes & FlushDeps = {
    watchQuit: getQuits,
    reportFlush: async (token: string, result: FlushResult = 'written') => {
      reportedResults.push({ token, result })
    },
    read: async (path): Promise<NoteResult> =>
      files.has(path) ? asValue({ body: files.get(path) ?? '' }) : asFailure('missing'),
    write: async (path, body): Promise<NoteResult> => {
      if (pendingWriteResolve) await new Promise<void>((through) => (pendingWriteResolve = through))
      writtenFiles.push({ path, body })
      files.set(path, body)
      return asValue({ body: '' })
    },
  }
  return {
    core,
    files,
    writtenFiles,
    reportedResults,
    holdWrites: () => {
      pendingWriteResolve = () => {}
    },
    releaseWrites: () => {
      const through = pendingWriteResolve
      pendingWriteResolve = null
      through?.()
    },
  }
}

/**
 * A tab whose text changed under the file. Records the action taken.
 */
function createConflict(note: string) {
  const actions: string[] = []
  let drop = () => {}
  const conflict: Conflict = {
    note,
    keep: async () => {
      actions.push('keep ' + note)
      drop()
    },
    take: async () => {
      actions.push('take ' + note)
      drop()
    },
  }
  return {
    conflict,
    actions,
    raise: (raising: (one: Conflict) => () => void) => {
      drop = raising(conflict)
    },
    /** The conflict stops standing, however that came about. */
    answered: () => drop(),
  }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))
const settle = async () => {
  for (let i = 0; i < 20; i++) await nap()
}

describe('a page asked to write what it owes', () => {
  it('writes an unsaved tab and only then says it has', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes(fakeCore.core)
    const going = useFileFlush(fakeCore.core)
    going.addHandler(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.setBody('Note.md', 'what the person was in the middle of')

    // No interval has fired, so what was typed is in the page and nowhere else.
    expect(fakeCore.writtenFiles).toEqual([])

    mockStream.send({ token: '7', flush: true })
    await settle()

    expect(fakeCore.writtenFiles).toEqual([
      { path: 'Note.md', body: 'what the person was in the middle of' },
    ])
    expect(fakeCore.reportedResults).toEqual([{ token: '7', result: 'written' }])
  })

  it('does not answer while the write it owes is still in the air', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes(fakeCore.core)
    const going = useFileFlush(fakeCore.core)
    going.addHandler(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.setBody('Note.md', 'held')
    fakeCore.holdWrites()

    mockStream.send({ token: '1', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([])

    fakeCore.releaseWrites()
    await settle()

    expect(fakeCore.writtenFiles).toEqual([{ path: 'Note.md', body: 'held' }])
    expect(fakeCore.reportedResults).toEqual([{ token: '1', result: 'written' }])
  })

  it('says nothing until the application asks', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes(fakeCore.core)
    const going = useFileFlush(fakeCore.core)
    going.addHandler(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.setBody('Note.md', 'still being written')

    // The stream opens by handing over the token, which asks for nothing.
    mockStream.send({ token: '3', flush: false })
    await settle()

    expect(fakeCore.writtenFiles).toEqual([])
    expect(fakeCore.reportedResults).toEqual([])
  })

  it('answers for a window with nothing open', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core)
    going.addHandler(openNotes(fakeCore.core).flush)
    void going.start()

    mockStream.send({ token: '0', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '0', result: 'written' }])
  })
})

describe('a page holding text the file changed under', () => {
  it('says there are conflicts outstanding, and does not say it has written', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core)
    const note = createConflict('Note.md')
    note.raise(going.raise)
    void going.start()

    mockStream.send({ token: '4', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '4', result: 'asking' }])
    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Note.md'])
  })

  it('says so before the writes it owes have landed', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes(fakeCore.core)
    const going = useFileFlush(fakeCore.core)
    going.addHandler(notes.flush)
    const note = createConflict('Held.md')
    note.raise(going.raise)

    void going.start()
    notes.open('Other.md')
    await settle()
    notes.setBody('Other.md', 'on its way')
    fakeCore.holdWrites()

    mockStream.send({ token: '5', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '5', result: 'asking' }])

    fakeCore.releaseWrites()
    await settle()

    // The write landed and the conflict still stands, so nothing has changed
    // about what the page owes.
    expect(fakeCore.writtenFiles).toEqual([{ path: 'Other.md', body: 'on its way' }])
    expect(fakeCore.reportedResults).toEqual([{ token: '5', result: 'asking' }])
  })

  it('says it has written once every conflict is settled', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core)
    const first = createConflict('One.md')
    const second = createConflict('Two.md')
    first.raise(going.raise)
    second.raise(going.raise)
    void going.start()

    mockStream.send({ token: '6', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '6', result: 'asking' }])

    await going.conflicts.value.find((one) => one.note === 'One.md')?.keep()
    await settle()

    // One of the two is answered, so the window is still owed something.
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '6', result: 'asking' })
    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Two.md'])

    await going.conflicts.value.find((one) => one.note === 'Two.md')?.take()
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '6', result: 'written' })
    expect(going.conflicts.value).toEqual([])
    expect(first.actions).toEqual(['keep One.md'])
    expect(second.actions).toEqual(['take Two.md'])
  })

  it('leaves a conflict the person put off standing, and stops drawing it', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core, async () => {})
    createConflict('Later.md').raise(going.raise)
    void going.start()

    mockStream.send({ token: '8', flush: true })
    await settle()

    going.conflicts.value[0]?.later()
    await settle()

    expect(going.conflicts.value).toEqual([])
    // Nothing was answered for it, so the window is still owed it.
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '8', result: 'asking' })

    // And it is still owed it the next time the window is asked for.
    mockStream.end()
    await settle()
    mockStream.send({ token: '9', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '9', result: 'asking' })
  })

  it('says nothing under a token the stream took with it', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core, async () => {})
    const note = createConflict('Note.md')
    note.raise(going.raise)
    void going.start()

    mockStream.send({ token: '2', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '2', result: 'asking' }])

    mockStream.end()
    await settle()
    note.answered()
    await settle()

    // Nothing answers to that token now, and the page has not been asked again.
    expect(fakeCore.reportedResults).toEqual([{ token: '2', result: 'asking' }])

    mockStream.send({ token: '3', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '3', result: 'written' })
  })

  it('raises what stands again under the token it is asked under next', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = useFileFlush(fakeCore.core, async () => {})
    createConflict('Note.md').raise(going.raise)
    void going.start()

    mockStream.send({ token: '9', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '9', result: 'asking' }])

    // The stream drops and the page listens again under a new token.
    mockStream.end()
    await settle()
    mockStream.send({ token: '10', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '10', result: 'asking' })
  })
})
