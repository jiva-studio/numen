import { describe, expect, it } from 'vitest'
import { effectScope, watchEffect } from 'vue'
import type { ApiError, Disposable, Result, VaultChangeEvent } from '@numen/plugin-sdk'
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
    const loading = map.waitForLoad().then(() => {
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

    await map.waitForLoad()

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
  })

  it('resolves waitForLoad after the first read even when a vault change queues a refresh', async () => {
    let releaseFirst: () => void = () => undefined
    let releaseSecond: () => void = () => undefined
    const firstGate = new Promise<void>((resolve) => {
      releaseFirst = resolve
    })
    const secondGate = new Promise<void>((resolve) => {
      releaseSecond = resolve
    })
    let calls = 0
    let onVaultChange: ((event: VaultChangeEvent) => void | Promise<void>) | undefined
    const map = createTagMap({
      api: {
        getTagMap: async (): Promise<Result<TagMapResponse, ApiError>> => {
          calls += 1
          if (calls === 1) {
            await firstGate
            return { ok: true, value: { tags: { 'notes/Quay.md': ['Port'] } } }
          }
          await secondGate
          return { ok: true, value: { tags: { 'notes/Quay.md': ['Port', 'Sea'] } } }
        },
      },
      events: {
        onVaultChange: (handler): Disposable => {
          onVaultChange = handler
          return { dispose: () => undefined }
        },
      },
    })
    const scope = effectScope()
    scope.run(() => watchEffect(() => map.getTags('notes/Quay.md')))
    let isLoaded = false
    const loading = map.waitForLoad().then(() => {
      isLoaded = true
    })
    await onVaultChange?.({})
    releaseFirst()
    await loading

    expect([isLoaded, calls, map.getTags('notes/Quay.md')]).toStrictEqual([true, 2, ['Port']])

    releaseSecond()
    await flush()
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea'])
    scope.stop()
    map.stop()
  })
})
