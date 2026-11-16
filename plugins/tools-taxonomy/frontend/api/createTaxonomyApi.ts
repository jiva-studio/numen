import { createApi, type Api } from '@numen/plugin-sdk'
import type { TagMapApi } from '../model/createTagMap'
import type { TagMapMessage } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.taxonomy'

/** The calls of the frontend to the backend, over the JSON service of the plugin. */
export function createTaxonomyApi(api: Api = createApi(SERVICE)): TagMapApi {
  return {
    watchTagMap: (options) =>
      api.stream<Record<string, never>, TagMapMessage>('WatchTagMap', {}, options),
  }
}
