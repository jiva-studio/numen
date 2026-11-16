import type { ApiError } from '@numen/plugin-sdk'
import type { TagMapMessage } from '../types'

export interface TagMapApi {
  watchTagMap(options?: { signal?: AbortSignal }): AsyncGenerator<TagMapMessage, ApiError | null>
}

export interface TagMap {
  getTags(path: string): readonly string[]
  stop(): void
}

export function createTagMap(_parts: { api: TagMapApi }): TagMap {
  return { getTags: () => [], stop: () => undefined }
}
