import { describe, expect, it, vi } from 'vitest'
import { effectScope, watchEffect } from 'vue'
import type { ApiError } from '@numen/plugin-sdk'
import type { TagMapMessage } from '../types'
import { createTagMap, type TagMapApi } from './createTagMap'

const flush = async () => {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}

/** The stream WatchTagMap of a backend: the test sends the messages and ends it. */
const createFeed = () => {
  const queue: TagMapMessage[] = []
  let wake: (() => void) | undefined
  let isEnded = false
  const signals: (AbortSignal | undefined)[] = []
  const watchTagMap = vi.fn(async function* (options?: { signal?: AbortSignal }) {
    signals.push(options?.signal)
    while (!isEnded) {
      const next = queue.shift()
      if (next) yield next
      else await new Promise<void>((resolve) => (wake = resolve))
    }
    return null as ApiError | null
  })
  const api: TagMapApi = { watchTagMap }
  return {
    api,
    watchTagMap,
    signals,
    send: async (message: TagMapMessage) => {
      queue.push(message)
      wake?.()
      await flush()
    },
    end: () => {
      isEnded = true
      wake?.()
    },
  }
}

const WHOLE: TagMapMessage = {
  tags: { 'notes/Quay.md': ['Port', 'Sea'], 'notes/Dock.md': ['Port', 'Sea', 'Quay'] },
}

describe('the tag map', () => {
  it('gives the tags of a path as the first message holds them, in order', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })

    await feed.send(WHOLE)

    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea'])
    map.stop()
  })

  it('gives no tags for a path the map does not hold', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })

    await feed.send(WHOLE)

    expect(map.getTags('notes/Pier.md')).toStrictEqual([])
    expect(map.getTags('notes')).toStrictEqual([])
    map.stop()
  })

  it('gives no tags before the first message', () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
    map.stop()
  })

  it('reads the tags once, however many paths are asked and however often', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)

    for (let i = 0; i < 100; i++) map.getTags(`notes/Note ${i}.md`)
    map.getTags('notes/Quay.md')
    map.getTags('notes/Quay.md')

    expect(feed.watchTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('takes a change of one path without a request and keeps the other paths', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)

    await feed.send({ changed: { 'notes/Quay.md': ['Port', 'Sea', 'Dock'] } })

    expect(map.getTags('notes/Quay.md')).toStrictEqual(['Port', 'Sea', 'Dock'])
    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    expect(feed.watchTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('holds the tags of a path that had none when a change names it', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)

    await feed.send({ changed: { 'notes/Pier.md': ['Port'] } })

    expect(map.getTags('notes/Pier.md')).toStrictEqual(['Port'])
    map.stop()
  })

  it('forgets the tags of a path a change names with an empty list', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)

    await feed.send({ changed: { 'notes/Quay.md': [] } })

    expect(map.getTags('notes/Quay.md')).toStrictEqual([])
    expect(map.getTags('notes/Dock.md')).toStrictEqual(['Port', 'Sea', 'Quay'])
    map.stop()
  })

  it('tells a watcher of one path when the tags of that path change', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)
    const seen: string[] = []
    const scope = effectScope()
    scope.run(() => watchEffect(() => seen.push(map.getTags('notes/Quay.md').join(', '))))

    await feed.send({ changed: { 'notes/Quay.md': ['Sea'] } })

    expect(seen).toStrictEqual(['Port, Sea', 'Sea'])
    scope.stop()
    map.stop()
  })

  it('does not tell a watcher of one path about a change of another path', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)
    const seen: string[] = []
    const scope = effectScope()
    scope.run(() => watchEffect(() => seen.push(map.getTags('notes/Quay.md').join(', '))))

    await feed.send({ changed: { 'notes/Dock.md': ['Port'] } })

    expect(seen).toStrictEqual(['Port, Sea'])
    scope.stop()
    map.stop()
  })

  it('ends the read when it is stopped', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await feed.send(WHOLE)

    map.stop()

    expect(feed.signals[0]?.aborted).toBe(true)
  })
})
