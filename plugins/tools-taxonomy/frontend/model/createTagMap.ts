import { shallowReactive } from 'vue'
import type { ApiError } from '@numen/plugin-sdk'
import type { TagMapMessage, TagsByPath } from '../types'

/** What the tag map asks of the backend. */
export interface TagMapApi {
  watchTagMap(options?: { signal?: AbortSignal }): AsyncGenerator<TagMapMessage, ApiError | null>
}

/** The tags of the notes of the vault by path, followed while the plugin runs. */
export interface TagMap {
  /** The tags of the note at the path in the order the note holds them; none for a path without tags. */
  getTags(path: string): readonly string[]
  /** Ends the read of the stream. */
  stop(): void
}

const NO_TAGS: readonly string[] = []

/**
 * The map of the tags of the vault: one read of the stream WatchTagMap, whose first message holds every
 * note that has tags and whose later messages hold the notes whose tags changed. A caller that reads the
 * tags of one path is notified when the tags of that path change and of no other.
 */
export function createTagMap({ api }: { api: TagMapApi }): TagMap {
  const tags = shallowReactive(new Map<string, readonly string[]>())
  const controller = new AbortController()

  const replace = (whole: TagsByPath): void => {
    tags.clear()
    for (const [path, found] of Object.entries(whole)) tags.set(path, found)
  }

  const change = (changed: TagsByPath): void => {
    for (const [path, found] of Object.entries(changed)) {
      if (found.length === 0) tags.delete(path)
      else tags.set(path, found)
    }
  }

  const read = async (): Promise<void> => {
    const messages = api.watchTagMap({ signal: controller.signal })
    for (;;) {
      const next = await messages.next()
      if (next.done) return
      if (next.value.tags) replace(next.value.tags)
      if (next.value.changed) change(next.value.changed)
    }
  }
  read().catch(() => undefined)

  return {
    getTags: (path) => tags.get(path) ?? NO_TAGS,
    stop: () => controller.abort(),
  }
}
