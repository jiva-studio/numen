/**
 * The reader reads the notes that views ask for in the same tick with one call of the vault, and
 * the notes that views ask for later with another.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { shallowRef } from 'vue'
import type { Disposable, NoteGraph, VaultChangeEvent } from '@numen/plugin-sdk'
import { createNotesReader, type NoteLease } from './notesReader'

const PEAR = 'notes/Pear rows.md'
const QUINCE = 'notes/Quince.md'
const MEDLAR = 'notes/Medlar.md'
const SETTLE_MS = 5000
const MOST_PATHS_PER_CALL = 64

const UNKNOWN: NoteGraph = {
  focus: { path: '', title: '' },
  focusType: 'note',
  related: [],
  connections: [],
}

const getRelatedOf = (path: string) => `${path}/related.md`

const createGraph = (path: string): NoteGraph => ({
  focus: { path, title: path },
  focusType: 'note',
  related: [
    { path: getRelatedOf(path), title: 'related', relation: 'child' as const, type: 'note' },
  ],
  connections: [],
})

interface Call {
  readonly paths: readonly string[]
  readonly depth: number | undefined
  resolve(graphs: readonly NoteGraph[]): void
  reject(error: unknown): void
}

const createRig = (showNames = true) => {
  const calls: Call[] = []
  const handlers = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const reported: unknown[] = []
  const settings = { showNames: shallowRef(showNames) }
  const reader = createNotesReader({
    notes: {
      listRelatedNotes: (paths: readonly string[], depth?: 1 | 2) =>
        new Promise<readonly NoteGraph[]>((resolve, reject) => {
          calls.push({ paths, depth, resolve, reject })
        }),
    },
    events: {
      onVaultChange: (handler): Disposable => {
        handlers.add(handler)
        return {
          dispose: () => {
            handlers.delete(handler)
          },
        }
      },
    },
    settings,
    reportFailure: (error) => reported.push(error),
  })
  const answerCall = async (call: Call) => {
    call.resolve(call.paths.map((path) => createGraph(path)))
    await flushPromises()
  }
  const emit = async (event: VaultChangeEvent) => {
    await Promise.all([...handlers].map(async (one) => one(event)))
  }
  const settle = async () => {
    await vi.advanceTimersByTimeAsync(SETTLE_MS)
    await flushPromises()
  }
  return { reader, settings, calls, answerCall, emit, settle, reported }
}

const getRelatedPaths = (lease: NoteLease) => lease.noteGraph.value?.related.map((one) => one.path)

afterEach(() => {
  vi.useRealTimers()
})

describe('the reads of the notes that views ask for in one tick', () => {
  it('reads the notes that views ask for in the same tick with one call holding all their paths', async () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    reader.acquire(MEDLAR)
    await flushPromises()

    expect(calls.map((one) => one.paths)).toStrictEqual([[PEAR, QUINCE, MEDLAR]])
  })

  it('makes no call before the tick that the views ask in has ended', () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR)
    reader.acquire(QUINCE)

    expect(calls).toHaveLength(0)
  })

  it('reads the notes that views ask for in two ticks with two calls', async () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    await flushPromises()
    reader.acquire(MEDLAR)
    await flushPromises()

    expect(calls.map((one) => one.paths)).toStrictEqual([[PEAR, QUINCE], [MEDLAR]])
  })

  it('puts a note that two views ask for in one tick once in the call', async () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    reader.acquire(PEAR)
    await flushPromises()

    expect(calls.map((one) => one.paths)).toStrictEqual([[PEAR, QUINCE]])
  })

  it('reads the notes with depth 1', async () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    await flushPromises()

    expect(calls.map((one) => one.depth)).toStrictEqual([1])
  })

  it('leaves out of the call a note whose only view left before the call', async () => {
    const { reader, calls } = createRig()

    const gone = reader.acquire(PEAR)
    reader.acquire(QUINCE)
    gone.release()
    await flushPromises()

    expect(calls.map((one) => one.paths)).toStrictEqual([[QUINCE]])
  })

  it('makes no call when every view left before the call', async () => {
    const { reader, calls } = createRig()

    reader.acquire(PEAR).release()
    await flushPromises()

    expect(calls).toHaveLength(0)
  })
})

describe('the answers of a call', () => {
  it('gives each view the graph at the place of its note in the call', async () => {
    const { reader, calls, answerCall } = createRig()
    const pear = reader.acquire(PEAR)
    const quince = reader.acquire(QUINCE)
    const medlar = reader.acquire(MEDLAR)
    await flushPromises()

    expect(calls).toHaveLength(1)
    await answerCall(calls[0]!)

    expect(getRelatedPaths(pear)).toStrictEqual([getRelatedOf(PEAR)])
    expect(getRelatedPaths(quince)).toStrictEqual([getRelatedOf(QUINCE)])
    expect(getRelatedPaths(medlar)).toStrictEqual([getRelatedOf(MEDLAR)])
  })

  it('gives both views of one note the graph of the note', async () => {
    const { reader, calls, answerCall } = createRig()
    const above = reader.acquire(PEAR)
    const quince = reader.acquire(QUINCE)
    const below = reader.acquire(PEAR)
    await flushPromises()

    expect(calls).toHaveLength(1)
    await answerCall(calls[0]!)

    expect(getRelatedPaths(above)).toStrictEqual([getRelatedOf(PEAR)])
    expect(getRelatedPaths(below)).toStrictEqual([getRelatedOf(PEAR)])
    expect(getRelatedPaths(quince)).toStrictEqual([getRelatedOf(QUINCE)])
  })

  it('clears only the view of a note that is not in the vault', async () => {
    const { reader, calls, reported } = createRig()
    const pear = reader.acquire(PEAR)
    const quince = reader.acquire(QUINCE)
    const medlar = reader.acquire(MEDLAR)
    await flushPromises()

    expect(calls).toHaveLength(1)
    calls[0]!.resolve([createGraph(PEAR), UNKNOWN, createGraph(MEDLAR)])
    await flushPromises()

    expect(getRelatedPaths(pear)).toStrictEqual([getRelatedOf(PEAR)])
    expect(quince.noteGraph.value).toBeNull()
    expect(getRelatedPaths(medlar)).toStrictEqual([getRelatedOf(MEDLAR)])
    expect(reported).toHaveLength(0)
  })

  it('reports a failed call once, however many views wait for it', async () => {
    const { reader, calls, reported } = createRig()
    const leases = [PEAR, QUINCE, MEDLAR].map((one) => reader.acquire(one))
    await flushPromises()

    expect(calls).toHaveLength(1)
    const failure = new Error('the vault is busy')
    calls[0]!.reject(failure)
    await flushPromises()

    expect(reported).toStrictEqual([failure])
    expect(leases.map((one) => one.noteGraph.value)).toStrictEqual([null, null, null])
  })

  it('reads again the notes of a failed call for a view that asks after it', async () => {
    const { reader, calls } = createRig()
    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    await flushPromises()
    expect(calls).toHaveLength(1)
    calls[0]!.reject(new Error('the vault is busy'))
    await flushPromises()

    reader.acquire(PEAR)
    await flushPromises()

    expect(calls.map((one) => one.paths)).toStrictEqual([[PEAR, QUINCE], [PEAR]])
  })

  it('splits the notes of one tick into calls of at most 64 notes and gives each view its own graph', async () => {
    const { reader, calls, answerCall } = createRig()
    const paths = Array.from({ length: 70 }, (_, index) => `notes/Leaf ${index + 1}.md`)
    const leases = paths.map((one) => reader.acquire(one))
    await flushPromises()

    expect(calls.map((one) => one.paths.length)).toStrictEqual([MOST_PATHS_PER_CALL, 6])
    expect(calls.flatMap((one) => one.paths)).toStrictEqual(paths)
    for (const call of calls) await answerCall(call)

    expect(leases.map(getRelatedPaths)).toStrictEqual(paths.map((one) => [getRelatedOf(one)]))
  })
})

describe('the reads that the vault makes again', () => {
  it('reads the changed notes again with one call', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, calls, answerCall, emit, settle } = createRig()
    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    reader.acquire(MEDLAR)
    await flushPromises()
    expect(calls).toHaveLength(1)
    await answerCall(calls[0]!)

    await emit({ paths: [PEAR, MEDLAR] })
    await emit({ paths: [QUINCE] })
    await settle()

    expect(calls.map((one) => [...one.paths].sort())).toStrictEqual([
      [MEDLAR, PEAR, QUINCE].sort(),
      [MEDLAR, PEAR, QUINCE].sort(),
    ])
  })

  it('reads every note in use again with one call for a change that names no note', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, calls, answerCall, emit, settle } = createRig()
    reader.acquire(PEAR)
    reader.acquire(QUINCE)
    await flushPromises()
    expect(calls).toHaveLength(1)
    await answerCall(calls[0]!)

    await emit({})
    await settle()

    expect(calls.map((one) => [...one.paths].sort())).toStrictEqual([
      [PEAR, QUINCE].sort(),
      [PEAR, QUINCE].sort(),
    ])
  })
})
