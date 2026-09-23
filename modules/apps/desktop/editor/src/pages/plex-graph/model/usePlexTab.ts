/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref } from 'vue'
import type { PlexNeighbourhood, PlexRelatedSeat, PlexDestination } from '@numen/ui'
import type { NoteType } from '@/entities/file'
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
    if (!deps.isReady.value || !around) return null
    const drawn = asPlex(around, map.getNodeId)
    map.retainNodeIds(drawn.nodes.map((node) => node.id))
    return drawn
  })

  const empty = computed(
    () =>
      deps.isReady.value &&
      !view.here.value &&
      !deps.openingPath.value &&
      !view.neighbourhood.value,
  )

  const dragged = computed<readonly string[]>(() => {
    const here = view.here.value
    if (!deps.isReady.value || !view.neighbourhood.value || !here) return []
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
    if (path && (await deps.editor.createInSeat(path, seat))) await view.go(path)
  }

  const joinNodes = async (from: string, to: string, seat: PlexRelatedSeat) => {
    const one = map.getNodePath(from)
    const other = map.getNodePath(to)
    if (one && other && (await deps.editor.join(one, other, seat))) await view.go(one)
  }

  const getName = (path: string): string => {
    const around = view.neighbourhood.value
    if (around?.focus.path === path && around.focus.title) return around.focus.title
    const near = around?.related.find((one) => one.path === path)
    return near?.title || fileOf(path).replace(/\.md$/, '')
  }

  const dropNodes = async (nodes: readonly string[], seat: PlexRelatedSeat) => {
    const here = view.here.value
    if (!here) return

    const notJoined: string[] = []
    let written = false

    for (const path of nodes) {
      if (path === here) continue
      if (await deps.editor.join(here, path, seat)) written = true
      else notJoined.push(getName(path))
    }

    if (notJoined.length > 0) deps.showMessage(`${words.notJoined} ${notJoined.join(', ')}`)
    if (written) await view.go(here)
  }

  const openNode = (node: string, how: PlexDestination = 'here') => {
    const path = map.getNodePath(node)
    if (path) deps.openNote(path, getName(path), how)
  }

  const openPart = (node: string, part: string) => {
    const path = map.getNodePath(node)
    if (!path || !/^\d+$/.test(part)) return
    deps.openNote(path, getName(path), 'here', Number(part))
  }

  const createNote = async () => {
    const path = await deps.createUntitledNote()
    if (path) await view.go(path)
  }

  const openMenu = (request: MenuRequest) => {
    menu.value = request
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
    if (path) deps.runCommand(id, path, getName(path))
  }

  const followMoves = (renames: readonly PathRename[]) => {
    view.followMoves(renames)
    map.updateRenamedNodes(renames)
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
    readParts,
    openPart,
    activate,
    createNode,
    joinNodes,
    dropNodes,
    openNode,
    createNote,
    openMenu,
    dismiss,
    chooseMenuItem,
    followMoves,
    getName,
  }
}
