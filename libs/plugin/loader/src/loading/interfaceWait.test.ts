import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { INTERFACE_WAIT_MS } from './limits'
import { waitWithinInterfaceLimit } from './interfaceWait'

const track = (wait: Promise<void>) => {
  const state = { isEnded: false }
  wait.then(
    () => {
      state.isEnded = true
    },
    () => undefined,
  )
  return state
}

describe('the wait for the plugins of the window slots', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('ends at INTERFACE_WAIT_MS when the work does not end', async () => {
    const wait = track(waitWithinInterfaceLimit(new Promise(() => undefined)))

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS - 1)
    expect(wait.isEnded).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(wait.isEnded).toBe(true)
  })

  it('ends when the work ends first, and leaves no timer behind', async () => {
    const wait = track(waitWithinInterfaceLimit(Promise.resolve()))

    await vi.advanceTimersByTimeAsync(0)

    expect(wait.isEnded).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('ends without failing when the work fails', async () => {
    await expect(
      waitWithinInterfaceLimit(Promise.reject(new Error('stopped'))),
    ).resolves.toBeUndefined()
  })

  it('never ends once the signal aborts, and leaves no timer behind', async () => {
    const aborting = new AbortController()
    const wait = track(waitWithinInterfaceLimit(new Promise(() => undefined), aborting.signal))

    aborting.abort()
    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS)

    expect(wait.isEnded).toBe(false)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not start on a signal that has already aborted', async () => {
    const aborting = new AbortController()
    aborting.abort()
    const wait = track(waitWithinInterfaceLimit(Promise.resolve(), aborting.signal))

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS)

    expect(wait.isEnded).toBe(false)
  })
})
