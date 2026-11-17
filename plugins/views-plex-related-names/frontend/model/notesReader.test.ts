/**
 * The reader that the views of the plex share: one read of the notes related to a note, however many
 * views show it, kept while a view uses it and read again when the vault changes it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { shallowRef } from 'vue'
import type { Disposable, NoteGraph, VaultChangeEvent } from '@numen/plugin-sdk'
import { createNotesReader, type NoteLease } from './notesReader'

const PEAR = 'notes/Pear rows.md'
const QUINCE = 'notes/Quince.md'
const SETTLE_MS = 5000

const createGraph = (path: string, related: readonly string[]): NoteGraph => ({
  focus: { path, title: path },
  focusType: 'note',
  related: related.map((one) => ({ path: one, title: one, relation: 'child', type: 'note' })),
  connections: [],
})

interface Pending {
  readonly path: string
  readonly depth: number | undefined
  resolve(answer: NoteGraph): void
  reject(error: unknown): void
}

/** A vault whose reads stay open until the test answers them, one by one. */
const createRig = (showNames = true) => {
  const pending: Pending[] = []
  const handlers = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const reported: unknown[] = []
  const settings = { showNames: shallowRef(showNames) }
  const reader = createNotesReader({
    notes: {
      listRelatedNotes: (path: string, depth?: 1 | 2) =>
        new Promise<NoteGraph>((resolve, reject) => {
          pending.push({ path, depth, resolve, reject })
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
  const getReads = (path?: string) =>
    path === undefined ? pending : pending.filter((one) => one.path === path)
  const answered = new Set<Pending>()
  const answer = async (path: string, related: readonly string[]) => {
    getReads(path)
      .filter((one) => !answered.has(one))
      .forEach((one) => {
        answered.add(one)
        one.resolve(createGraph(path, related))
      })
    await flushPromises()
  }
  const emit = async (event: VaultChangeEvent) => {
    await Promise.all(
      [...handlers].map(async (one) => {
        await one(event)
      }),
    )
  }
  const settle = async () => {
    await vi.advanceTimersByTimeAsync(SETTLE_MS)
    await flushPromises()
  }
  return { reader, settings, getReads, answer, emit, settle, reported, handlers }
}

const getRelatedPaths = (lease: NoteLease) => lease.noteGraph.value?.related.map((one) => one.path)

afterEach(() => {
  vi.useRealTimers()
})

describe('the reader of the related notes', () => {
  it('reads once a note that two views ask for before the answer arrives', () => {
    const { reader, getReads } = createRig()

    reader.acquire(PEAR)
    reader.acquire(PEAR)

    expect(getReads(PEAR)).toHaveLength(1)
  })

  it('reads the notes related to the related notes of a note, with depth 2', () => {
    const { reader, getReads } = createRig()

    reader.acquire(PEAR)

    expect(getReads(PEAR)[0]?.depth).toBe(2)
  })

  it('gives both views the one answer', async () => {
    const { reader, answer } = createRig()
    const above = reader.acquire(PEAR)
    const below = reader.acquire(PEAR)

    await answer(PEAR, ['notes/Rootstock list.md'])

    expect(getRelatedPaths(above)).toStrictEqual(['notes/Rootstock list.md'])
    expect(getRelatedPaths(below)).toStrictEqual(['notes/Rootstock list.md'])
  })

  it('gives a view that asks after the answer came that answer without reading again', async () => {
    const { reader, answer, getReads } = createRig()
    reader.acquire(PEAR)
    await answer(PEAR, ['notes/Rootstock list.md'])

    const late = reader.acquire(PEAR)

    expect(getRelatedPaths(late)).toStrictEqual(['notes/Rootstock list.md'])
    expect(getReads(PEAR)).toHaveLength(1)
  })

  it('starts the reads of several notes in the same tick, without waiting for one another', () => {
    const { reader, getReads } = createRig()
    const paths = Array.from({ length: 5 }, (_, index) => `notes/Leaf ${index + 1}.md`)

    paths.forEach((one) => reader.acquire(one))

    expect(getReads().map((one) => one.path)).toStrictEqual(paths)
  })

  it('answers each note with its own notes', async () => {
    const { reader, answer } = createRig()
    const pear = reader.acquire(PEAR)
    const quince = reader.acquire(QUINCE)

    await answer(QUINCE, ['notes/Jam.md'])
    await answer(PEAR, ['notes/Rootstock list.md'])

    expect(getRelatedPaths(pear)).toStrictEqual(['notes/Rootstock list.md'])
    expect(getRelatedPaths(quince)).toStrictEqual(['notes/Jam.md'])
  })

  it('reads nothing for a note without a path', () => {
    const { reader, getReads } = createRig()

    const lease = reader.acquire('')

    expect(getReads()).toHaveLength(0)
    expect(lease.noteGraph.value).toBeNull()
  })
})

describe('the reader while views come and go', () => {
  it('keeps the answer while one of two views still uses it', async () => {
    const { reader, answer, getReads } = createRig()
    const first = reader.acquire(PEAR)
    reader.acquire(PEAR)
    await answer(PEAR, ['notes/Rootstock list.md'])

    first.release()
    const third = reader.acquire(PEAR)

    expect(getRelatedPaths(third)).toStrictEqual(['notes/Rootstock list.md'])
    expect(getReads(PEAR)).toHaveLength(1)
  })

  it('forgets the answer when the last view that used it is gone', async () => {
    const { reader, answer, getReads } = createRig()
    const first = reader.acquire(PEAR)
    const second = reader.acquire(PEAR)
    await answer(PEAR, ['notes/Rootstock list.md'])

    first.release()
    second.release()
    const later = reader.acquire(PEAR)

    expect(later.noteGraph.value).toBeNull()
    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('reads the note again for a view that asks after the view that began the read has left', () => {
    const { reader, getReads } = createRig()
    reader.acquire(PEAR).release()

    reader.acquire(PEAR)

    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('does not read again for a change of a note that no view uses any more', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, answer, emit, settle, getReads } = createRig()
    const lease = reader.acquire(PEAR)
    await answer(PEAR, [])
    lease.release()

    await emit({ paths: [PEAR] })
    await settle()

    expect(getReads(PEAR)).toHaveLength(1)
  })

  it('drops a change that is waiting when the last view that used the note is gone', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, answer, emit, settle, getReads } = createRig()
    const lease = reader.acquire(PEAR)
    await answer(PEAR, [])

    await emit({ paths: [PEAR] })
    lease.release()
    await settle()

    expect(getReads(PEAR)).toHaveLength(1)
  })

  it('keeps no entry for a note that was released twice', async () => {
    const { reader, answer, getReads } = createRig()
    const first = reader.acquire(PEAR)
    const second = reader.acquire(PEAR)
    await answer(PEAR, [])

    first.release()
    first.release()
    const third = reader.acquire(PEAR)

    expect(getReads(PEAR)).toHaveLength(1)
    second.release()
    third.release()
    reader.acquire(PEAR)
    expect(getReads(PEAR)).toHaveLength(2)
  })
})

describe('the reader while the vault changes', () => {
  const setupRead = async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const rig = createRig()
    const above = rig.reader.acquire(PEAR)
    const below = rig.reader.acquire(PEAR)
    await rig.answer(PEAR, ['notes/Rootstock list.md'])
    return { ...rig, above, below }
  }

  it('reads a changed note once again for both views', async () => {
    const { emit, settle, getReads } = await setupRead()

    await emit({ paths: [PEAR] })
    await settle()

    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('gives both views the answer of the new read', async () => {
    const { emit, settle, answer, above, below } = await setupRead()

    await emit({ paths: [PEAR] })
    await settle()
    await answer(PEAR, ['notes/Rootstock list.md', 'notes/Compost.md'])

    expect(getRelatedPaths(above)).toStrictEqual(['notes/Rootstock list.md', 'notes/Compost.md'])
    expect(getRelatedPaths(below)).toStrictEqual(['notes/Rootstock list.md', 'notes/Compost.md'])
  })

  it('reads the note again when a related note of it is the one that changed', async () => {
    const { emit, settle, getReads } = await setupRead()

    await emit({ paths: ['notes/Rootstock list.md'] })
    await settle()

    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('reads the note once for a burst of changes', async () => {
    const { emit, settle, getReads } = await setupRead()

    for (let index = 0; index < 5; index += 1) await emit({ paths: [PEAR] })
    await settle()

    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('reads nothing again for a change of a note unrelated to the note', async () => {
    const { emit, settle, getReads } = await setupRead()

    await emit({ paths: ['notes/Beehives.md'] })
    await settle()

    expect(getReads()).toHaveLength(1)
  })

  it('reads every note in use once for a change that does not say which notes changed', async () => {
    const { reader, answer, emit, settle, getReads } = await setupRead()
    reader.acquire(QUINCE)
    await answer(QUINCE, [])

    await emit({})
    await settle()

    expect(getReads(PEAR)).toHaveLength(2)
    expect(getReads(QUINCE)).toHaveLength(2)
  })

  it('reads again only the note that changed, not the other notes in use', async () => {
    const { reader, answer, emit, settle, getReads } = await setupRead()
    reader.acquire(QUINCE)
    await answer(QUINCE, ['notes/Jam.md'])

    await emit({ paths: [PEAR] })
    await settle()

    expect(getReads(PEAR)).toHaveLength(2)
    expect(getReads(QUINCE)).toHaveLength(1)
  })

  it('leaves the old answer in place while the new read is open', async () => {
    const { emit, settle, above } = await setupRead()

    await emit({ paths: [PEAR] })
    await settle()

    expect(getRelatedPaths(above)).toStrictEqual(['notes/Rootstock list.md'])
  })

  it('follows the vault through one subscription, however many views use notes', () => {
    const { reader, handlers } = createRig()

    reader.acquire(PEAR)
    reader.acquire(PEAR)
    reader.acquire(QUINCE)

    expect(handlers.size).toBe(1)
  })

  it('stops following the vault and reads nothing more once it is disposed', async () => {
    const { reader, answer, emit, settle, getReads, handlers } = await setupRead()
    reader.acquire(QUINCE)
    await answer(QUINCE, [])

    await emit({ paths: [PEAR] })
    reader.dispose()
    await settle()

    expect(handlers.size).toBe(0)
    expect(getReads()).toHaveLength(2)
  })
})

describe('the reader while the names are hidden', () => {
  it('reads nothing for a view that asks', () => {
    const { reader, getReads } = createRig(false)

    const lease = reader.acquire(PEAR)

    expect(getReads()).toHaveLength(0)
    expect(lease.noteGraph.value).toBeNull()
  })

  it('reads each note in use once when the names are shown', async () => {
    const { reader, settings, getReads } = createRig(false)
    reader.acquire(PEAR)
    reader.acquire(PEAR)
    reader.acquire(QUINCE)

    settings.showNames.value = true
    await flushPromises()

    expect(getReads(PEAR)).toHaveLength(1)
    expect(getReads(QUINCE)).toHaveLength(1)
  })

  it('takes the answer away when the names are hidden', async () => {
    const { reader, settings, answer } = createRig()
    const lease = reader.acquire(PEAR)
    await answer(PEAR, ['notes/Rootstock list.md'])

    settings.showNames.value = false
    await flushPromises()

    expect(lease.noteGraph.value).toBeNull()
  })

  it('reads nothing on a change of the vault', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, settings, answer, emit, settle, getReads } = createRig()
    reader.acquire(PEAR)
    await answer(PEAR, [])
    settings.showNames.value = false
    await flushPromises()

    await emit({ paths: [PEAR] })
    await settle()

    expect(getReads()).toHaveLength(1)
  })
})

describe('the reader when a read fails', () => {
  it('reports a failed read once for the two views that use the note', async () => {
    const { reader, getReads, reported } = createRig()
    reader.acquire(PEAR)
    reader.acquire(PEAR)

    getReads(PEAR)[0]?.reject(new Error('the vault is closed'))
    await flushPromises()

    expect(reported).toHaveLength(1)
    expect(reported[0]).toStrictEqual(new Error('the vault is closed'))
  })

  it('gives the views no names after the failure', async () => {
    const { reader, getReads } = createRig()
    const lease = reader.acquire(PEAR)

    getReads(PEAR)[0]?.reject(new Error('the vault is closed'))
    await flushPromises()

    expect(lease.noteGraph.value).toBeNull()
  })

  it('reads the note again for a view that asks after the failure', async () => {
    const { reader, getReads } = createRig()
    reader.acquire(PEAR)
    getReads(PEAR)[0]?.reject(new Error('the vault is closed'))
    await flushPromises()

    reader.acquire(PEAR)

    expect(getReads(PEAR)).toHaveLength(2)
  })

  it('reports once and shows no names when the read of a change fails', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { reader, answer, emit, settle, getReads, reported } = createRig()
    const lease = reader.acquire(PEAR)
    await answer(PEAR, ['notes/Rootstock list.md'])

    await emit({ paths: [PEAR] })
    await settle()
    getReads(PEAR)[1]?.reject(new Error('the vault is closed'))
    await flushPromises()

    expect(reported).toHaveLength(1)
    expect(lease.noteGraph.value).toBeNull()
  })
})

describe('the reader when a note is read again before the first read is answered', () => {
  const setupTwoReads = async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const rig = createRig()
    const lease = rig.reader.acquire(PEAR)
    await rig.emit({})
    await rig.settle()
    return { ...rig, lease }
  }

  it('shows the answer of the later read when the earlier read answers after it', async () => {
    const { getReads, lease } = await setupTwoReads()
    const [earlier, later] = getReads(PEAR)

    later?.resolve(createGraph(PEAR, ['notes/Compost.md']))
    await flushPromises()
    earlier?.resolve(createGraph(PEAR, ['notes/Rootstock list.md']))
    await flushPromises()

    expect(getRelatedPaths(lease)).toStrictEqual(['notes/Compost.md'])
  })

  it('does not report the failure of an earlier read that the later one replaced', async () => {
    const { getReads, lease, reported } = await setupTwoReads()
    const [earlier, later] = getReads(PEAR)

    later?.resolve(createGraph(PEAR, ['notes/Compost.md']))
    await flushPromises()
    earlier?.reject(new Error('the vault is closed'))
    await flushPromises()

    expect(reported).toHaveLength(0)
    expect(getRelatedPaths(lease)).toStrictEqual(['notes/Compost.md'])
  })
})
