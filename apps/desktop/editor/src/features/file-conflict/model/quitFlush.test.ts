/**
 * The path by which quitting flushes a buffer that only the page holds.
 *
 * The guarded failure is silent: the window closes, the application reports
 * that everything was written, and the last seconds of typing are lost.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/core-client'

import { useOpenNotes, flattenOpenNotes } from '@/entities/note'
import { answerNotes, type NoteAnswer } from '@/testing/noteCore'
import { createFileFlush, type Conflict, type FlushDeps } from './flush'
import { createNoteFlushHandler } from './noteFlush'

/** The note calls of the core the test answered last. */
let notesApi = answerNotes({})

const openNotes = (how?: Parameters<typeof useOpenNotes>[1]) =>
  flattenOpenNotes(useOpenNotes(notesApi, how))

type FlushResult = NonNullable<Parameters<FlushDeps['reportFlush']>[1]>

function rethrow(error: unknown): never {
  throw error
}

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
        await new Promise<void>((awakened) => {
          wake = awakened
        })
      }
    },
  }
}

function createFakeCore(getQuits: () => AsyncIterable<{ token: string; flush: boolean }>) {
  const files = new Map<string, string>()
  const writtenFiles: { path: string; body: string }[] = []
  const reportedResults: { token: string; result: FlushResult }[] = []
  /** Writes wait here until a test lets them through. */
  let pendingWriteResolve: (() => void) | null = null

  const core: FlushDeps = {
    watchQuit: getQuits,
    reportFlush: async (token: string, result: FlushResult = 'written') => {
      reportedResults.push({ token, result })
    },
  }
  notesApi = answerNotes({
    read: async (path): Promise<NoteAnswer> =>
      files.has(path) ? asValue({ body: files.get(path) ?? '' }) : asFailure('notFound'),
    write: async (path, body): Promise<NoteAnswer> => {
      if (pendingWriteResolve) await new Promise<void>((through) => (pendingWriteResolve = through))
      writtenFiles.push({ path, body })
      files.set(path, body)
      return asValue({ body: '' })
    },
  })
  return {
    core,
    files,
    writtenFiles,
    reportedResults,
    holdWrites: () => {
      pendingWriteResolve = vi.fn()
    },
    releaseWrites: () => {
      const resolveWrite = pendingWriteResolve
      pendingWriteResolve = null
      resolveWrite?.()
    },
  }
}

function createConflict(note: string) {
  const actions: string[] = []
  let drop: () => void = vi.fn()
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
    dismiss: () => drop(),
  }
}

const delay = () => new Promise((resolve) => setTimeout(resolve, 0))
const settle = async () => {
  for (let i = 0; i < 20; i++) await delay()
}

describe('a page requested to write its unsaved changes', () => {
  it('writes an unsaved tab and only then says it has', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    going.start().catch(rethrow)

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

  it('does not answer while its pending write is still in the air', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    going.start().catch(rethrow)

    notes.open('Note.md')
    await settle()
    notes.setBody('Note.md', 'member')
    fakeCore.holdWrites()

    mockStream.send({ token: '1', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([])

    fakeCore.releaseWrites()
    await settle()

    expect(fakeCore.writtenFiles).toEqual([{ path: 'Note.md', body: 'member' }])
    expect(fakeCore.reportedResults).toEqual([{ token: '1', result: 'written' }])
  })

  it('says nothing until the application asks', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    going.start().catch(rethrow)

    notes.open('Note.md')
    await settle()
    notes.setBody('Note.md', 'still being written')

    // The stream opens with a token and no flush request.
    mockStream.send({ token: '3', flush: false })
    await settle()

    expect(fakeCore.writtenFiles).toEqual([])
    expect(fakeCore.reportedResults).toEqual([])
  })

  it('answers for a window with nothing open', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(openNotes()))
    going.start().catch(rethrow)

    mockStream.send({ token: '0', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '0', result: 'written' }])
  })
})

