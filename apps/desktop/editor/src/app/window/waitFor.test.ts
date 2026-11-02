import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { waitFor } from './waitFor'

describe('waitFor', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('ends at the end of the wait when the work does not end', async () => {
    let isEnded = false
    const ended = waitFor(new Promise(() => undefined), 2_000).then(() => {
      isEnded = true
    })
    await vi.advanceTimersByTimeAsync(1_999)
    expect(isEnded).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(isEnded).toBe(true)
    await ended
  })

  it('ends when the work ends before the wait', async () => {
    let isEnded = false
    const ended = waitFor(Promise.resolve(), 2_000).then(() => {
      isEnded = true
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(isEnded).toBe(true)
    await ended
  })

  it('ends without failing when the work fails', async () => {
    await expect(waitFor(Promise.reject(new Error('stopped')), 2_000)).resolves.toBeUndefined()
  })

  it('leaves no timer behind once the work has ended', async () => {
    await waitFor(Promise.resolve(), 2_000)
    expect(vi.getTimerCount()).toBe(0)
  })
})
