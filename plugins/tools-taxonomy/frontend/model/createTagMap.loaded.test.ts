import { describe, expect, it } from 'vitest'
import type { ApiError, Result } from '@numen/plugin-sdk'
import type { TagMapResponse } from '../types'
import { createTagMap } from './createTagMap'

const flush = async () => {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}

describe('the tag map that is loaded', () => {
  it('is loaded once the response was read and not before', async () => {
    let release: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const map = createTagMap({
      api: {
        getTagMap: async (): Promise<Result<TagMapResponse, ApiError>> => {
          await gate
          return { ok: true, value: { tags: { 'notes/Quay.md': ['Port'] } } }
        },
      },
    })
    let isLoaded = false
    const loading = map.whenLoaded().then(() => {
      isLoaded = true
    })
    await flush()
    const before = isLoaded
    release()
    await loading

    expect([before, isLoaded, map.getTags('notes/Quay.md')]).toStrictEqual([false, true, ['Port']])
  })

  it('is loaded when the request fails', async () => {
    const map = createTagMap({
      api: {
        getTagMap: async (): Promise<Result<TagMapResponse, ApiError>> => ({
          ok: false,
          error: { code: 'unavailable', message: 'down' },
        }),
      },
    })

    await map.whenLoaded()

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
  })
})
