/**
 * Following a vault that is still being read: the picture is drawn again on
 * every change, and the following stops with the scope it was made in.
 */
import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { watchVault } from './watchVault'
import type { Core } from '../core'

/** What the core was asked to follow with, and how the stream answered. */
const core = (changes: () => AsyncGenerator<unknown>) => {
  const request: { signal?: AbortSignal } = {}
  const reached = {
    vault: {
      watchVaultChanges: (_request: unknown, options: { signal: AbortSignal }) => {
        request.signal = options.signal
        return changes()
      },
    },
  } as unknown as Core
  return { reached, asked: request }
}

/** Everything already queued, run. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

describe('following a vault', () => {
  it('draws again on every change', async () => {
    const redraw = vi.fn()
    const { reached } = core(async function* () {
      yield {}
      yield {}
      yield {}
    })

    const scope = effectScope()
    scope.run(() => watchVault(reached, redraw))
    await settle()

    expect(redraw).toHaveBeenCalledTimes(3)
    scope.stop()
  })

  it('stops following where the scope it was made in goes', async () => {
    const { reached, asked } = core(async function* () {
      yield {}
      await new Promise(() => {})
    })

    const scope = effectScope()
    scope.run(() => watchVault(reached, () => {}))
    await settle()
    expect(asked.signal?.aborted).toBe(false)

    scope.stop()
    expect(asked.signal?.aborted).toBe(true)
  })

  // A vault closed under the stream is nothing to report: there is only
  // nothing left to follow.
  it('says nothing where the stream breaks', async () => {
    const redraw = vi.fn()
    const { reached } = core(async function* () {
      yield {}
      throw new Error('the vault is closed')
    })

    const scope = effectScope()
    expect(() => scope.run(() => watchVault(reached, redraw))).not.toThrow()
    await settle()

    expect(redraw).toHaveBeenCalledTimes(1)
    scope.stop()
  })
})
