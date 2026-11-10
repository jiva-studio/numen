/**
 * The search requests made while the user types, and which response is drawn.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useDebouncedSearch } from './debounced'

const SEARCH_DEBOUNCE_MS = 120

beforeEach(() => {
  vi.useFakeTimers()
})

/** Searches started by a test and awaited after it. */
const started: Promise<unknown>[] = []

afterEach(async () => {
  await vi.runAllTimersAsync()
  await Promise.all(started.splice(0))
  vi.useRealTimers()
})

describe('a text typed', () => {
  it('is requested once, after the pause that follows the last keystroke', async () => {
    const run = vi.fn((_query: string) => Promise.resolve())
    const { search } = useDebouncedSearch()

    started.push(search('q', { blank: vi.fn(), run }))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS - 1)
    started.push(search(' qu ', { blank: vi.fn(), run }))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS - 1)
    expect(run).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    expect(run).toHaveBeenCalledTimes(1)
    expect(run.mock.calls[0]?.[0]).toBe('qu')
  })

  it('is blank where it holds only space, and nothing waits', async () => {
    const blank = vi.fn()
    const pending = vi.fn()
    const run = vi.fn((_query: string) => Promise.resolve())
    await useDebouncedSearch().search('  ', { blank, pending, run })

    expect(blank).toHaveBeenCalledTimes(1)
    expect(pending).not.toHaveBeenCalled()
    expect(run).not.toHaveBeenCalled()
  })

  it('is cancelled by a cancel call', async () => {
    const run = vi.fn((_query: string) => Promise.resolve())
    const { search, cancel } = useDebouncedSearch()

    started.push(search('q', { blank: vi.fn(), run }))
    cancel()
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)

    expect(run).not.toHaveBeenCalled()
  })

  it('waits on the clock it is handed', async () => {
    const waited: number[] = []
    const run = vi.fn((_query: string) => Promise.resolve())
    await useDebouncedSearch({
      wait: async (ms) => {
        waited.push(ms)
      },
    }).search('q', { blank: vi.fn(), run })

    expect(waited).toStrictEqual([SEARCH_DEBOUNCE_MS])
    expect(run).toHaveBeenCalledTimes(1)
  })

  it('closes the request guard when the scope disposes', () => {
    const scope = effectScope()
    let isClosed = false
    scope.run(() => {
      useDebouncedSearch({
        guard: {
          acquireRequest: () => ({ isCurrent: true, claim: () => true }),
          cancel: vi.fn(),
          close: () => {
            isClosed = true
          },
          open: () => true,
        },
      })
    })
    scope.stop()
    expect(isClosed).toBe(true)
  })
})
