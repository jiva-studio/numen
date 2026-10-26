/**
 * The counting of the vaults is started by whichever part of the review asks
 * first, and ended when the plugin is disposed.
 *
 * The streams are HTTP calls that stay open until something ends them. A
 * plugin disposed without ending them leaves them running against a page
 * nobody is looking at.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createVaultCounts } from './counts'

const wire = vi.hoisted(() => {
  /** A stream that stays open, so nothing ends on its own. */
  async function* waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  }
  return {
    waits,
    taken: [] as AbortSignal[],
    countsOpened: 0,
    reloadsOpened: 0,
    reloads: (_signal?: AbortSignal): AsyncIterable<unknown> => waits(),
  }
})

vi.mock('../shared/clients', () => ({
  cards: {
    watchCardsDue: async function* (_: unknown, call?: { signal?: AbortSignal }) {
      if (call?.signal) wire.taken.push(call.signal)
      wire.countsOpened += 1
      yield { day: '2026-04-02', vaults: [] }
    },
    watchReloads: (_: unknown, call?: { signal?: AbortSignal }) => {
      if (call?.signal) wire.taken.push(call.signal)
      wire.reloadsOpened += 1
      return wire.reloads(call?.signal)
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

/** How long a stream that went waits before it is taken up again, in milliseconds. */
const AGAIN = 1000

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

beforeEach(() => {
  wire.taken.length = 0
  wire.countsOpened = 0
  wire.reloadsOpened = 0
  wire.reloads = () => wire.waits()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the counting of the vaults', () => {
  it('waits for the first part that asks for it', async () => {
    createVaultCounts({ reportError: vi.fn() })
    await settle()

    expect(wire.countsOpened).toBe(0)
    expect(wire.reloadsOpened).toBe(0)
  })

  it('starts once, however many parts ask', async () => {
    const counts = createVaultCounts({ reportError: vi.fn() })

    counts.start()
    counts.start()
    await settle()

    expect(wire.countsOpened).toBe(1)
    expect(wire.reloadsOpened).toBe(1)
    expect(counts.day.value).toBe('2026-04-02')
    counts.dispose()
  })

  it('starts nothing once the plugin is disposed', async () => {
    const counts = createVaultCounts({ reportError: vi.fn() })

    counts.dispose()
    counts.start()
    await settle()

    expect(wire.countsOpened).toBe(0)
  })
})

describe('what the counting follows', () => {
  it('is ended when the plugin is disposed', () => {
    wire.reloads = failsOnAbort
    const counts = createVaultCounts({ reportError: vi.fn() })
    counts.start()
    expect(wire.taken.some((one) => one.aborted)).toBe(false)

    counts.dispose()

    expect(wire.taken.every((one) => one.aborted)).toBe(true)
  })

  it('says nothing about a stream it ended itself, and takes nothing up again', async () => {
    wire.reloads = failsOnAbort
    const reportError = vi.fn()
    const counts = createVaultCounts({ reportError })
    counts.start()

    counts.dispose()
    await settle()

    expect(reportError).not.toHaveBeenCalled()
    expect(wire.reloadsOpened).toBe(1)
  })

  it('takes a stream that went up again after a wait', async () => {
    vi.useFakeTimers()
    wire.reloads = (signal) => (wire.reloadsOpened > 1 ? failsOnAbort(signal) : fails())
    const counts = createVaultCounts({ reportError: vi.fn() })
    counts.start()

    await vi.advanceTimersByTimeAsync(AGAIN - 1)
    expect(wire.reloadsOpened).toBe(1)

    await vi.advanceTimersByTimeAsync(1)
    expect(wire.reloadsOpened).toBe(2)
    counts.dispose()
  })
})

describe('a deck written underneath the window', () => {
  it('counts the vaults again on a message that carries a reload, and on no other', async () => {
    wire.reloads = () => sends({ reload: false }, { reload: true }, { reload: false })
    const counts = createVaultCounts({ reportError: vi.fn() })

    counts.start()
    await settle()
    await settle()

    expect(wire.countsOpened).toBe(2)
    counts.dispose()
  })

  it('is answered by the refresh the open review sets, and by counting once it is unset', async () => {
    let send: (item: unknown) => void = () => {}
    wire.reloads = async function* () {
      for (;;) yield await new Promise((then) => (send = then))
    }
    const counts = createVaultCounts({ reportError: vi.fn() })
    const refresh = vi.fn(async () => {})
    counts.start()
    await settle()

    counts.setRefresh(refresh)
    send({ reload: true })
    await settle()
    expect(refresh).toHaveBeenCalledOnce()
    expect(wire.countsOpened).toBe(1)

    counts.setRefresh(null)
    send({ reload: true })
    await settle()
    await settle()
    expect(refresh).toHaveBeenCalledOnce()
    expect(wire.countsOpened).toBe(2)
    counts.dispose()
  })
})
