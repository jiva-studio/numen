import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import { openNotes } from './notes'
import type { Notes } from './types'

describe('openNotes for flashcards studio', () => {
  it('opens note and loads body asynchronously', async () => {
    const notes: Notes = {
      read: vi.fn(async (_path) => asValue({ body: 'deck content', at: 'v1' })),
      write: vi.fn(async () => asValue({ body: '', at: 'v2' })),
    }
    const store = openNotes(notes)

    store.open('tab-1', 'Animals.md')
    expect(store.getOpenIds()).toStrictEqual(['tab-1'])
    expect(store.getPath('tab-1')).toBe('Animals.md')
    expect(store.getOpenNote('tab-1').state).toBe('loading')

    await vi.waitFor(() => {
      expect(store.getOpenNote('tab-1').state).toBe('clean')
    })

    expect(store.getOpenNote('tab-1').body).toBe('deck content')
    expect(store.getFilePath('tab-1')).toBe('v1')
  })

  it('marks error when read fails or throws', async () => {
    const notes: Notes = {
      read: vi.fn(async () => asFailure('notADeck')),
      write: vi.fn(async () => asValue({ body: '', at: '' })),
    }
    const store = openNotes(notes)

    store.open('tab-1', 'bad.md')
    await vi.waitFor(() => {
      expect(store.getOpenNote('tab-1').state).toBe('error')
    })
    expect(store.getOpenNote('tab-1').error).toBe('notANote')

    // Thrown error
    const failingNotes: Notes = {
      read: vi.fn(async () => {
        throw new Error('network down')
      }),
      write: vi.fn(async () => asValue({ body: '', at: '' })),
    }
    const store2 = openNotes(failingNotes)
    store2.open('tab-2', 'down.md')
    await vi.waitFor(() => {
      expect(store2.getOpenNote('tab-2').state).toBe('error')
    })
    expect(store2.getOpenNote('tab-2').error).toBe('unreachable')
  })

  it('transitions to unsaved and writes on flush', async () => {
    const writes: Array<{ path: string; body: string }> = []
    const notes: Notes = {
      read: vi.fn(async () => asValue({ body: 'initial', at: 'v1' })),
      write: vi.fn(async (path, body) => {
        writes.push({ path, body })
        return asValue({ body: '', at: 'v2' })
      }),
    }
    const store = openNotes(notes)
    store.open('tab-1', 'Decks.md')
    await vi.waitFor(() => expect(store.getOpenNote('tab-1').state).toBe('clean'))

    store.setBody('tab-1', 'updated body')
    expect(store.getOpenNote('tab-1').state).toBe('unsaved')

    await store.flushPending('tab-1')
    expect(store.getOpenNote('tab-1').state).toBe('clean')
    expect(writes).toHaveLength(1)
    expect(writes[0]?.body).toBe('updated body')
  })

  it('marks stale on write conflict and handles keepMine and takeFile', async () => {
    let writeCount = 0
    let readCount = 0
    const onReplaced = vi.fn()
    const notes: Notes = {
      read: vi.fn(async () => {
        readCount++
        return asValue({
          body: readCount === 1 ? 'initial' : 'file from disk',
          at: `v${readCount}`,
        })
      }),
      write: vi.fn(async (_path, _body, _observed) => {
        writeCount++
        if (writeCount === 1) {
          return asFailure('changed')
        }
        return asValue({ body: '', at: 'v3' })
      }),
    }

    const store = openNotes(notes, { onReplaced })
    store.open('tab-1', 'Decks.md')
    await vi.waitFor(() => expect(store.getOpenNote('tab-1').state).toBe('clean'))

    store.setBody('tab-1', 'my local edit')
    await store.flush()

    expect(store.getOpenNote('tab-1').state).toBe('stale')
    expect(store.stale('tab-1')).not.toBeNull()

    // When stale, close refuses to drop the tab
    const closedWhileStale = await store.close('tab-1')
    expect(closedWhileStale).toBe(false)
    expect(store.getOpenIds()).toHaveLength(1)

    // keepMine forces write
    store.keep('tab-1')
    await vi.waitFor(() => {
      expect(store.getOpenNote('tab-1').state).toBe('clean')
    })
    expect(writeCount).toBe(2)

    // takeFile re-reads
    store.take('tab-1')
    await vi.waitFor(() => {
      expect(store.getOpenNote('tab-1').body).toBe('file from disk')
    })
    expect(onReplaced).toHaveBeenCalledWith('Decks.md')

    // Now clean, close succeeds
    const closed = await store.close('tab-1')
    expect(closed).toBe(true)
    expect(store.getOpenIds()).toHaveLength(0)
  })

  it('handles renames and external modifications in applyPathChanges', async () => {
    const notes: Notes = {
      read: vi.fn(async () => asValue({ body: 'disk text', at: 'v1' })),
      write: vi.fn(async () => asValue({ body: '', at: 'v2' })),
    }
    const store = openNotes(notes)
    store.open('tab-1', 'Old.md')
    await vi.waitFor(() => expect(store.getOpenNote('tab-1').state).toBe('clean'))

    // Rename
    store.applyPathChanges([], [{ from: 'Old.md', to: 'New.md' }])
    expect(store.getPath('tab-1')).toBe('New.md')

    // Modified on disk when unsaved -> stale
    store.setBody('tab-1', 'unsaved changes')
    store.applyPathChanges(['New.md'])
    expect(store.getOpenNote('tab-1').state).toBe('stale')
  })
})
