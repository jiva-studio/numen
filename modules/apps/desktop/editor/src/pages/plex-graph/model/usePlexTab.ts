/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref, type Ref } from 'vue'
import type { PlexNeighbourhood, PlexRelatedSeat, PlexDestination } from '@numen/ui'
import type { NoteType } from '@/entities/file'
import { fileOf, type PathRename } from '@/shared/paths'
import { NEW_NOTE, OFFERED } from '../lib/menu'
import { asPlex, typesIn } from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import type { PlexView } from './usePlexView'
import { WORDS as words } from '../words'
import type {
  DirectionalLinkItem,
  LinkDirectionMode,
  LinkInspectorRequest,
  LinkInspectorSavePayload,
  MenuRequest,
  PlexEditor,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
} from '../types'
import { usePlexParts } from './usePlexParts'

export type {
  DirectionalLinkItem,
  LinkDirectionMode,
  LinkInspectorRequest,
  LinkInspectorSavePayload,
  MenuRequest,
  PlexEditor,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
}

import {
  createMutualInspectorLinks,
  createSingleInspectorLink,
  getInverseSeat,
} from './linkInspector'

export { getInverseSeat }

export function usePlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()
  const linkDirections = ref<Map<string, string>>(new Map())
  const linkDescriptions = ref<Map<string, string>>(new Map())

  const picture = computed<PlexNeighbourhood | null>(() => {
    const around = view.neighbourhood.value
    if (!deps.isReady.value || !around) return null
    const drawn = asPlex(around, map.getNodeId, linkDirections.value)
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
  const quickLink = ref<QuickLinkRequest | null>(null)
  const isNavigatingOnCreate = (deps.isNavigatingOnCreate ?? ref(true)) as Ref<boolean>

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
    if (path) {
      void view.go(path)
      deps.onSelectPath?.(path, getName(path))
    }
  }

  const openQuickLink = (request: QuickLinkRequest) => {
    quickLink.value = request
  }

  const dismissQuickLink = () => {
    quickLink.value = null
  }

  const searchNotes = async (
    query: string,
  ): Promise<readonly { path: string; title: string }[]> => {
    if (!deps.searchNames) return []
    const results = await deps.searchNames(query)
    return results.map((item) => ({ path: item.path, title: item.title }))
  }

  const confirmQuickLink = async (titleOrPath: string, isExisting: boolean) => {
    const active = quickLink.value
    quickLink.value = null
    if (!active) return

    const fromPath = map.getNodePath(active.from)
    if (!fromPath) return

    if (isExisting) {
      const success = await deps.editor.join(fromPath, titleOrPath, active.seat)
      if (success) {
        if (isNavigatingOnCreate.value) {
          await view.go(titleOrPath)
        } else {
          await view.go(fromPath)
        }
      }
    } else {
      const title = titleOrPath.trim()
      if (!title) return
      const created = await deps.editor.createWithTitle(title, fromPath, active.seat)
      if (created) {
        if (isNavigatingOnCreate.value) {
          await view.go(created.path)
        } else {
          await view.go(fromPath)
        }
      }
    }
  }

  const createNode = async (from: string, seat: PlexRelatedSeat, at?: { x: number; y: number }) => {
    if (at) {
      openQuickLink({ from, seat, at })
      return
    }
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
    let hasJoinedAny = false

    for (const path of nodes) {
      if (path === here) continue
      if (await deps.editor.join(here, path, seat)) hasJoinedAny = true
      else notJoined.push(getName(path))
    }

    if (notJoined.length > 0) deps.showMessage(`${words.notJoined} ${notJoined.join(', ')}`)
    if (hasJoinedAny) await view.go(here)
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

  const linkInspector = ref<LinkInspectorRequest | null>(null)

  const openLinkInspector = (pair: string, at: { x: number; y: number }) => {
    const parts = pair.split(' ')
    if (parts.length < 2) return
    const idA = parts[0]!
    const idB = parts[1]!
    const pathA = map.getNodePath(idA)
    const pathB = map.getNodePath(idB)
    if (!pathA || !pathB) return

    const titleA = getName(pathA)
    const titleB = getName(pathB)
    const around = view.neighbourhood.value
    const related = around?.related.find((r) => r.path === pathB || r.path === pathA)
    const isFocusA = pathA === around?.focus.path

    const links = related?.isMutual
      ? createMutualInspectorLinks(pathA, pathB, related, isFocusA, linkDescriptions.value)
      : createSingleInspectorLink(
          pathA,
          pathB,
          related,
          isFocusA,
          linkDirections.value,
          linkDescriptions.value,
        )

    linkInspector.value = {
      pairKey: pair,
      nodeA: { id: idA, title: titleA, path: pathA },
      nodeB: { id: idB, title: titleB, path: pathB },
      links,
      at,
    }
  }

  const dismissLinkInspector = () => {
    linkInspector.value = null
  }

  const saveLinkInspector = async (payload: LinkInspectorSavePayload) => {
    for (const removed of payload.removedLinks) {
      await deps.editor.removeLink(removed.from, removed.to, removed.role)
      linkDescriptions.value.delete(`${removed.from}->${removed.to}`)
    }

    for (const row of payload.rows) {
      await deps.editor.join(row.from, row.to, row.role, row.description)
      linkDescriptions.value.set(`${row.from}->${row.to}`, row.description)
    }

    if (payload.rows.length === 1) {
      const row = payload.rows[0]!
      const pairKey = [row.from, row.to].sort().join(' ')
      if (row.direction === 'undirected') {
        linkDirections.value.set(pairKey, 'undirected')
      } else {
        linkDirections.value.set(pairKey, `${row.from}->${row.to}`)
      }
    } else if (payload.rows.length > 1) {
      const row = payload.rows[0]!
      const pairKey = [row.from, row.to].sort().join(' ')
      linkDirections.value.delete(pairKey)
    }

    const here = view.here.value
    if (here) await view.go(here)
    dismissLinkInspector()
  }

  const removeEntireLink = async (pairKey: string) => {
    const active = linkInspector.value
    if (active && active.pairKey === pairKey) {
      for (const link of active.links) {
        await deps.editor.removeLink(link.from, link.to, link.role)
        linkDescriptions.value.delete(`${link.from}->${link.to}`)
      }
    }
    const here = view.here.value
    if (here) await view.go(here)
    dismissLinkInspector()
  }

  const openMenu = (request: MenuRequest) => {
    menu.value = request
  }

  const dismiss = () => {
    menu.value = null
    quickLink.value = null
    linkInspector.value = null
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
    quickLink,
    linkInspector,
    isNavigatingOnCreate,
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
    openQuickLink,
    dismissQuickLink,
    confirmQuickLink,
    searchNotes,
    openLinkInspector,
    dismissLinkInspector,
    saveLinkInspector,
    removeEntireLink,
  }
}
