/**
 * Search and lookup domain methods for the window core.
 */
import { search } from '@/shared/clients'
import type { Selector } from '@numen/plugin-host'
import { modes, mapNoteType, mapRange } from './words'
import type { SearchDeps } from '@/features/command-palette'

const parseSelector = (json: string): Selector | null => {
  try {
    const value: unknown = JSON.parse(json)
    return typeof value === 'object' ? (value as Selector) : null
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
      at: one.ranges.map(mapRange),
      type: mapNoteType(one.type),
      ...(one.tabKind ? { tabKind: one.tabKind } : {}),
    }))
  },
  search: async (query, mode, limit) => {
    const answer = await search.searchPassages({ query, limit, mode: modes[mode] })
    return answer.found.map((one) => ({
      path: one.path,
      title: one.note?.title ?? '',
      isNote: one.note !== undefined,
      type: mapNoteType(one.type),
      ...(one.tabKind ? { tabKind: one.tabKind } : {}),
      selector: parseSelector(one.selector),
      text: one.text,
      start: one.range?.start ?? 0,
      length: (one.range?.end ?? 0) - (one.range?.start ?? 0),
      line: one.line,
      at: one.ranges.map(mapRange),
    }))
  },
}
