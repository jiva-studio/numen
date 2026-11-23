/**
 * Tab state, window registration, and interactions for an open plex graph tab.
 */
import { computed, getCurrentScope, onScopeDispose, shallowRef, watch, type Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import {
  formatErrorMessage,
  getFileName,
  getRelationErrorWords,
  getRenamedPath,
  NOTE_TYPE,
  useSlotMenu,
  type CommandTarget,
} from '@numen/plugin-sdk'
import { convertNoteGraph, getNoteTypes, getTabKinds } from '../lib/picture'
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
  PlexBoardContext,
  PlexWidget,
  PlexWidgetSide,
  PlexWidgetsState,
} from '../types'
import { useLinkInspector } from './useLinkInspector'
import { useQuickLink } from './useQuickLink'

export type { PlexTabDeps, PlexTabState }

function usePlexPicture(
  view: PlexView,
  deps: PlexTabDeps,
  map: ReturnType<typeof createNodeIdMap>,
  getName: (path: string) => string,
) {
  const picture = computed<PlexGraph | null>(() => {
    const noteGraph = view.noteGraph.value
    if (!deps.window.isReady.value || !noteGraph) return null
    return convertNoteGraph(noteGraph, map.getNodeId, deps.look.references.value)
  })

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

  const widgets = usePlexWidgets(deps, map, {
    key: deps.tabId,
    here: view.here,
    go: (path) => moveFocus({ view, deps, getName }, path),
  })

  return { view, picture, empty, dragged, creatable: deps.creatable, widgets }
}

/** What the tab last learnt of a note: its type and the tab kind the core chose for it. */
interface KnownKind {
  readonly type: NoteType
  readonly tabKind: string | undefined
}

function usePlexNoteKinds(view: PlexView, map: ReturnType<typeof createNodeIdMap>) {
  const known = shallowRef<ReadonlyMap<string, KnownKind>>(new Map())

  watch(
    () => view.noteGraph.value,
    (noteGraph) => {
      if (!noteGraph) return
      const tabKinds = getTabKinds(noteGraph)
      const next = new Map(known.value)
      for (const [path, type] of getNoteTypes(noteGraph)) {
        next.set(path, { type, tabKind: tabKinds.get(path) })
      }
      known.value = next
    },
    { immediate: true, flush: 'sync' },
  )

  const followMoves = (renames: readonly PathRename[]): void => {
    known.value = new Map(
      [...known.value].map(([path, kind]) => [getRenamedPath(renames, path) || path, kind]),
    )
  }

  const getType = (node: string): NoteType =>
    known.value.get(map.getNodePath(node))?.type ?? NOTE_TYPE

  const getTabKind = (node: string): string | undefined =>
    known.value.get(map.getNodePath(node))?.tabKind

  return { getType, getTabKind, followMoves }
}

/** What moving the focus to a note needs. */
interface FocusMover {
  readonly view: PlexView
  readonly deps: PlexTabDeps
  readonly getName: (path: string) => string
}

function moveFocus({ view, deps, getName }: FocusMover, path: string): void {
  if (!path) return
  view.go(path).catch((error: unknown) => deps.window.showMessage(formatErrorMessage(error)))
  deps.window.onSelectPath?.(path, getName(path))
}

function createPlexNodeActions(options: {
  view: PlexView
  tabDeps: PlexTabDeps
  map: ReturnType<typeof createNodeIdMap>
  getName: (path: string) => string
  openQuickLink: (request: QuickLinkRequest) => void
  followKindMoves: (renames: readonly PathRename[]) => void
}) {
  const { view, tabDeps: deps, map, getName, openQuickLink, followKindMoves } = options
  const activate = (node: string) => moveFocus({ view, deps, getName }, map.getNodePath(node))

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
    await view.go(deps.look.activateLinked.value ? other : one)
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
    view.lifecycle.followMoves(renames)
    map.updateRenamedNodes(renames)
    followKindMoves(renames)
  }

  return {
    activate,
    addNode,
    joinNodes,
    dropNodes,
    openNode,
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
  const reportFailure = (error: unknown): void => deps.window.showMessage(formatErrorMessage(error))
  const nodeMenu = useSlotMenu<NodeMenuTarget>({
    listItems: (command, { target }) => deps.menu.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const request = computed(() => nodeMenu.target.value?.request ?? null)
  const menuItems = computed<readonly MenuItem[]>(() => nodeMenu.menu.value?.items ?? [])

  const getNodeTarget = (node: string, path: string): CommandTarget => ({
    path,
    title: getName(path),
    isFolder: false,
    others: [],
    tab: '',
    noteType: getType(node),
  })

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

  const makeNote = async (): Promise<void> => {
    const made = await deps.menu.runCommand(CREATE_NOTE, getCanvasTarget())
    if (made) await showCreated(made)
  }

  return {
    request,
    menuItems,
    getMenuIcon,
    getNodeIcon,
    openMenu,
    dismiss,
    chooseMenuItem,
    makeNote,
  }
}

function usePlexWidgets(
  deps: PlexTabDeps,
  map: ReturnType<typeof createNodeIdMap>,
  board: PlexBoardContext,
): PlexWidgetsState {
  const watchSide = (side: PlexWidgetSide) => {
    const entries = shallowRef<readonly PlexWidget[]>([])
    const handle = deps.watchWidgets?.(side, (items) => {
      entries.value = items.flatMap((item) =>
        item.kind === 'view' ? [{ key: item.key, component: item.component }] : [],
      )
    })
    if (handle && getCurrentScope()) onScopeDispose(() => handle.dispose())
    return entries
  }
  return {
    above: watchSide('above'),
    below: watchSide('below'),
    over: watchSide('over'),
    boardTop: watchSide('boardTop'),
    boardBottom: watchSide('boardBottom'),
    board,
    getPath: map.getNodePath,
  }
}

function getPlexNodeName(noteGraph: PlexView['noteGraph']['value'], path: string): string {
  if (noteGraph?.focus.path === path && noteGraph.focus.title) return noteGraph.focus.title
  const foundNode = noteGraph?.related.find((one) => one.path === path)
  return foundNode?.title || getFileName(path).replace(/\.md$/, '')
}

export function createPlexTab(view: PlexView, deps: PlexTabDeps): PlexTabState {
  const map = createNodeIdMap()
  const getName = (path: string): string => getPlexNodeName(view.noteGraph.value, path)
  const canvas = usePlexPicture(view, deps, map, getName)
  const { getType, getTabKind, followMoves } = usePlexNoteKinds(view, map)

  const quickLink = useQuickLink(view, deps, map)
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
    followKindMoves: followMoves,
  })
  const menu = usePlexMenu({
    map,
    tabDeps: deps,
    getName,
    getType,
    getTabKind,
    showCreated: (path) =>
      view.go(deps.look.activateCreated.value || !view.here.value ? path : view.here.value),
    dismissOthers: () => {
      quickLink.dismissQuickLink()
      inspector.dismissLinkInspector()
    },
  })

  const reportFailure = (error: unknown): void => deps.window.showMessage(formatErrorMessage(error))

  return {
    reportFailure,
    canvas,
    menu,
    nodes,
    quickLink,
    inspector,
    look: deps.look,
  }
}
