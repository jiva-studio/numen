/**
 * What is asked while a person types, and which answer is drawn.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { SEARCH_DEBOUNCE_MS, useDebouncedSearch, useSearchResults } from './search'

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

const deferred = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((isDone) => (resolve = isDone))
  return { promise, resolve }
}

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

describe('the answers to a search', () => {
  it('are loading from the keystroke until the answer lands', async () => {
    const answer = deferred<readonly string[]>()
    const { results, isLoading, setQuery } = useSearchResults(() => answer.promise)

    started.push(setQuery('q'))
    expect(isLoading.value).toBe(true)
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
    expect(isLoading.value).toBe(true)

    answer.resolve(['a'])
    await vi.advanceTimersByTimeAsync(0)
    expect(results.value).toStrictEqual(['a'])
    expect(isLoading.value).toBe(false)
  })

  it('are those of the later query where answers land out of order', async () => {
    const first = deferred<readonly string[]>()
    const second = deferred<readonly string[]>()
    const search = vi.fn().mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const { results, setQuery } = useSearchResults<string>(search)

    started.push(setQuery('a'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
    started.push(setQuery('ab'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)

    second.resolve(['later'])
    await vi.advanceTimersByTimeAsync(0)
    first.resolve(['earlier'])
    await vi.advanceTimersByTimeAsync(0)

    expect(results.value).toStrictEqual(['later'])
  })

  it('are none where the search fails, and loading ends', async () => {
    const { results, isLoading, setQuery } = useSearchResults<string>(() =>
      Promise.reject(new Error('down')),
    )

    started.push(setQuery('q'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)

    expect(results.value).toStrictEqual([])
    expect(isLoading.value).toBe(false)
  })

  it('are cleared by a blank text', async () => {
    const { results, setQuery } = useSearchResults(async () => ['a'])

    started.push(setQuery('q'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
    expect(results.value).toStrictEqual(['a'])

    await setQuery('')
    expect(results.value).toStrictEqual([])
  })

  it('starts not loading', () => {
    const { isLoading } = useSearchResults(async () => [])
    expect(isLoading.value).toBe(false)
  })

  it('are cleared when a later search fails', async () => {
    let hasFailed = false
    const { results, setQuery } = useSearchResults<string>(async () => {
      if (hasFailed) throw new Error('down')
      return ['prev']
    })

    started.push(setQuery('q'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
    expect(results.value).toStrictEqual(['prev'])

    hasFailed = true
    started.push(setQuery('q2'))
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
    expect(results.value).toStrictEqual([])
  })
})
