/** The poll starts a read one interval after the previous read finished and stops once it reports a stall. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Result } from '@numen/result'
import type { LinkInfo } from '@/entities/identity'
import { createLinkInfo } from '@/testing/identityClient'
import { createLinkPoll } from './createLinkPoll'

describe('createLinkPoll', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('starts no read while the previous one is still waiting for the core', async () => {
    const pending: Array<(answer: Result<LinkInfo, string>) => void> = []
    const read = vi.fn(() => new Promise<Result<LinkInfo, string>>((resolve) => pending.push(resolve)))
    const poll = createLinkPoll({
      read,
      isEnded: () => false,
      onInfo: () => undefined,
      isWaiting: () => false,
      stallAfter: 3,
      onStalled: () => undefined,
      intervalMs: 1000,
      run: (task) => {
        task.catch(() => undefined)
      },
    })

    poll.start()
    await vi.advanceTimersByTimeAsync(5000)

    expect(read).toHaveBeenCalledTimes(1)
    poll.dispose()
  })

  it('reports a stall once and reads no more', async () => {
    const read = vi.fn(() => Promise.resolve({ ok: true as const, value: createLinkInfo() }))
    const onStalled = vi.fn()
    const poll = createLinkPoll({
      read,
      isEnded: () => false,
      onInfo: () => undefined,
      isWaiting: () => true,
      stallAfter: 2,
      onStalled,
      intervalMs: 1000,
      run: (task) => {
        task.catch(() => undefined)
      },
    })

    poll.start()
    await vi.advanceTimersByTimeAsync(10_000)

    expect(onStalled).toHaveBeenCalledTimes(1)
    expect(read).toHaveBeenCalledTimes(2)
    poll.dispose()
  })
})
