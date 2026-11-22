import { describe, expect, it, vi } from 'vitest'
import { createFakeApi } from '../testing/fakeApi'
import { createFieldCache } from './createFieldCache'

describe('the field cache', () => {
  it('is empty before the first read and holds the last answer after a refresh', async () => {
    const fake = createFakeApi({
      fields: { projects: [{ name: 'home', count: 2 }], contexts: [], tags: [] },
    })
    const cache = createFieldCache(fake.api)
    expect(cache.get()).toEqual({ projects: [], contexts: [], tags: [] })

    await cache.refresh()
    expect(cache.get().projects).toEqual([{ name: 'home', count: 2 }])

    fake.setFields({ projects: [], contexts: [{ name: 'phone', count: 1 }], tags: [] })
    await cache.refresh()
    expect(cache.get().contexts).toEqual([{ name: 'phone', count: 1 }])
  })

  it('keeps what it has when the backend fails', async () => {
    const fake = createFakeApi({
      fields: { projects: [{ name: 'home', count: 2 }], contexts: [], tags: [] },
    })
    const cache = createFieldCache(fake.api)
    await cache.refresh()

    const failing = createFieldCache({
      listFields: async () => ({ ok: false, error: { code: 'internal', message: 'x' } }),
    } as never)
    await failing.refresh()

    expect(cache.get().projects).toHaveLength(1)
    expect(failing.get()).toEqual({ projects: [], contexts: [], tags: [] })
  })

  it('asks the backend again while it is empty, no more than once in a while', async () => {
    let at = 0
    const listFields = vi.fn(async () => ({
      ok: true as const,
      value: { projects: [], contexts: [], tags: [] },
    }))
    const cache = createFieldCache({ listFields } as never, { now: () => at })

    await cache.ensure()
    await cache.ensure()
    expect(listFields).toHaveBeenCalledTimes(1)

    at = 5000
    await cache.ensure()
    expect(listFields).toHaveBeenCalledTimes(2)
  })

  it('does not ask again once it holds something', async () => {
    const fake = createFakeApi({
      fields: { projects: [{ name: 'home', count: 2 }], contexts: [], tags: [] },
    })
    const cache = createFieldCache(fake.api, { now: () => 0 })
    await cache.refresh()

    await cache.ensure()

    expect(cache.get().projects).toHaveLength(1)
  })

  it('waits for a read that is already running before it decides to read', async () => {
    const answers: ((value: unknown) => void)[] = []
    const listFields = vi.fn(() => new Promise((resolve) => answers.push(resolve)))
    const cache = createFieldCache({ listFields } as never, { now: () => 0 })

    const running = cache.refresh()
    const ensured = cache.ensure()
    answers[0]?.({
      ok: true,
      value: { projects: [{ name: 'work', count: 1 }], contexts: [], tags: [] },
    })
    await ensured
    await running

    expect(listFields).toHaveBeenCalledTimes(1)
    expect(cache.get().projects).toEqual([{ name: 'work', count: 1 }])
  })

  it('does not end the wait before the newest of several reads answered', async () => {
    const answers: ((value: unknown) => void)[] = []
    const listFields = vi.fn(() => new Promise((resolve) => answers.push(resolve)))
    const cache = createFieldCache({ listFields } as never, { now: () => 0 })

    const first = cache.refresh()
    const second = cache.refresh()
    const ensured = cache.ensure()
    answers[0]?.({ ok: true, value: { projects: [], contexts: [], tags: [] } })
    await first
    answers[1]?.({
      ok: true,
      value: { projects: [{ name: 'work', count: 1 }], contexts: [], tags: [] },
    })
    await ensured
    await second

    expect(cache.get().projects).toEqual([{ name: 'work', count: 1 }])
  })
})