describe('a page holding text of a note whose file was deleted', () => {
  async function createGoneNote() {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    going.start().catch(rethrow)

    fakeCore.files.set('Gone.md', 'what was saved')
    notes.open('Gone.md')
    await settle()
    fakeCore.files.delete('Gone.md')
    notes.applyPathChanges(['Gone.md'])
    await settle()
    notes.setBody('Gone.md', 'typed after the file went')
    return { mockStream, fakeCore, notes, going }
  }

  it('stops the quit and names the note, writing nothing by itself', async () => {
    const { mockStream, fakeCore, notes, going } = await createGoneNote()

    mockStream.send({ token: '1', flush: true })
    await settle()

    expect(fakeCore.writtenFiles.filter((one) => one.path === 'Gone.md')).toEqual([])
    expect(fakeCore.reportedResults).toEqual([{ token: '1', result: 'unsaved' }])
    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Gone.md'])
    expect(await notes.close('Gone.md')).toBe(false)
    expect(notes.getOpenNote('Gone.md').body).toBe('typed after the file went')
  })

  it('writes the text as the file again when the person keeps it, and then lets the quit go', async () => {
    const { mockStream, fakeCore, going } = await createGoneNote()
    mockStream.send({ token: '1', flush: true })
    await settle()

    await going.conflicts.value[0]?.keep()
    await settle()

    expect(fakeCore.writtenFiles.filter((one) => one.path === 'Gone.md')).toEqual([
      { path: 'Gone.md', body: 'typed after the file went' },
    ])
    expect(going.conflicts.value).toEqual([])
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '1', result: 'written' })
  })

  it('drops the text when the person discards it, and then lets the quit go', async () => {
    const { mockStream, fakeCore, notes, going } = await createGoneNote()
    mockStream.send({ token: '1', flush: true })
    await settle()

    await going.conflicts.value[0]?.take()
    await settle()

    expect(fakeCore.writtenFiles.filter((one) => one.path === 'Gone.md')).toEqual([])
    expect(notes.getOpenNote('Gone.md').body).toBe('what was saved')
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '1', result: 'written' })
  })
})

describe('a page holding text the file changed under', () => {
  it('says there are conflicts outstanding, and does not say it has written', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(fakeCore.core)
    const note = createConflict('Note.md')
    note.raise(going.raise)
    going.start().catch(rethrow)

    mockStream.send({ token: '4', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '4', result: 'unsaved' }])
    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Note.md'])
  })

  it('says so before its pending writes have landed', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    const note = createConflict('Held.md')
    note.raise(going.raise)

    going.start().catch(rethrow)
    notes.open('Other.md')
    await settle()
    notes.setBody('Other.md', 'on its way')
    fakeCore.holdWrites()

    mockStream.send({ token: '5', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '5', result: 'unsaved' }])

    fakeCore.releaseWrites()
    await settle()

    // The write succeeded and the conflict remains, so the reported result is
    // unchanged.
    expect(fakeCore.writtenFiles).toEqual([{ path: 'Other.md', body: 'on its way' }])
    expect(fakeCore.reportedResults).toEqual([{ token: '5', result: 'unsaved' }])
  })

  it('says it has written once every conflict is settled', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(fakeCore.core)
    const first = createConflict('One.md')
    const second = createConflict('Two.md')
    first.raise(going.raise)
    second.raise(going.raise)
    going.start().catch(rethrow)

    mockStream.send({ token: '6', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '6', result: 'unsaved' }])

    await going.conflicts.value.find((one) => one.note === 'One.md')?.keep()
    await settle()

    // One of the two conflicts is resolved, so the result is still pending.
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '6', result: 'unsaved' })
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
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    createConflict('Later.md').raise(going.raise)
    going.start().catch(rethrow)

    mockStream.send({ token: '8', flush: true })
    await settle()

    going.conflicts.value[0]?.later()
    await settle()

    expect(going.conflicts.value).toEqual([])
    // The conflict is not resolved, so the result is still pending.
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '8', result: 'unsaved' })

    // The result is still pending on the next flush request.
    mockStream.end()
    await settle()
    mockStream.send({ token: '9', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '9', result: 'unsaved' })
  })

  it('says nothing under a token the stream took with it', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    const note = createConflict('Note.md')
    note.raise(going.raise)
    going.start().catch(rethrow)

    mockStream.send({ token: '2', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '2', result: 'unsaved' }])

    mockStream.end()
    await settle()
    note.dismiss()
    await settle()

    // No result is reported for that token now, and the page has not been requested again.
    expect(fakeCore.reportedResults).toEqual([{ token: '2', result: 'unsaved' }])

    mockStream.send({ token: '3', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '3', result: 'written' })
  })

  it('raises what remains again under the token it is requested under next', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    createConflict('Note.md').raise(going.raise)
    going.start().catch(rethrow)

    mockStream.send({ token: '9', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '9', result: 'unsaved' }])

    // The stream drops and the page reconnects under a new token.
    mockStream.end()
    await settle()
    mockStream.send({ token: '10', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '10', result: 'unsaved' })
  })
})

