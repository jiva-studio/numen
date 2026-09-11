/**
 * Search and lookup domain methods for the window core.
 */
import { search } from './clients'
import { modes, noteType, run, sourceKind } from './words'
import type { SearchDeps } from '../../features/command-palette/search'

export type SearchOperations = SearchDeps

export const searchOperations: SearchOperations = {
  names: async (query, limit) => {
    const answer = await search.searchNames({ query, limit })
    return answer.found.map((one) => ({
      path: one.note?.path ?? '',
      title: one.note?.title ?? '',
      heading: one.heading?.text ?? '',
      line: one.heading?.line ?? -1,
      at: one.spans.map(run),
      type: noteType(one.type),
    }))
  },
  search: async (query, mode, limit) => {
    const answer = await search.searchPassages({ query, limit, mode: modes[mode] })
    return answer.found.map((one) => ({
      path: one.path,
      title: one.note?.title ?? '',
      isNote: one.note !== undefined,
      type: noteType(one.type),
      kind: sourceKind(one.kind),
      text: one.text,
      start: one.span?.from ?? 0,
      length: (one.span?.to ?? 0) - (one.span?.from ?? 0),
      line: one.line,
      at: one.spans.map(run),
    }))
  },
}

export const searchCore = searchOperations
