/**
 * Search and lookup domain methods for the window core.
 */
import { search } from '@/shared/clients'
import type { Selector } from '@numen/plugin-sdk'
import { modes, mapNoteType, run } from './words'
import type { SearchDeps } from '@/features/command-palette'

/** The selector of a hit, parsed from its JSON, and null where it has none. */
const parseSelector = (json: string): Selector | null => {
  if (!json) return null
  try {
    const value: unknown = JSON.parse(json)
    return typeof value === 'object' && value !== null ? (value as Selector) : null
  } catch {
    return null
  }
}

export const searchCore: SearchDeps = {
  names: async (query, limit) => {
    const answer = await search.searchNames({ query, limit })
    return answer.found.map((one) => ({
      path: one.note?.path ?? '',
      title: one.note?.title ?? '',
      heading: one.heading?.text ?? '',
      line: one.heading?.line ?? -1,
      at: one.spans.map(run),
      type: mapNoteType(one.type),
    }))
  },
  search: async (query, mode, limit) => {
    const answer = await search.searchPassages({ query, limit, mode: modes[mode] })
    return answer.found.map((one) => ({
      path: one.path,
      title: one.note?.title ?? '',
      isNote: one.note !== undefined,
      type: mapNoteType(one.type),
      selector: parseSelector(one.selector),
      text: one.text,
      start: one.span?.from ?? 0,
      length: (one.span?.to ?? 0) - (one.span?.from ?? 0),
      line: one.line,
      at: one.spans.map(run),
    }))
  },
}
