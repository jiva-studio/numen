/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref, watch } from 'vue'
import type { PlexNeighbourhood, PlexRelation, PlexDestination } from '@numen/ui'
import { getFileName } from '@numen/plugin-sdk'
import { NEW_NOTE, OFFERED } from '../lib/menu'
import { convertNeighbourhood, getNoteTypes } from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { WORDS as words } from '../words'
import type {
  DirectionalLinkItem,
  LinkDirectionMode,
  LinkInspectorRequest,
  MenuRequest,
  PlexEditor,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
  PlexView,
  PathRename,
  NoteType,
} from '../types'
import { useLinkInspector } from './useLinkInspector'
import { useQuickLink } from './useQuickLink'
import { usePlexHeadings } from './usePlexHeadings'

export type {
  DirectionalLinkItem,
  LinkDirectionMode,
  LinkInspectorRequest,
  MenuRequest,
  PlexEditor,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
}

export { getInverseRelation } from './linkInspector'

function usePlexPicture(
  view: PlexView,
  deps: PlexTabDeps,
  map: ReturnType<typeof createNodeIdMap>,
) {
  const picture = computed<PlexNeighbourhood | null>(() => {
    const around = view.neighbourhood.value
    if (!deps.isReady.value || !around) return null
    return convertNeighbourhood(around, map.getNodeId)
  })

  // Stryker disable next-line ArrayDeclaration: node ids are numeric strings, so an extra entry retains nothing.
  let lastDrawnIds: readonly string[] = []
  watch(
    picture,
    (drawn) => {
      // Stryker disable next-line ArrayDeclaration: node ids are numeric strings, so an extra entry retains nothing.
      const drawnIds = drawn?.nodes.map((node) => node.id) ?? []
      map.retainNodeIds([...drawnIds, ...lastDrawnIds])
      lastDrawnIds = drawnIds
    },
    { immediate: true },
  )

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

  const types = computed<ReadonlyMap<string, NoteType>>(() => {
    const around = view.neighbourhood.value
    return around ? getNoteTypes(around) : new Map()
  })

  const getType = (node: string): NoteType => types.value.get(map.getNodePath(node)) ?? 'note'

  const drawn = computed<readonly string[]>(() => {
    const around = view.neighbourhood.value
    if (!around) return []
    const paths = [around.focus.path]
    for (const related of around.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  return { picture, empty, dragged, types, getType, drawn }
}

function usePlexNodeActions(options: {
  view: PlexView
  tabDeps: PlexTabDeps
  map: ReturnType<typeof createNodeIdMap>
  getName: (path: string) => string
  openQuickLink: (request: QuickLinkRequest) => void
}) {
  const { view, tabDeps: deps, map, getName, openQuickLink } = options
  const activate = (node: string) => {
    const path = map.getNodePath(node)
    if (path) {
      void view.go(path)
      deps.onSelectPath?.(path, getName(path))
    }
  }

  const createNode = async (
    from: string,
    relation: PlexRelation,
    at?: { x: number; y: number },
  ) => {
    if (at) {
      openQuickLink({ from, relation, at })
      return
    }
    const path = map.getNodePath(from)
    if (path && (await deps.editor.createInRelation(path, relation))) await view.go(path)
  }

  const joinNodes = async (from: string, to: string, relation: PlexRelation) => {
    const one = map.getNodePath(from)
    const other = map.getNodePath(to)
    if (one && other && (await deps.editor.join(one, other, relation))) await view.go(one)
  }

  const dropNodes = async (nodes: readonly string[], relation: PlexRelation) => {
    const here = view.here.value
    if (!here) return

    const notJoined: string[] = []
    let hasJoinedAny = false

    for (const path of nodes) {
      if (path === here) continue
      if (await deps.editor.join(here, path, relation)) hasJoinedAny = true
      else notJoined.push(getName(path))
    }

    if (notJoined.length > 0) deps.showMessage(`${words.notJoined} ${notJoined.join(', ')}`)
    if (hasJoinedAny) await view.go(here)
  }

  const openNode = (node: string, how: PlexDestination = 'here') => {
    const path = map.getNodePath(node)
    if (path) deps.openNote(path, { title: getName(path), showing: how })
  }

  const openHeading = (node: string, heading: string) => {
    const path = map.getNodePath(node)
    if (!path || !/^\d+$/.test(heading)) return
    deps.openNote(path, { title: getName(path), showing: 'here', line: Number(heading) })
  }

  const createNote = async () => {
    const path = await deps.createUntitledNote()
    if (path) await view.go(path)
  }

  return { activate, createNode, joinNodes, dropNodes, openNode, openHeading, createNote }
}

function usePlexMenu(options: {
  map: ReturnType<typeof createNodeIdMap>
  tabDeps: PlexTabDeps
  getName: (path: string) => string
  createNote: () => Promise<void>
}) {
  const { map, tabDeps: deps, getName, createNote } = options
  const menu = ref<MenuRequest | null>(null)

  const openMenu = (request: MenuRequest) => {
    menu.value = request
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

  return { menu, openMenu, chooseMenuItem }
}

function getPlexNodeName(around: PlexView['neighbourhood']['value'], path: string): string {
  if (around?.focus.path === path && around.focus.title) return around.focus.title
  const near = around?.related.find((one) => one.path === path)
  return near?.title || getFileName(path).replace(/\.md$/, '')
}

export function usePlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()
  const { picture, empty, dragged, types, getType, drawn } = usePlexPicture(view, deps, map)

  const getName = (path: string): string => getPlexNodeName(view.neighbourhood.value, path)
  const quickLinkModel = useQuickLink(view, deps, map)
  const { readHeadings, getHeadings } = usePlexHeadings(drawn, types, deps, map)
  const linkInspectorModel = useLinkInspector(view, deps.editor, map, getName)

  const nodeActions = usePlexNodeActions({
    view,
    tabDeps: deps,
    map,
    getName,
    openQuickLink: quickLinkModel.openQuickLink,
  })
  const { menu, openMenu, chooseMenuItem } = usePlexMenu({
    map,
    tabDeps: deps,
    getName,
    createNote: nodeActions.createNote,
  })

  return {
    view,
    picture,
    empty,
    dragged,
    menu,
    creatable: deps.creatable,
    getType,
    getHeadings,
    maxHeadings: deps.headings,
    readHeadings,
    openMenu,
    dismiss: () => {
      menu.value = null
      quickLinkModel.dismissQuickLink()
      linkInspectorModel.dismissLinkInspector()
    },
    chooseMenuItem,
    followMoves: (renames: readonly PathRename[]) => {
      view.followMoves(renames)
      map.updateRenamedNodes(renames)
    },
    getName,
    ...quickLinkModel,
    ...linkInspectorModel,
    ...nodeActions,
  }
}
