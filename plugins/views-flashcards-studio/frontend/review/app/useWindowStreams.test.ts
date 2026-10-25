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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { WORDS } from '../pages/session'
import { useWindowStreams } from './useWindowStreams'

const wire = vi.hoisted(() => {
  /** A stream that stays open, so nothing ends on its own. */
  async function* waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  }
  return {
    waits,
    taken: [] as AbortSignal[],
    tasksAsked: [] as unknown[],
    reloadsOpened: 0,
    reloads: (_signal?: AbortSignal): AsyncIterable<unknown> => waits(),
    tasks: (): AsyncIterable<unknown> => waits(),
    agent: (): Promise<{ unreachable: string }> => new Promise<never>(() => {}),
  }
})

vi.mock('../shared/clients', () => ({
  WINDOW: 'review',
  cards: {
    watchReloads: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) wire.taken.push(call.signal)
      wire.reloadsOpened += 1
      return wire.reloads(call?.signal)
    },
  },
  windowService: {
    watchTasks: (request: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) wire.taken.push(call.signal)
      wire.tasksAsked.push(request)
      return wire.tasks()
    },
  },
  agentService: {
    getAgentState: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) wire.taken.push(call.signal)
      return wire.agent()
    },
  },
}))

/** A stream that ends in a fault the moment its call is ended. */
async function* failsOnAbort(signal?: AbortSignal): AsyncGenerator<never> {
  await new Promise<never>((_, fail) => {
    signal?.addEventListener('abort', () => fail(new Error('the call was ended')))
  })
}

/** A stream that hands over what it is given, and then stays open. */
async function* sends(...items: unknown[]): AsyncGenerator<unknown> {
  yield* items
  await new Promise<never>(() => {})
}

/** A stream that ends in a fault as it opens. */
async function* fails(): AsyncGenerator<never> {
  await Promise.resolve()
  throw new Error('the connection went')
}

/** The streams followed in a mounted window, and what they were handed. */
const openWindow = () => {
  const deps = {
    reportError: vi.fn(),
    setTasks: vi.fn(),
    onKeyDown: () => {},
    count: async () => {},
    stop: vi.fn(),
    refresh: vi.fn(async () => {}),
    unreachable: ref(''),
  }
  const host = defineComponent({
    setup() {
      useWindowStreams(deps)
      return () => null
    },
  })
  return { window: mount(host), deps }
}

/** How long a stream that went waits before it is taken up again, in milliseconds. */
const AGAIN = 1000

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

beforeEach(() => {
  wire.taken.length = 0
  wire.tasksAsked.length = 0
  wire.reloadsOpened = 0
  wire.reloads = () => wire.waits()
  wire.tasks = () => wire.waits()
  wire.agent = () => new Promise<never>(() => {})
})

afterEach(() => {
  vi.useRealTimers()
})

describe('what the window follows', () => {
  it('is ended when the window closes', () => {
    const { window } = openWindow()
    // Every call this window has in flight carries a signal, and it is one.
    expect(wire.taken.length).toBe(3)
    expect(new Set(wire.taken).size).toBe(1)
    expect(wire.taken.every((one) => !one.aborted)).toBe(true)

    window.unmount()
    expect(wire.taken.every((one) => one.aborted)).toBe(true)
  })

  it('ends the counting when the window closes', () => {
    const { window, deps } = openWindow()
    expect(deps.stop).not.toHaveBeenCalled()

    window.unmount()
    expect(deps.stop).toHaveBeenCalledOnce()
  })

  it('says nothing about a stream it ended itself, and takes nothing up again', async () => {
    wire.reloads = failsOnAbort
    const { window, deps } = openWindow()

    window.unmount()
    await settle()

    expect(deps.reportError).not.toHaveBeenCalled()
    expect(wire.reloadsOpened).toBe(1)
  })

  it('takes a stream that went up again after a wait', async () => {
    vi.useFakeTimers()
    wire.reloads = (signal) => (wire.reloadsOpened > 1 ? failsOnAbort(signal) : fails())
    const { window } = openWindow()

    await vi.advanceTimersByTimeAsync(AGAIN - 1)
    expect(wire.reloadsOpened).toBe(1)

    await vi.advanceTimersByTimeAsync(1)
    expect(wire.reloadsOpened).toBe(2)
    window.unmount()
  })

  it('follows the work being done behind its own window', () => {
    const { window } = openWindow()

    expect(wire.tasksAsked).toEqual([{ window: 'review' }])
    window.unmount()
  })

  it('passes the work being done over as cards to draw', async () => {
    wire.tasks = () =>
      sends({
        tasks: [
          {
            id: 'indexing\t01A',
            title: 'Indexing the vault',
            about: 'Garden',
            error: 'no such folder',
            isUserStarted: true,
          },
        ],
      })
    const { window, deps } = openWindow()
    await settle()

    expect(deps.setTasks).toHaveBeenCalledWith([
      {
        id: 'indexing\t01A',
        label: 'Indexing the vault',
        about: 'Garden',
        error: 'no such folder',
        isUserStarted: true,
      },
    ])
    window.unmount()
  })

  it('reads again on a message that carries a reload, and on no other', async () => {
    wire.reloads = () => sends({ reload: false }, { reload: true }, { reload: false })
    const { window, deps } = openWindow()
    await settle()

    expect(deps.refresh).toHaveBeenCalledOnce()
    window.unmount()
  })
})

describe('whether a card can be requested about', () => {
  it('is what the agent says', async () => {
    wire.agent = async () => ({ unreachable: 'no model is set up' })
    const { window, deps } = openWindow()
    await settle()

    expect(deps.unreachable.value).toBe('no model is set up')
    window.unmount()
  })

  it('is no, when the agent cannot be requested', async () => {
    wire.agent = async () => {
      throw new Error('the connection went')
    }
    const { window, deps } = openWindow()
    await settle()

    expect(deps.unreachable.value).toBe(WORDS.unreachable)
    expect(deps.unreachable.value).not.toBe('')
    window.unmount()
  })
})
