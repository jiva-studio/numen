/**
 * What a workspace evaluates to after a gesture.
 *
 * Every function takes a workspace and gives back another; nothing is changed
 * in place. A gesture that asks for something the tree does not hold gives the
 * workspace back untouched.
 */
import type { NodeId, Side, TabId, Workspace } from './node'
import {
  isBranch,
  findNode,
  paneById,
  paneWithTab,
  getPanes,
  pathTo,
  replaceAt,
  withChildren,
} from './tree'
import { detach, mapPane, putBeside, commit, type NodeIdFactory } from './splice'

export type { NodeIdFactory }

export interface TabDrop {
  readonly tab: TabId
  /** The pane it was let go on. */
  readonly onto: NodeId
  readonly side: Side
}

export function focusPane(workspace: Workspace, id: NodeId): Workspace {
  return paneById(workspace.root, id) ? { ...workspace, focus: id } : workspace
}

export function activateTab(workspace: Workspace, tab: TabId): Workspace {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return workspace

  return {
    ...workspace,
    root: mapPane(workspace.root, holder.id, (pane) => ({ ...pane, active: tab })),
    focus: holder.id,
  }
}

/** A tab already open is shown where it is. */
export function openTab(
  workspace: Workspace,
  tab: TabId,
  into: NodeId = workspace.focus,
): Workspace {
  if (paneWithTab(workspace.root, tab)) return activateTab(workspace, tab)

  const target = paneById(workspace.root, into) ?? getPanes(workspace.root)[0]
  if (!target) return workspace

  return {
    ...workspace,
    root: mapPane(workspace.root, target.id, (pane) => ({
      ...pane,
      tabs: [...pane.tabs, tab],
      active: tab,
    })),
    focus: target.id,
  }
}

/**
 * A tab opened in a pane of its own, alongside another one. A tab already
 * open is taken from wherever it was, and one that is alone in the pane it
 * would be put beside is only shown.
 *
 * The middle is not a side, and a tab asked for there stays where it is.
 */
export interface OpenBesidePlacement {
  readonly side: Side
  readonly createId: NodeIdFactory
  readonly onto?: NodeId
}

/**
 * A tab opened in a pane of its own, alongside another one. A tab already
 * open is taken from wherever it was, and one that is alone in the pane it
 * would be put beside is only shown.
 *
 * The middle is not a side, and a tab asked for there stays where it is.
 */
export function openTabBeside(
  workspace: Workspace,
  tab: TabId,
  placement: OpenBesidePlacement,
): Workspace {
  const { side, createId, onto = workspace.focus } = placement
  const target = paneById(workspace.root, onto) ?? getPanes(workspace.root)[0]
  if (!target || side === 'center') return workspace

  const source = paneWithTab(workspace.root, tab)
  if (source && source.id === target.id && source.tabs.length === 1) {
    return activateTab(workspace, tab)
  }

  const root = detach(workspace.root, tab)
  const landed = putBeside(root, { onto: target.id, tab, side, axis: workspace.axis }, createId)
  return commit({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

export function closeTab(workspace: Workspace, tab: TabId): Workspace {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return workspace
  return commit({ ...workspace, root: detach(workspace.root, tab) }, holder.id)
}

/**
 * A tab moved along its own strip. `slot` is counted in gaps between tabs, as
 * `getSlot` reports them.
 */
export function moveTabWithin(workspace: Workspace, tab: TabId, slot: number): Workspace {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return workspace

  const from = holder.tabs.indexOf(tab)
  const rest = holder.tabs.filter((each) => each !== tab)
  const targetIndex = Math.max(0, Math.min(slot > from ? slot - 1 : slot, rest.length))
  const tabs = [...rest.slice(0, targetIndex), tab, ...rest.slice(targetIndex)]

  return {
    ...workspace,
    root: mapPane(workspace.root, holder.id, (pane) => ({ ...pane, tabs })),
    focus: holder.id,
  }
}

/**
 * A tab let go on a pane: it joins the stack, or it takes half the pane and
 * the two share the room.
 *
 * A tab let go where it started, with nowhere else to go, is only shown.
 */
export function dropTab(workspace: Workspace, drop: TabDrop, createId: NodeIdFactory): Workspace {
  const target = paneById(workspace.root, drop.onto)
  const source = paneWithTab(workspace.root, drop.tab)
  if (!target || !source) return workspace

  if (source.id === target.id && (drop.side === 'center' || source.tabs.length === 1)) {
    return activateTab(workspace, drop.tab)
  }

  const root = detach(workspace.root, drop.tab)

  if (drop.side === 'center') {
    const joined = mapPane(root, target.id, (pane) => ({
      ...pane,
      tabs: [...pane.tabs, drop.tab],
      active: drop.tab,
    }))
    return commit({ ...workspace, root: joined }, target.id)
  }

  const landed = putBeside(
    root,
    { onto: target.id, tab: drop.tab, side: drop.side, axis: workspace.axis },
    createId,
  )
  return commit({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

export interface EdgeDrop {
  readonly tab: TabId
  readonly side: Side
}

/** A tab let go on the outer edge divides the whole workspace. */
export function dropOnEdge(
  workspace: Workspace,
  drop: EdgeDrop,
  createId: NodeIdFactory,
): Workspace {
  const { tab, side } = drop
  if (!paneWithTab(workspace.root, tab) || side === 'center') return workspace

  const root = detach(workspace.root, tab)
  const landed = putBeside(root, { onto: root.id, tab, side, axis: workspace.axis }, createId)
  return commit({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

export function resizeBranch(
  workspace: Workspace,
  id: NodeId,
  sizes: readonly number[],
): Workspace {
  const path = pathTo(workspace.root, id)
  if (!path) return workspace

  const node = findNode(workspace.root, path)
  if (!node || !isBranch(node)) return workspace

  return {
    ...workspace,
    root: replaceAt(workspace.root, path, withChildren(node, node.children, sizes)),
  }
}
