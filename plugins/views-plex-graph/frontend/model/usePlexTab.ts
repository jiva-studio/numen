/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, watch, type Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import {
  formatErrorMessage,
  getFileName,
  getRelationErrorWords,
  NOTE_TYPE,
  useSlotMenu,
  type CommandTarget,
  type Caught,
} from '@numen/plugin-sdk'
import {
  convertNoteGraph,
  getNamePath,
  getNames,
  getNameToken,
  getNoteTypes,
  getTabKinds,
} from '../lib/picture'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { findRole } from '../lib/roles'
import { CREATE_NOTE } from '../lib/commands'
import { WORDS as words } from '../words'
import type {
  MenuRequest,
  PlexTabDeps,
  PlexTabState,
  QuickLinkRequest,
  PlexView,
  PathRename,
  NoteType,
  PlexGraph,
  PlexRelation,
  PlexDestination,
  PlexNames,
  PlexNamesState,
} from '../types'
import { useLinkInspector } from './useLinkInspector'
import { useQuickLink } from './useQuickLink'
import { usePlexHeadings } from './usePlexHeadings'

export type { PlexTabDeps, PlexTabState }

function usePlexPicture(
  view: PlexView,
  deps: PlexTabDeps,
  map: ReturnType<typeof createNodeIdMap>,
) {
  const picture = computed<PlexGraph | null>(() => {
    const noteGraph = view.noteGraph.value
    if (!deps.window.isReady.value || !noteGraph) return null
    return convertNoteGraph(noteGraph, map.getNodeId, deps.look.references.value)
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
      deps.window.isReady.value &&
      !view.here.value &&
      !deps.window.openingPath.value &&
      !view.noteGraph.value,
  )

  const dragged = computed<readonly string[]>(() => {
    const here = view.here.value
    if (!deps.window.isReady.value || !view.noteGraph.value || !here) return []
    return deps.window.dragged.value.filter((path) => path !== here)
  })

  return { view, picture, empty, dragged, creatable: deps.creatable }
}

function usePlexNoteKinds(view: PlexView, map: ReturnType<typeof createNodeIdMap>) {
  const types = computed<ReadonlyMap<string, NoteType>>(() => {
    const noteGraph = view.noteGraph.value
    return noteGraph ? getNoteTypes(noteGraph) : new Map()
  })

  const getType = (node: string): NoteType => types.value.get(map.getNodePath(node)) ?? NOTE_TYPE

  const tabKinds = computed<ReadonlyMap<string, string>>(() => {
    const noteGraph = view.noteGraph.value
    return noteGraph ? getTabKinds(noteGraph) : new Map()
  })

  const getTabKind = (node: string): string | undefined => tabKinds.value.get(map.getNodePath(node))

  const shownPaths = computed<readonly string[]>(() => {
    const noteGraph = view.noteGraph.value
    if (!noteGraph) return []
    const paths = [noteGraph.focus.path]
    for (const related of noteGraph.related) {
      if (!related.relatedTo) paths.push(related.path)
    }
    return paths.filter(Boolean)
  })

  return { types, getType, getTabKind, shownPaths }
}

function createPlexNodeActions(options: {
  view: PlexView
  tabDeps: PlexTabDeps
  map: ReturnType<typeof createNodeIdMap>
  getName: (path: string) => string
  openQuickLink: (request: QuickLinkRequest) => void
}) {
  const { view, tabDeps: deps, map, getName, openQuickLink } = options
  const goTo = (path: string) => {
    if (!path) return
    view.go(path).catch((error: Caught) => deps.window.showMessage(formatErrorMessage(error)))
    deps.window.onSelectPath?.(path, getName(path))
  }
  const activate = (node: string) => goTo(map.getNodePath(node))
  const chooseName = (name: string) => goTo(getNamePath(name))

  const addNode = async (from: string, relation: PlexRelation, pos?: { x: number; y: number }) => {
    if (pos) {
      openQuickLink({ from, relation, at: pos })
      return
    }
    const path = map.getNodePath(from)
    const role = findRole(deps.creatable, relation)
    if (!path || !role) return
    const made = await deps.editor.createRelated({ from: path, role })
    if (!made.ok) return deps.window.showMessage(getRelationErrorWords(made.error))
    await view.go(path)
  }

  const joinNodes = async (from: string, to: string, relation: PlexRelation) => {
    const one = map.getNodePath(from)
    const other = map.getNodePath(to)
    const role = findRole(deps.creatable, relation)
    if (!one || !other || !role) return
    const joined = await deps.editor.join(one, other, { role })
    if (!joined.ok) return deps.window.showMessage(getRelationErrorWords(joined.error))
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

    if (notJoined.length > 0) deps.window.showMessage(`${words.notJoined} ${notJoined.join(', ')}`)
    if (hasJoinedAny) await view.go(here)
  }

  const openNode = (node: string, how: PlexDestination = 'current') => {
    const path = map.getNodePath(node)
    if (path) deps.window.openNote(path, { title: getName(path), showing: how })
  }

  const followMoves = (renames: readonly PathRename[]) => {
    view.followMoves(renames)
    map.updateRenamedNodes(renames)
  }

  const openHeading = (node: string, heading: string) => {
    const path = map.getNodePath(node)
    if (!path || !/^\d+$/.test(heading)) return
    deps.window.openNote(path, { title: getName(path), showing: 'current', line: Number(heading) })
  }

  return {
    activate,
    addNode,
    joinNodes,
    dropNodes,
    openNode,
    openHeading,
    chooseName,
    followMoves,
    getName,
  }
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
  getTabKind: (node: string) => string | undefined
  showCreated: (path: string) => Promise<void>
  dismissOthers: () => void
}) {
  const { map, tabDeps: deps, getName, getType, getTabKind, showCreated, dismissOthers } = options
  const reportFailure = (error: Caught): void => deps.window.showMessage(formatErrorMessage(error))
  const nodeMenu = useSlotMenu<NodeMenuTarget>({
    listItems: (command, { target }) => deps.menu.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const request = computed(() => nodeMenu.target.value?.request ?? null)
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

  const dismiss = () => {
    close()
    dismissOthers()
  }

  const openMenu = (menuRequest: MenuRequest) => {
    close()
    const path = menuRequest.node === null ? '' : map.getNodePath(menuRequest.node)
    if (menuRequest.node !== null && !path) return
    const target =
      menuRequest.node === null ? getCanvasTarget() : getNodeTarget(menuRequest.node, path ?? '')
    nodeMenu.open(() => ({
      target: { request: menuRequest, target },
      entries: deps.menu.getMenuEntries(target),
    }))
  }

  const chooseMenuItem = (id: string) => {
    const choice = nodeMenu.pick(id)
    if (!choice) return
    deps.menu
      .runCommand(choice.command, choice.target.target, choice.itemId, choice.values)
      .then((made) => (choice.command === CREATE_NOTE && made ? showCreated(made) : undefined))
      .catch(reportFailure)
  }

  /** The icon of a row, which is the one of the command or submenu of its entry. */
  const getMenuIcon = (id: string): Component | null =>
    (nodeMenu.getIcon(id) as Component | undefined) ?? null

  /** The icon a node is shownPaths with, and nothing for an ordinary note. */
  const getNodeIcon = (node: string): Component | null => {
    const tabKind = getTabKind(node)
    return tabKind === undefined ? null : deps.window.getNoteIcon(tabKind)
  }

  return { request, menuItems, getMenuIcon, getNodeIcon, openMenu, dismiss, chooseMenuItem }
}

const NO_NAMES: PlexNames = { above: [], below: [] }

function usePlexNames(
  view: PlexView,
  deps: PlexTabDeps,
  map: ReturnType<typeof createNodeIdMap>,
): PlexNamesState {
  const { showNames, namesPerNote } = deps.names
  const names = computed(() => {
    const noteGraph = view.noteGraph.value
    if (!noteGraph || !showNames.value) return new Map<string, PlexNames>()
    return getNames(noteGraph, namesPerNote.value, getNameToken)
  })
  const getNodeNames = (node: string): PlexNames =>
    names.value.get(getNameToken(map.getNodePath(node))) ?? NO_NAMES
  return { showNames, getNames: getNodeNames }
}

function getPlexNodeName(noteGraph: PlexView['noteGraph']['value'], path: string): string {
  if (noteGraph?.focus.path === path && noteGraph.focus.title) return noteGraph.focus.title
  const foundNode = noteGraph?.related.find((one) => one.path === path)
  return foundNode?.title || getFileName(path).replace(/\.md$/, '')
}

export function createPlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()
  const canvas = usePlexPicture(view, deps, map)
  const { types, getType, getTabKind, shownPaths } = usePlexNoteKinds(view, map)

  const getName = (path: string): string => getPlexNodeName(view.noteGraph.value, path)
  const quickLink = useQuickLink(view, deps, map)
  const headings = usePlexHeadings(shownPaths, types, deps, map)
  const names = usePlexNames(view, deps, map)
  const inspector = useLinkInspector(
    view,
    { editor: deps.editor, showMessage: deps.window.showMessage },
    map,
    getName,
  )

  const nodes = createPlexNodeActions({
    view,
    tabDeps: deps,
    map,
    getName,
    openQuickLink: quickLink.openQuickLink,
  })
  const menu = usePlexMenu({
    map,
    tabDeps: deps,
    getName,
    getType,
    getTabKind,
    showCreated: (path) => view.go(path),
    dismissOthers: () => {
      quickLink.dismissQuickLink()
      inspector.dismissLinkInspector()
    },
  })

  const reportFailure = (error: Caught): void => deps.window.showMessage(formatErrorMessage(error))

  return {
    reportFailure,
    canvas,
    menu,
    headings,
    names,
    nodes,
    quickLink,
    inspector,
    look: deps.look,
  }
}
