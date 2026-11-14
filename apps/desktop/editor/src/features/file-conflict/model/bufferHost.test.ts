import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { UnsavedBuffersError, type FileBuffers } from '@numen/plugin-loader'
import { useOpenNotes, flattenOpenNotes } from '@/entities/note'
import { createBufferHost } from './bufferHost'
import { createFileFlush } from './flush'

const openNotes = (how?: Parameters<typeof useOpenNotes>[0]) => flattenOpenNotes(useOpenNotes(how))

const createFlush = () => {
  const remove = vi.fn(() => true)
  const drop = vi.fn()
  return {
    remove,
    drop,
    fileFlush: { addHandler: vi.fn(() => remove), raise: vi.fn(() => drop) },
  }
}

const createBuffers = () => openNotes()

describe('the files a plugin has open', () => {
  it('are written out when the window quits', () => {
    const { fileFlush } = createFlush()
    const buffers = createBuffers()

    createBufferHost({ fileFlush }).track(buffers)

    expect(fileFlush.addHandler).toHaveBeenCalledOnce()
  })

  it('are release by the window when the plugin stops tracking them', () => {
    const { fileFlush, remove } = createFlush()

    const tracked = createBufferHost({ fileFlush }).track(createBuffers())
    expect(remove).not.toHaveBeenCalled()
    tracked.dispose()

    expect(remove).toHaveBeenCalledOnce()
  })

  it('put a file that moved under a tab to the person before the window is closed', async () => {
    const { fileFlush, drop } = createFlush()
    const states = ref<Record<string, string>>({})
    const staleBuffers = {
      flush: vi.fn(),
      getOpenIds: () => Object.keys(states.value),
      getOpenNote: (id: string) => ({ state: states.value[id] ?? 'clean' }),
      keep: vi.fn(),
      take: vi.fn(),
    } as unknown as FileBuffers
    const tracked = createBufferHost({ fileFlush }).track(staleBuffers)

    states.value = { tab: 'stale' }
    await nextTick()
    expect(fileFlush.raise).toHaveBeenCalledOnce()

    tracked.dispose()
    expect(drop).toHaveBeenCalledOnce()
  })
})

describe('the text a plugin could not write when the window quits', () => {
  /** A quit is requested once, and the stream then stays open. */
  const requestQuit = () => ({
    watchQuit: async function* () {
      yield { token: '1', flush: true }
      await new Promise(() => undefined)
    },
    reportFlush: vi.fn(() => Promise.resolve()),
  })
  const settle = async () => {
    for (let i = 0; i < 10; i++) await new Promise((done) => setTimeout(done, 0))
  }

  it('is shown to the person by plugin and name, and kept or given up as the plugin says', async () => {
    const quits = requestQuit()
    const fileFlush = createFileFlush(quits)
    const settled: string[] = []
    const flush = () =>
      Promise.reject(
        new UnsavedBuffersError(
          [
            {
              name: 'Decks.md',
              keep: async () => {
                settled.push('keep')
              },
              take: async () => {
                settled.push('take')
              },
            },
          ],
          'md.numen.flashcards-studio',
        ),
      )
    createBufferHost({ fileFlush }).track({ flush })

    fileFlush.start().catch(() => undefined)
    await settle()
    fileFlush.close()

    expect(fileFlush.conflicts.value.map((one) => one.note)).toEqual([
      'md.numen.flashcards-studio: Decks.md',
    ])
    expect(quits.reportFlush).toHaveBeenCalledWith('1', 'unsaved')
    await fileFlush.conflicts.value[0]?.keep()
    expect(settled).toEqual(['keep'])
  })
})

describe('the work a plugin finishes before the window quits', () => {
  it('is taken by the window, and release again', () => {
    const { fileFlush, remove } = createFlush()
    const handler = () => Promise.resolve()

    const added = createBufferHost({ fileFlush }).addFlushHandler(handler)
    expect(fileFlush.addHandler).toHaveBeenCalledOnce()
    expect(remove).not.toHaveBeenCalled()

    added.dispose()
    expect(remove).toHaveBeenCalledOnce()
  })
})
