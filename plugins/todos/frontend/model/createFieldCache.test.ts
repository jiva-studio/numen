import { describe, expect, it } from 'vitest'
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
    const fake = createFakeApi({ fields: { projects: [{ name: 'home', count: 2 }], contexts: [], tags: [] } })
    const cache = createFieldCache(fake.api)
    await cache.refresh()

    const failing = createFieldCache({
      listFields: async () => ({ ok: false, error: { code: 'internal', message: 'x' } }),
    } as never)
    await failing.refresh()

    expect(cache.get().projects).toHaveLength(1)
    expect(failing.get()).toEqual({ projects: [], contexts: [], tags: [] })
  })
})
