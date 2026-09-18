// @vitest-environment jsdom
/**
 * A window that closes ends what it started.
 *
 * The streams are HTTP calls that stay open until something ends them. A window
 * unmounted without ending them leaves them running against a page nobody is
 * looking at, and the next message is handled by a screen that is gone.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const taken: AbortSignal[] = []

/** A stream that stays open, so nothing ends on its own. */
async function* waits(): AsyncGenerator<never> {
  await new Promise<never>(() => {})
}

vi.mock('@/shared/clients', () => ({
  WINDOW: 'review',
  cards: {
    watchReloads: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) taken.push(call.signal)
      return waits()
    },
  },
  itself: {
    watchTasks: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) taken.push(call.signal)
      return waits()
    },
  },
  agentService: {
    getAgentState: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) taken.push(call.signal)
      return new Promise<never>(() => {})
    },
  },
}))

describe('what the window follows', () => {
  it('is ended when the window closes', async () => {
    const { useWindowStreams } = await import('./useWindowStreams')
    const host = defineComponent({
      setup() {
        useWindowStreams({
          reportError: () => {},
          setTasks: () => {},
          onKeyDown: () => {},
          count: async () => {},
          stop: () => {},
          refresh: async () => {},
          unreachable: ref(''),
        })
        return () => null
      },
    })

    const window = mount(host)
    // Every call this window has in flight carries a signal, and it is one.
    expect(taken.length).toBe(3)
    expect(new Set(taken).size).toBe(1)
    expect(taken.every((one) => !one.aborted)).toBe(true)

    window.unmount()
    expect(taken.every((one) => one.aborted)).toBe(true)
  })
})
