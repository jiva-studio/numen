/** A failure of a background task is not swallowed: it surfaces as an uncaught error. */
import { describe, expect, it, vi } from 'vitest'
import { reportFailure } from './report'

describe('reporting a failure', () => {
  it('rethrows the rejection from a microtask, so it reaches the uncaught handler', async () => {
    const queued: (() => void)[] = []
    const queue = vi.spyOn(globalThis, 'queueMicrotask').mockImplementation((run) => {
      queued.push(run)
    })
    const failure = new Error('the shed is gone')

    reportFailure(failure)

    expect(queued).toHaveLength(1)
    expect(() => queued[0]?.()).toThrow(failure)
    queue.mockRestore()
  })

  it('throws nothing in the place it is called from', () => {
    const queue = vi.spyOn(globalThis, 'queueMicrotask').mockImplementation(() => undefined)
    expect(() => reportFailure(new Error('late'))).not.toThrow()
    queue.mockRestore()
  })
})
