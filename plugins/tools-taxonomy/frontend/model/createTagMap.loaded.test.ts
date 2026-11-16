import { describe, expect, it } from 'vitest'
import type { ApiError } from '@numen/plugin-sdk'
import type { TagMapMessage } from '../types'
import { createTagMap } from './createTagMap'

const flush = async () => {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}

describe('the tag map that is loaded', () => {
  it('is loaded once the first message was read and not before', async () => {
    let release: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const map = createTagMap({
      api: {
        watchTagMap: async function* (): AsyncGenerator<TagMapMessage, ApiError | null> {
          await gate
          yield { tags: { 'notes/Quay.md': ['Port'] } }
          return new Promise<never>(() => undefined)
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

  it('is loaded when the stream ends without a message', async () => {
    const map = createTagMap({
      api: {
        watchTagMap: async function* (): AsyncGenerator<TagMapMessage, ApiError | null> {
          return { code: 'unavailable', message: 'down' }
        },
      },
    })

    await map.whenLoaded()

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
  })
})
