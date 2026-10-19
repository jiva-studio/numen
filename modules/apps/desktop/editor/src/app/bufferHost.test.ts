/**
 * The files held open by a plugin's tabs, written out when the window quits.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { openNotes, type FileBuffers } from '@numen/plugin-sdk'
import { createBufferHost } from './bufferHost'

const createFlush = () => {
  const remove = vi.fn(() => true)
  const drop = vi.fn()
  return {
    remove,
    drop,
    fileFlush: { addHandler: vi.fn(() => remove), raise: vi.fn(() => drop) },
  }
}

const createBuffers = () =>
  openNotes({
    read: async () => ({ ok: true, value: { body: 'text', at: '1' } }),
    write: async () => ({ ok: true, value: { body: '', at: '2' } }),
  })

describe('the files a plugin has open', () => {
  it('are written out when the window quits', () => {
    const { fileFlush } = createFlush()
    const buffers = createBuffers()

    createBufferHost({ fileFlush }).track(buffers)

    expect(fileFlush.addHandler).toHaveBeenCalledWith(buffers.flush)
  })

  it('are let go of by the window when the plugin stops tracking them', () => {
    const { fileFlush, remove } = createFlush()

    const tracked = createBufferHost({ fileFlush }).track(createBuffers())
    expect(remove).not.toHaveBeenCalled()
    tracked.dispose()

    expect(remove).toHaveBeenCalledOnce()
  })

  it('put a file that moved under a tab to the person before the window goes', async () => {
    const { fileFlush, drop } = createFlush()
    const states = ref<Record<string, string>>({})
    const stale = {
      flush: vi.fn(),
      getOpenIds: () => Object.keys(states.value),
      getOpenNote: (id: string) => ({ state: states.value[id] ?? 'clean' }),
      keep: vi.fn(),
      take: vi.fn(),
    } as unknown as FileBuffers
    const tracked = createBufferHost({ fileFlush }).track(stale)

    states.value = { tab: 'stale' }
    await nextTick()
    expect(fileFlush.raise).toHaveBeenCalledOnce()

    tracked.dispose()
    expect(drop).toHaveBeenCalledOnce()
  })
})

describe('the work a plugin finishes before the window quits', () => {
  it('is taken by the window, and let go of again', () => {
    const { fileFlush, remove } = createFlush()
    const handler = () => Promise.resolve()

    const added = createBufferHost({ fileFlush }).addFlushHandler(handler)
    expect(fileFlush.addHandler).toHaveBeenCalledWith(handler)
    expect(remove).not.toHaveBeenCalled()

    added.dispose()
    expect(remove).toHaveBeenCalledOnce()
  })
})