describe('the conflicts a person put off', () => {
  const getNotes = (going: ReturnType<typeof createFileFlush>) =>
    going.conflicts.value.map((one) => one.note)

  it('are drawn again when the application asks again on the same stream', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    createConflict('Later.md').raise(going.raise)
    going.start().catch(rethrow)
    mockStream.send({ token: '1', flush: true })
    await settle()
    going.conflicts.value[0]?.later()

    mockStream.send({ token: '2', flush: true })
    await settle()

    expect(getNotes(going)).toEqual(['Later.md'])
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '2', result: 'unsaved' })
  })

  it('stay off the screen while other conflicts are raised, and the others stay drawn', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    createConflict('A.md').raise(going.raise)
    createConflict('B.md').raise(going.raise)
    going.start().catch(rethrow)
    mockStream.send({ token: '1', flush: true })
    await settle()

    going.conflicts.value.find((one) => one.note === 'A.md')?.later()

    expect(getNotes(going)).toEqual(['B.md'])

    createConflict('C.md').raise(going.raise)
    await settle()

    expect(getNotes(going)).toEqual(['B.md', 'C.md'])
  })
})

describe('a conflict put off and settled', () => {
  it('is drawn again when the same note raises another', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    const first = createConflict('Again.md')
    first.raise(going.raise)
    going.start().catch(rethrow)
    mockStream.send({ token: '1', flush: true })
    await settle()
    going.conflicts.value[0]?.later()
    first.dismiss()

    createConflict('Again.md').raise(going.raise)
    await settle()

    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Again.md'])
  })
})

describe('the handlers of a page', () => {
  it('are no longer waited for once removed', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    let calls = 0
    const remove = going.addHandler(async () => {
      calls += 1
    })
    going.start().catch(rethrow)

    remove()
    mockStream.send({ token: '1', flush: true })
    await settle()

    expect(calls).toBe(0)
  })
})

describe('writes pending at an earlier request', () => {
  it('do not answer for the ask that came after them', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(
      fakeCore.core,
      vi.fn(() => Promise.resolve()),
    )
    const gates: (() => void)[] = []
    going.addHandler(() => new Promise<void>((isDone) => gates.push(isDone)))
    going.start().catch(rethrow)
    mockStream.send({ token: '1', flush: true })
    await settle()
    mockStream.send({ token: '2', flush: true })
    await settle()

    gates[0]?.()
    await settle()

    expect(fakeCore.reportedResults).toEqual([])

    gates[1]?.()
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '2', result: 'written' }])
  })
})

describe('a page that is closed', () => {
  /** A core whose quit stream is the mock one, and which keeps what it was given to listen with. */
  const createListeningCore = () => {
    const mockStream = createMockStream()
    const signals: AbortSignal[] = []
    const core: FlushDeps = {
      watchQuit: (signal) => {
        signals.push(signal)
        return mockStream.read()
      },
      reportFlush: vi.fn(() => Promise.resolve()),
    }
    return { mockStream, signals, core }
  }

  it('releases the stream it listens on', async () => {
    const { signals, core } = createListeningCore()
    const going = createFileFlush(
      core,
      vi.fn(() => Promise.resolve()),
    )
    going.start().catch(rethrow)
    await settle()

    going.close()

    expect(signals.map((one) => one.aborted)).toEqual([true])
  })

  it('does not listen again once the stream has ended', async () => {
    const { mockStream, signals, core } = createListeningCore()
    const going = createFileFlush(
      core,
      vi.fn(() => Promise.resolve()),
    )
    going.start().catch(rethrow)
    await settle()

    going.close()
    mockStream.end()
    await settle()

    expect(signals).toHaveLength(1)
  })
})

