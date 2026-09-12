/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref } from 'vue'
import type {
  PlexNeighbourhood,
  PlexRelatedSeat,
  PlexDestination,
} from '@numen/ui'
import type { NoteType } from '@/shared/file'
import { fileOf, type PathRename } from '@/shared/paths'
import { NEW_NOTE, OFFERED } from '../lib/menu'
import { asPlex, typesIn } from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import type { PlexView } from './usePlexView'
import { WORDS as words } from '../words'
import type { MenuRequest, PlexEditor, PlexTabDeps, PlexTabState } from '../types'
import { usePlexParts } from './usePlexParts'

export type { MenuRequest, PlexEditor, PlexTabDeps, PlexTabState }

export function usePlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()

  const picture = computed<PlexNeighbourhood | null>(() => {
    const around = view.neighbourhood.value
    if (!deps.ready.value || !around) return null
    const drawn = asPlex(around, map.getNodeId)
    map.retainNodeIds(drawn.nodes.map((node) => node.id))
    return drawn
  })

  const empty = computed(
    () => deps.ready.value && !view.here.value && !deps.opening.value && !view.neighbourhood.value,
  )

  const dragged = computed<readonly string[]>(() => {
    const here = view.here.value
    if (!deps.ready.value || !view.neighbourhood.value || !here) return []
    return deps.dragged.value.filter((path) => path !== here)
  })

  const menu = ref<MenuRequest | null>(null)

  const types = computed<ReadonlyMap<string, NoteType>>(() => {
    const around = view.neighbourhood.value
    return around ? typesIn(around) : new Map()
  })

  const typeOf = (node: string): NoteType => types.value.get(map.getNodePath(node) ?? '') ?? 'note'

  const drawn = computed<readonly string[]>(() => {
    const around = view.neighbourhood.value
    if (!around) return []
    const paths = [around.focus.path]
    for (const related of around.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  const { readParts, getParts } = usePlexParts(drawn, types, deps, map)

  const activate = (node: string) => {
    const path = map.getNodePath(node)
    if (path) void view.go(path)
  }

  const createNode = async (from: string, seat: PlexRelatedSeat) => {
    const path = map.getNodePath(from)
    if (path && (await deps.makes.make(path, seat))) await view.go(path)
  }

  const joinNodes = async (from: string, to: string, seat: PlexRelatedSeat) => {
    const one = map.getNodePath(from)
    const other = map.getNodePath(to)
    if (one && other && (await deps.makes.join(one, other, seat))) await view.go(one)
  }

  const getName = (path: string): string => {
    const around = view.neighbourhood.value
    if (around?.focus.path === path && around.focus.title) return around.focus.title
    const near = around?.related.find((one) => one.path === path)
    return near?.title || fileOf(path).replace(/\.md$/, '')
  }

  const dropNodes = async (draggedNodes: readonly string[], seat: PlexRelatedSeat) => {
    const here = view.here.value
    if (!here) return

    const refused: string[] = []
    let written = false

    for (const path of draggedNodes) {
      if (path === here) continue
      if (await deps.makes.join(here, path, seat)) written = true
      else refused.push(getName(path))
    }

    if (refused.length > 0) deps.says(`${words.refused} ${refused.join(', ')}`)
    if (written) await view.go(here)
  }

  const openNode = (node: string, showing: PlexDestination = 'here') => {
    const path = map.getNodePath(node)
    if (path) deps.opens(path, getName(path), showing)
  }

  const openPart = (node: string, part: string) => {
    const path = map.getNodePath(node)
    if (!path || !/^\d+$/.test(part)) return
    deps.opens(path, getName(path), 'here', Number(part))
  }

  const createNote = async () => {
    const path = await deps.writes()
    if (path) await view.go(path)
  }

  const openMenu = (asked: MenuRequest) => {
    menu.value = asked
  }

  const dismiss = () => {
    menu.value = null
  }

  const chooseMenuItem = (id: string) => {
    const on = menu.value
    menu.value = null
    if (!on) return
    if (on.node === null) {
      if (id === NEW_NOTE) void createNote()
      return
    }
    if (!OFFERED.has(id)) return
    const path = map.getNodePath(on.node)
    if (path) deps.runs(id, path, getName(path))
  }

  const followMoves = (renamed: readonly PathRename[]) => {
    view.followMoves(renamed)
    map.updateRenamedNodes(renamed)
  }

  return {
    view,
    picture,
    empty,
    dragged,
    menu,
    creatable: deps.creatable,
    typeOf,
    partsOf: getParts,
    mostParts: deps.parts,
    reads: readParts,
    readParts,
    entered: openPart,
    openPart,
    activate,
    made: createNode,
    createNode,
    joined: joinNodes,
    joinNodes,
    brought: dropNodes,
    dropNodes,
    opens: openNode,
    openNode,
    writes: createNote,
    createNote,
    asks: openMenu,
    openMenu,
    dismiss,
    chose: chooseMenuItem,
    chooseMenuItem,
    follows: followMoves,
    followMoves,
    nameOf: getName,
    getName,
  }
}
