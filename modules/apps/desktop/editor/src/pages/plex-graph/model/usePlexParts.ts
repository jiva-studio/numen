/**
 * Heading parts fetching and reactivity for notes drawn on a plex graph.
 */
import { shallowRef, watch, type Ref } from 'vue'
import type { PlexPart } from '@numen/ui'
import { answerGuard } from '@/shared/questions'
import type { NoteType } from '@/shared/file'
import { asParts } from '../lib/picture'
import type { PlexTabDeps } from '../types'
import type { NodeIdMap } from '../lib/nodeIdMap'

export function usePlexParts(
  drawn: Ref<readonly string[]>,
  types: Ref<ReadonlyMap<string, NoteType>>,
  deps: PlexTabDeps,
  nodeIdMap: NodeIdMap,
) {
  const parts = shallowRef<ReadonlyMap<string, readonly PlexPart[]>>(new Map())
  const reading = answerGuard()

  const readParts = async () => {
    const paths = drawn.value.filter((path) => (types.value.get(path) ?? 'note') === 'note')
    const mine = reading.ask()
    if (!deps.hangs.value || paths.length === 0) {
      parts.value = new Map()
      return
    }
    try {
      const found = await deps.inside(paths)
      if (!mine.current) return
      parts.value = new Map([...found].map(([path, held]) => [path, asParts(held)]))
    } catch {
      // Reading parts failed.
      if (mine.current) parts.value = new Map()
    }
  }

  watch(drawn, () => void readParts(), { immediate: true })
  watch(deps.hangs, () => void readParts())

  const getParts = (node: string): readonly PlexPart[] =>
    deps.hangs.value ? (parts.value.get(nodeIdMap.getNodePath(node) ?? '') ?? []) : []

  return { parts, readParts, getParts }
}
