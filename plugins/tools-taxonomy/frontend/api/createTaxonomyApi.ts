import { createApi, type Api, type ApiError, type Result } from '@numen/plugin-sdk'
import type { TagMapResponse } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.taxonomy'

/** The calls of the frontend to the backend, over the JSON service of the plugin. */
export function createTaxonomyApi(api: Api = createApi(SERVICE)): {
  getTagMap(options?: { signal?: AbortSignal }): Promise<Result<TagMapResponse, ApiError>>
} {
  return {
    getTagMap: (options) =>
      api.call<Record<string, never>, TagMapResponse>('GetTagMap', {}, options),
  }
}
