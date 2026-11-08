/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, ref, watch, type Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import type { PlexNeighbourhood, PlexRelation, PlexDestination } from '../plex'
import {
  formatErrorMessage,
  getFileName,
  getRelationErrorWords,
  NOTE_TYPE,
  useSlotMenu,
  type CommandTarget,
} from '@numen/plugin-sdk'
import { NEW_NOTE, NONE } from '../lib/menu'
import { convertNeighbourhood, getNoteTypes } from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { findRole } from '../lib/roles'
import { WORDS as words } from '../words'
import type {
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
  LinkInspectorRequest,
  MenuRequest,
  PlexEditor,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
}

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
      view.go(path).catch((error: unknown) => deps.showMessage(formatErrorMessage(error)))
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
    const role = findRole(deps.creatable, relation)
    if (!path || !role) return
    const made = await deps.editor.createRelated({ from: path, role })
    if (!made.ok) return deps.showMessage(getRelationErrorWords(made.error))
    await view.go(path)
  }

  const joinNodes = async (from: string, to: string, relation: PlexRelation) => {
    const one = map.getNodePath(from)
    const other = map.getNodePath(to)
    const role = findRole(deps.creatable, relation)
    if (!one || !other || !role) return
    const joined = await deps.editor.join(one, other, { role })
    if (!joined.ok) return deps.showMessage(getRelationErrorWords(joined.error))
    await view.go(one)
  }

  const dropNodes = async (nodes: readonly string[], relation: PlexRelation) => {
    const here = view.here.value
    const role = findRole(deps.creatable, relation)
    if (!here || !role) return

    const notJoined: string[] = []
    let hasJoinedAny = false

    for (const path of nodes) {
      if (path === here) continue
      if ((await deps.editor.join(here, path, { role })).ok) hasJoinedAny = true
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

/** The words of the rows the menu on a node writes itself. */
const MENU_WORDS = { loadingText: words.loadingItems }

/** What the menu on a node is over: where it was requested, and the note of the node. */
interface NodeMenuTarget {
  readonly request: MenuRequest
  readonly target: CommandTarget
}

function usePlexMenu(options: {
  map: ReturnType<typeof createNodeIdMap>
  tabDeps: PlexTabDeps
  getName: (path: string) => string
  getType: (node: string) => NoteType
  createNote: () => Promise<void>
}) {
  const { map, tabDeps: deps, getName, getType, createNote } = options
  const reportFailure = (error: unknown): void => deps.showMessage(formatErrorMessage(error))
  const canvasMenu = ref<MenuRequest | null>(null)
  const nodeMenu = useSlotMenu<NodeMenuTarget>({
    listItems: (command, { target }) => deps.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const menu = computed(() => canvasMenu.value ?? nodeMenu.target.value?.request ?? null)
  const menuItems = computed<readonly MenuItem[]>(() =>
    canvasMenu.value ? NONE : (nodeMenu.menu.value?.items ?? []),
  )

  const getNodeTarget = (node: string, path: string): CommandTarget => {
    const type = getType(node)
    const target = { path, title: getName(path), isFolder: false, others: [], tab: '' }
    return type === NOTE_TYPE ? target : { ...target, noteType: type }
  }

  const close = () => {
    canvasMenu.value = null
    nodeMenu.close()
  }

  const openMenu = (request: MenuRequest) => {
    close()
    if (request.node === null) {
      canvasMenu.value = request
      return
    }
    const path = map.getNodePath(request.node)
    if (!path) return
    const target = getNodeTarget(request.node, path)
    nodeMenu.open(() => ({ target: { request, target }, entries: deps.getMenuEntries(target) }))
  }

  const chooseMenuItem = (id: string) => {
    if (canvasMenu.value) {
      close()
      if (id === NEW_NOTE) createNote().catch(reportFailure)
      return
    }
    const choice = nodeMenu.pick(id)
    if (!choice) return
    deps.runCommand(choice.command, choice.target.target, choice.argument).catch(reportFailure)
  }

  /** The icon of a row, which is the one of the command or submenu of its entry. */
  const getMenuIcon = (id: string): Component | null =>
    (nodeMenu.getIcon(id) as Component | undefined) ?? null

  return { menu, menuItems, openMenu, chooseMenuItem, getMenuIcon, close }
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
  const linkInspectorModel = useLinkInspector(
    view,
    { editor: deps.editor, showMessage: deps.showMessage },
    map,
    getName,
  )

  const nodeActions = createPlexNodeActions({
    view,
    tabDeps: deps,
    map,
    getName,
    openQuickLink: quickLinkModel.openQuickLink,
  })
  const { menu, menuItems, openMenu, chooseMenuItem, getMenuIcon, close } = usePlexMenu({
    map,
    tabDeps: deps,
    getName,
    getType,
    createNote: nodeActions.createNote,
  })

  const reportFailure = (error: unknown): void => deps.showMessage(formatErrorMessage(error))

  const tabState: PlexTabState = {
    view,
    reportFailure,
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
      close()
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
