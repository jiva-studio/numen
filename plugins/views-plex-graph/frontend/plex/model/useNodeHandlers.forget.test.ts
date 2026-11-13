/** The listeners of a note that has left the frame are dropped. */
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useNodeHandlers, type NodeReports } from './useNodeHandlers'

const createReports = (): NodeReports => ({
  activate: vi.fn(),
  show: vi.fn(),
  reach: vi.fn(),
  ask: vi.fn(),
  menu: vi.fn(),
  enter: vi.fn(),
  attend: vi.fn(),
})

describe('the listeners of a note that left the frame', () => {
  it('are made anew when the note comes back, and stay the same while it is drawn', async () => {
    const live = reactive({ ids: ['a', 'b'] })
    const getListeners = useNodeHandlers(createReports(), () => live.ids)
    const first = getListeners('a')
    const other = getListeners('b')

    live.ids = ['b']
    await nextTick()
    expect(getListeners('b')).toBe(other)

    live.ids = ['a', 'b']
    await nextTick()
    expect(getListeners('a')).not.toBe(first)
  })
})
