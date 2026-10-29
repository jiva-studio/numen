/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref, watch, type Component } from 'vue'
import { findIcon } from '../lib/icons'
import type { PlexNeighbourhood, PlexRelation, PlexDestination } from '../plex'
import { getFileName, NOTE_TYPE } from '@numen/plugin-sdk'
import { createNodeItems, NEW_NOTE, NONE, OFFERED } from '../lib/menu'
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

function usePlexImage(view: PlexView, deps: PlexTabDeps, map: ReturnType<typeof createNodeIdMap>) {
  const picture = computed<PlexNeighbourhood | null>(() => {
    const neighbourhood = view.neighbourhood.value
    if (!deps.isReady.value || !neighbourhood) return null
    return convertNeighbourhood(neighbourhood, map.getNodeId)
  })

  // Stryker disable next-line ArrayDeclaration: node ids are numeric strings, so an extra entry retains nothing.
  let lastDrawnIds: readonly string[] = []
  watch(
    picture,
    (rendered) => {
      // Stryker disable next-line ArrayDeclaration: node ids are numeric strings, so an extra entry retains nothing.
      const drawnIds = rendered?.nodes.map((node) => node.id) ?? []
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
    const neighbourhood = view.neighbourhood.value
    return neighbourhood ? getNoteTypes(neighbourhood) : new Map()
  })

  const getType = (node: string): NoteType => types.value.get(map.getNodePath(node)) ?? NOTE_TYPE

  const activePaths = computed<readonly string[]>(() => {
    const neighbourhood = view.neighbourhood.value
    if (!neighbourhood) return []
    const paths = [neighbourhood.focus.path]
    for (const related of neighbourhood.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  return { picture, empty, dragged, types, getType, drawn: activePaths }
}

function createPlexNodeActions(options: {
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
    pos?: { x: number; y: number },
  ) => {
    if (pos) {
      openQuickLink({ from, relation, at: pos })
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

  const openNode = (node: string, how: PlexDestination = 'current') => {
    const path = map.getNodePath(node)
    if (path) deps.openNote(path, { title: getName(path), showing: how })
  }

  const openHeading = (node: string, heading: string) => {
    const path = map.getNodePath(node)
    if (!path || !/^\d+$/.test(heading)) return
    deps.openNote(path, { title: getName(path), showing: 'current', line: Number(heading) })
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
  const declared = computed(() => (menu.value?.node === null ? [] : deps.getNoteCommands()))
  const menuItems = computed(() =>
    menu.value?.node === null ? NONE : createNodeItems(declared.value),
  )

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
    if (!OFFERED.has(id) && !declared.value.some((one) => one.id === id)) return
    const path = map.getNodePath(on.node)
    if (path) deps.runCommand(id, path, getName(path))
  }

  /** The icon of an item: the plex's own, or the one the window gives a declared command. */
  const getMenuIcon = (id: string): Component | null => findIcon(id) ?? deps.getCommandIcon(id)

  return { menu, menuItems, openMenu, chooseMenuItem, getMenuIcon }
}

function getPlexNodeName(neighbourhood: PlexView['neighbourhood']['value'], path: string): string {
  if (neighbourhood?.focus.path === path && neighbourhood.focus.title)
    return neighbourhood.focus.title
  const foundNode = neighbourhood?.related.find((one) => one.path === path)
  return foundNode?.title || getFileName(path).replace(/\.md$/, '')
}

export function createPlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()
  const {
    picture,
    empty,
    dragged,
    types,
    getType,
    drawn: drawnNodes,
  } = usePlexImage(view, deps, map)

  const getName = (path: string): string => getPlexNodeName(view.neighbourhood.value, path)
  const quickLinkModel = useQuickLink(view, deps, map)
  const { readHeadings, getHeadings } = usePlexHeadings(drawnNodes, types, deps, map)
  const linkInspectorModel = useLinkInspector(view, deps.editor, map, getName)

  const nodeActions = createPlexNodeActions({
    view,
    tabDeps: deps,
    map,
    getName,
    openQuickLink: quickLinkModel.openQuickLink,
  })
  const { menu, menuItems, openMenu, chooseMenuItem, getMenuIcon } = usePlexMenu({
    map,
    tabDeps: deps,
    getName,
    createNote: nodeActions.createNote,
  })

  const tabState: PlexTabState = {
    view,
    picture,
    empty,
    dragged,
    menu,
    menuItems,
    getMenuIcon,
    getNodeIcon: (node: string) => {
      const type = getType(node)
      return type === NOTE_TYPE ? null : deps.getNoteIcon(type)
    },
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

  return tabState
}

export const usePlexTab = createPlexTab
