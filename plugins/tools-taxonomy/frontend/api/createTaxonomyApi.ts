import {
  asFailure,
  asValue,
  createApi,
  type Api,
  type ApiError,
  type JsonValue,
  type Result,
} from '@numen/plugin-sdk'
import type { TagMapResponse } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.taxonomy'

const isStringArray = (value: JsonValue | undefined): value is readonly string[] =>
  Array.isArray(value) && value.every((entry): entry is string => typeof entry === 'string')

const isJsonObject = (
  value: JsonValue | undefined,
): value is { readonly [key: string]: JsonValue } =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const parseTagMap = (reply: JsonValue): TagMapResponse | null => {
  if (!isJsonObject(reply)) return null
  const rawTags = reply['tags']
  if (!isJsonObject(rawTags)) return null
  const tags: Record<string, readonly string[]> = {}
  for (const [path, found] of Object.entries(rawTags)) {
    if (!isStringArray(found)) return null
    tags[path] = found
  }
  return { tags }
}

/** The calls of the frontend to the backend, over the JSON service of the plugin. */
export function createTaxonomyApi(api: Api = createApi(SERVICE)): {
  getTagMap(options?: { signal?: AbortSignal }): Promise<Result<TagMapResponse, ApiError>>
} {
  return {
    getTagMap: async (options) => {
      const result = await api.call<Record<string, never>, JsonValue>('GetTagMap', {}, options)
      if (!result.ok) return result
      const parsed = parseTagMap(result.value)
      if (!parsed) return asFailure({ code: 'internal', message: 'Invalid GetTagMap response' })
      return asValue(parsed)
    },
  }
}
