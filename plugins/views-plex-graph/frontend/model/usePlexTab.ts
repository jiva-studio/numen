/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, watch, type Component } from 'vue'
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
import { convertNeighbourhood, getNoteTypes, getTabKinds } from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { findRole } from '../lib/roles'
import { CREATE_NOTE } from '../lib/commands'
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

  const tabKinds = computed<ReadonlyMap<string, string>>(() => {
    const neighbourhood = view.neighbourhood.value
    return neighbourhood ? getTabKinds(neighbourhood) : new Map()
  })

  const getTabKind = (node: string): string | undefined => tabKinds.value.get(map.getNodePath(node))

  const activePaths = computed<readonly string[]>(() => {
    const neighbourhood = view.neighbourhood.value
    if (!neighbourhood) return []
    const paths = [neighbourhood.focus.path]
    for (const related of neighbourhood.related) paths.push(related.path)
    return paths.filter(Boolean)
  })

  return { picture, empty, dragged, types, getType, getTabKind, drawn: activePaths }
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

  return { activate, createNode, joinNodes, dropNodes, openNode, openHeading }
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
  showCreated: (path: string) => Promise<void>
}) {
  const { map, tabDeps: deps, getName, getType, showCreated } = options
  const reportFailure = (error: unknown): void => deps.showMessage(formatErrorMessage(error))
  const nodeMenu = useSlotMenu<NodeMenuTarget>({
    listItems: (command, { target }) => deps.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const menu = computed(() => nodeMenu.target.value?.request ?? null)
  const menuItems = computed<readonly MenuItem[]>(() => nodeMenu.menu.value?.items ?? [])

  const getNodeTarget = (node: string, path: string): CommandTarget => {
    const type = getType(node)
    const target = { path, title: getName(path), isFolder: false, others: [], tab: '' }
    return type === NOTE_TYPE ? target : { ...target, noteType: type }
  }

  /** The vault root over this tab: the target of a place where there is no node. */
  const getCanvasTarget = (): CommandTarget => ({
    path: '',
    title: '',
    isFolder: true,
    others: [],
    tab: deps.tabId,
  })

  const close = () => nodeMenu.close()

  const openMenu = (request: MenuRequest) => {
    close()
    const path = request.node === null ? '' : map.getNodePath(request.node)
    if (request.node !== null && !path) return
    const target =
      request.node === null ? getCanvasTarget() : getNodeTarget(request.node, path ?? '')
    nodeMenu.open(() => ({ target: { request, target }, entries: deps.getMenuEntries(target) }))
  }

  const chooseMenuItem = (id: string) => {
    const choice = nodeMenu.pick(id)
    if (!choice) return
    deps
      .runCommand(choice.command, choice.target.target, choice.itemId, choice.values)
      .then((made) => (choice.command === CREATE_NOTE && made ? showCreated(made) : undefined))
      .catch(reportFailure)
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
    getTabKind,
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
    showCreated: (path) => view.go(path),
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
      const tabKind = getTabKind(node)
      return tabKind === undefined ? null : deps.getNoteIcon(tabKind)
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
