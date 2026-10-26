/**
 * Fetching and reactivity of the headings of notes drawn on a plex graph.
 */
import { shallowRef, watch, type Ref } from 'vue'
import type { PlexHeading } from '../plex'
import { createRequestGuard } from '../lib/requestGuard'
import { convertHeadings } from '../lib/picture'
import type { PlexTabDeps, NoteType } from '../types'
import type { NodeIdMap } from '../lib/nodeIdMap'

export function usePlexHeadings(
  paths: Ref<readonly string[]>,
  types: Ref<ReadonlyMap<string, NoteType>>,
  ...[deps, nodeIdMap]: [deps: PlexTabDeps, nodeIdMap: NodeIdMap]
) {
  const headings = shallowRef<ReadonlyMap<string, readonly PlexHeading[]>>(new Map())
  const guard = createRequestGuard()

  const readHeadings = async () => {
    const notes = paths.value.filter((path) => (types.value.get(path) ?? 'note') === 'note')
    const mine = guard.acquireRequest()
    if (!deps.isNesting.value || notes.length === 0) {
      headings.value = new Map()
      return
    }
    try {
      const found = await deps.readHeadings(notes)
      if (!mine.isCurrent) return
      headings.value = new Map(
        [...found].map(([path, rawHeadings]) => [path, convertHeadings(rawHeadings)]),
      )
    } catch {
      // Reading headings failed.
      if (mine.isCurrent) headings.value = new Map()
    }
  }

  watch(paths, () => void readHeadings(), { immediate: true })
  watch(deps.isNesting, () => void readHeadings())

  const getHeadings = (node: string): readonly PlexHeading[] =>
    deps.isNesting.value ? (headings.value.get(nodeIdMap.getNodePath(node)) ?? []) : []

  return { headings, readHeadings, getHeadings }
}