describe('a quit stream that ended', () => {
  afterEach(() => vi.useRealTimers())

  it('is listened to again after the wait the page keeps by default', async () => {
    vi.useFakeTimers()
    let listens = 0
    const core: FlushDeps = {
      watchQuit: async function* () {
        listens += 1
        yield* []
      },
      reportFlush: vi.fn(() => Promise.resolve()),
    }
    const going = createFileFlush(core)
    going.start().catch(rethrow)
    await vi.advanceTimersByTimeAsync(0)
    expect(listens).toBe(1)

    await vi.advanceTimersByTimeAsync(1000)

    expect(listens).toBe(2)
    going.close()
  })
})

describe('a page holding text the vault would not take', () => {
  /** A vault whose writes fail as long as `refusing` is set. */
  const createRefusingVault = (refusing: { code: string | null }) => {
    const files = new Map<string, string>([['Tides.md', 'high tide at six']])
    notesApi = answerNotes({
      read: async (path): Promise<NoteAnswer> =>
        asValue({ body: files.get(path) ?? '', at: `1 1 at:${files.get(path) ?? ''}` }),
      write: async (path, body): Promise<NoteAnswer> => {
        if (refusing.code !== null) return asFailure(refusing.code)
        files.set(path, body)
        return asValue({ body: '', at: `1 1 at:${body}` })
      },
    })
    return files
  }

  const quitWith = async (code: string) => {
    const refusing = { code: code as string | null }
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const files = createRefusingVault(refusing)
    const notes = openNotes()
    const going = createFileFlush(fakeCore.core)
    going.addHandler(createNoteFlushHandler(notes))
    going.start().catch(rethrow)
    notes.open('Tides.md')
    await settle()
    notes.setBody('Tides.md', 'high tide at six, low at noon')
    // The first save fails by itself, as it would in the minutes before the quit.
    notes.save('Tides.md')
    await settle()
    mockStream.send({ token: '1', flush: true })
    await settle()
    return { refusing, files, mockStream, fakeCore, notes, going }
  }

  it.each(['tooLarge', 'bodyUnwritable', 'unreachable'])(
    'does not say it has written when the vault answers %s',
    async (code) => {
      const { fakeCore, going } = await quitWith(code)

      expect(fakeCore.reportedResults).toEqual([{ token: '1', result: 'unsaved' }])
      expect(going.conflicts.value.map((one) => one.note)).toEqual(['Tides.md'])
    },
  )

  it('says it has written once the vault takes the text on another try', async () => {
    const { refusing, files, mockStream, fakeCore, going } = await quitWith('unreachable')

    refusing.code = null
    await going.conflicts.value[0]?.keep()
    await settle()

    expect(files.get('Tides.md')).toBe('high tide at six, low at noon')
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '1', result: 'written' })
    expect(going.conflicts.value).toEqual([])
    mockStream.end()
  })

  it('says it has written once the person gives the text up', async () => {
    const { files, fakeCore, notes, going } = await quitWith('tooLarge')

    await going.conflicts.value[0]?.take()
    await settle()

    expect(files.get('Tides.md')).toBe('high tide at six')
    expect(notes.getOpenNote('Tides.md').body).toBe('high tide at six')
    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '1', result: 'written' })
  })

  it('stays unsaved when the retry fails again', async () => {
    const { fakeCore, going } = await quitWith('unreachable')

    await going.conflicts.value[0]?.keep()
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '1', result: 'unsaved' })
    expect(going.conflicts.value.map((one) => one.note)).toEqual(['Tides.md'])
  })

  it('asks again at the next quit, and does not drop the text', async () => {
    const { mockStream, fakeCore, notes } = await quitWith('unreachable')

    mockStream.send({ token: '2', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '2', result: 'unsaved' })
    expect(notes.getOpenNote('Tides.md').body).toBe('high tide at six, low at noon')
  })
})

describe('a handler that fails', () => {
  it('leaves the quit unsaved instead of written', async () => {
    const mockStream = createMockStream()
    const fakeCore = createFakeCore(mockStream.read)
    const going = createFileFlush(fakeCore.core)
    let calls = 0
    going.addHandler(async () => {
      calls += 1
      if (calls === 1) throw new Error('the buffer would not write')
    })
    going.start().catch(rethrow)

    mockStream.send({ token: '1', flush: true })
    await settle()

    expect(fakeCore.reportedResults).toEqual([{ token: '1', result: 'unsaved' }])

    mockStream.send({ token: '2', flush: true })
    await settle()

    expect(fakeCore.reportedResults.at(-1)).toEqual({ token: '2', result: 'written' })
  })
})
