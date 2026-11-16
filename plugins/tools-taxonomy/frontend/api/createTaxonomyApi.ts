import { createApi, type Api, type ApiError } from '@numen/plugin-sdk'
import type { TagMapMessage } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.taxonomy'

/** The calls of the frontend to the backend, over the JSON service of the plugin. */
export function createTaxonomyApi(api: Api = createApi(SERVICE)): {
  watchTagMap(options?: { signal?: AbortSignal }): AsyncGenerator<TagMapMessage, ApiError | null>
} {
  return {
    watchTagMap: (options) =>
      api.stream<Record<string, never>, TagMapMessage>('WatchTagMap', {}, options),
  }
}
