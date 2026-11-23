import { describe, expect, it, vi } from 'vitest'
import { effectScope, watchEffect } from 'vue'
import type { ApiError, Disposable, Result, VaultChangeEvent } from '@numen/plugin-sdk'
import type { TagMapResponse, TagsByPath } from '../types'
import { createTagMap } from './createTagMap'

const flush = async () => {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}

const WHOLE: TagMapResponse = {
  tags: { 'notes/Quay.md': ['Port', 'Sea'], 'notes/Dock.md': ['Port', 'Sea', 'Quay'] },
}

const createFeed = (initial: TagsByPath = WHOLE.tags) => {
  let current: TagsByPath = initial
  const listeners = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const signals: (AbortSignal | undefined)[] = []
  const getTagMap = vi.fn(
    async (options?: { signal?: AbortSignal }): Promise<Result<TagMapResponse, ApiError>> => {
      signals.push(options?.signal)
      return { ok: true, value: { tags: current } }
    },
  )
  const api = { getTagMap }
  const events = {
    onVaultChange: (handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable => {
      listeners.add(handler)
      return {
        dispose: () => {
          listeners.delete(handler)
        },
      }
    },
  }
  return {
    api,
    events,
    getTagMap,
    signals,
    changeVault: async (next: TagsByPath) => {
      current = next
      for (const listener of listeners) await listener({})
      await flush()
    },
  }
}

describe('the tag map', () => {
  it('does not call getTagMap before getTags or waitForLoad', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api, events: feed.events })

    await feed.changeVault({ 'notes/Quay.md': ['Port'] })

    expect(feed.getTagMap).not.toHaveBeenCalled()
    map.stop()
  })

  it('gives the tags of a path as the backend holds them, in order', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })

    await map.waitForLoad()

    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea'])
    map.stop()
  })

  it('gives no tags for a path the map does not hold', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })

    await map.waitForLoad()

    expect(map.getTags('notes/Pier.md')).toStrictEqual([])
    expect(map.getTags('notes')).toStrictEqual([])
    map.stop()
  })

  it('gives no tags before the first answer', () => {
    const map = createTagMap({
      api: { getTagMap: () => new Promise<never>(() => undefined) },
    })

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
    map.stop()
  })

  it('reads the map once for any number of paths', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    for (let i = 0; i < 100; i++) map.getTags(`notes/Note ${i}.md`)
    map.getTags('notes/Quay.md')
    map.getTags('notes/Quay.md')

    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('takes a change of one path from setTags without a request and keeps the other paths', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    map.setTags({ 'notes/Quay.md': ['Port', 'Sea', 'Dock'] })

    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea', 'Dock'])
    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('holds the tags of a path that had none when setTags names it', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    map.setTags({ 'notes/Pier.md': ['Port'] })

    expect(map.getTags('notes/Pier.md')).toStrictEqual(['Port'])
    map.stop()
  })

  it('forgets the tags of a path setTags names with an empty list', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    map.setTags({ 'notes/Quay.md': [] })

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    map.stop()
  })

  it('notifies a watcher of a path when a vault change changes the tags of that path', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api, events: feed.events })
    await map.waitForLoad()
    const seen: string[] = []
    const scope = effectScope()
    scope.run(() => watchEffect(() => seen.push(map.getTags('notes/Quay.md').join(', '))))

    await feed.changeVault({ 'notes/Quay.md': ['Sea'], 'notes/Dock.md': ['Port', 'Sea', 'Quay'] })

    expect(seen).toStrictEqual(['Port, Sea', 'Sea'])
    scope.stop()
    map.stop()
  })

  it('does not notify a watcher of a path when a vault change changes only another path', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api, events: feed.events })
    await map.waitForLoad()
    const seen: string[] = []
    const scope = effectScope()
    scope.run(() => watchEffect(() => seen.push(map.getTags('notes/Quay.md').join(', '))))

    await feed.changeVault({ 'notes/Quay.md': ['Port', 'Sea'], 'notes/Dock.md': ['Port'] })

    expect(seen).toStrictEqual(['Port, Sea'])
    scope.stop()
    map.stop()
  })

  it('reloads on the next read after a vault change when no watcher is active', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api, events: feed.events })
    const scope = effectScope()
    scope.run(() => watchEffect(() => map.getTags('notes/Quay.md')))
    await flush()
    expect(feed.getTagMap).toHaveBeenCalledTimes(1)

    scope.stop()
    await feed.changeVault({ 'notes/Quay.md': ['Sea'] })
    expect(feed.getTagMap).toHaveBeenCalledTimes(1)

    await map.waitForLoad()
    expect(feed.getTagMap).toHaveBeenCalledTimes(2)
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Sea'])
    map.stop()
  })

  it('keeps a snapshot applied during a pending write and refreshes', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api, events: feed.events })
    const scope = effectScope()
    scope.run(() => watchEffect(() => map.getTags('notes/Quay.md')))
    await flush()

    const finishWrite = map.beginWrite()
    await feed.changeVault({ 'notes/Quay.md': ['Port', 'Sea', 'Harbor'] })
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea', 'Harbor'])

    finishWrite({ 'notes/Quay.md': ['Port'] })
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea', 'Harbor'])
    await flush()
    expect(feed.getTagMap).toHaveBeenCalledTimes(3)
    scope.stop()
    map.stop()
  })

  it('aborts an in-flight read when it is stopped', async () => {
    const signals: (AbortSignal | undefined)[] = []
    const map = createTagMap({
      api: {
        getTagMap: (options) => {
          signals.push(options?.signal)
          return new Promise<never>(() => undefined)
        },
      },
    })
    map.getTags('notes/Quay.md')
    await flush()

    map.stop()

    expect(signals[0]?.aborted).toBe(true)
  })
})
