/**
 * What a workspace comes to after a gesture.
 *
 * Every function takes a workspace and gives back another; nothing is changed
 * in place. A gesture that asks for something the tree does not hold gives the
 * workspace back untouched.
 */
import {
  branch,
  isBranch,
  isPane,
  leads,
  nodeAt,
  normalize,
  orientationAt,
  orientationOf,
  pane,
  paneById,
  paneWithTab,
  panesOf,
  pathTo,
  replaceAt,
  withChildren,
  type NodeId,
  type Orientation,
  type Pane,
  type Side,
  type TabId,
  type Workspace,
  type WorkspaceNode,
} from './model'
import { insert } from './model/shares'

/** Where an identity for a pane or a branch a gesture makes comes from. */
export type Naming = () => NodeId

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
    root: mapPane(workspace.root, holder.id, (held) => ({ ...held, active: tab })),
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

  const target = paneById(workspace.root, into) ?? panesOf(workspace.root)[0]
  if (!target) return workspace

  return {
    ...workspace,
    root: mapPane(workspace.root, target.id, (held) => ({
      ...held,
      tabs: [...held.tabs, tab],
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
export function openTabBeside(
  workspace: Workspace,
  tab: TabId,
  side: Side,
  naming: Naming,
  onto: NodeId = workspace.focus,
): Workspace {
  const target = paneById(workspace.root, onto) ?? panesOf(workspace.root)[0]
  if (!target || side === 'center') return workspace

  const source = paneWithTab(workspace.root, tab)
  if (source && source.id === target.id && source.tabs.length === 1) {
    return activateTab(workspace, tab)
  }

  const root = detach(workspace.root, tab)
  const landed = beside(root, target.id, tab, side, workspace.axis, naming)
  return settle({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

export function closeTab(workspace: Workspace, tab: TabId): Workspace {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return workspace
  return settle({ ...workspace, root: detach(workspace.root, tab) }, holder.id)
}

/**
 * A tab moved along its own strip. `slot` is counted in gaps between tabs, as
 * `slotAt` reports them.
 */
export function moveTabWithin(workspace: Workspace, tab: TabId, slot: number): Workspace {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return workspace

  const from = holder.tabs.indexOf(tab)
  const rest = holder.tabs.filter((each) => each !== tab)
  const at = Math.max(0, Math.min(slot > from ? slot - 1 : slot, rest.length))
  const tabs = [...rest.slice(0, at), tab, ...rest.slice(at)]

  return {
    ...workspace,
    root: mapPane(workspace.root, holder.id, (held) => ({ ...held, tabs })),
    focus: holder.id,
  }
}

/**
 * A tab let go on a pane: it joins the stack, or it takes half the pane and
 * the two share the room.
 *
 * A tab let go where it started, with nowhere else to go, is only shown.
 */
export function dropTab(workspace: Workspace, drop: TabDrop, naming: Naming): Workspace {
  const target = paneById(workspace.root, drop.onto)
  const source = paneWithTab(workspace.root, drop.tab)
  if (!target || !source) return workspace

  if (source.id === target.id && (drop.side === 'center' || source.tabs.length === 1)) {
    return activateTab(workspace, drop.tab)
  }

  const root = detach(workspace.root, drop.tab)

  if (drop.side === 'center') {
    const joined = mapPane(root, target.id, (held) => ({
      ...held,
      tabs: [...held.tabs, drop.tab],
      active: drop.tab,
    }))
    return settle({ ...workspace, root: joined }, target.id)
  }

  const landed = beside(root, target.id, drop.tab, drop.side, workspace.axis, naming)
  return settle({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

/** A tab let go on the outer edge divides the whole workspace. */
export function dropOnEdge(
  workspace: Workspace,
  tab: TabId,
  side: Side,
  naming: Naming,
): Workspace {
  if (!paneWithTab(workspace.root, tab) || side === 'center') return workspace

  const root = detach(workspace.root, tab)
  const landed = beside(root, root.id, tab, side, workspace.axis, naming)
  return settle({ ...workspace, root: landed.root, axis: landed.axis }, landed.focus)
}

export function resizeBranch(
  workspace: Workspace,
  id: NodeId,
  sizes: readonly number[],
): Workspace {
  const path = pathTo(workspace.root, id)
  if (!path) return workspace

  const node = nodeAt(workspace.root, path)
  if (!node || !isBranch(node)) return workspace

  return {
    ...workspace,
    root: replaceAt(workspace.root, path, withChildren(node, node.children, sizes)),
  }
}

interface Landed {
  readonly root: WorkspaceNode
  readonly axis: Orientation
  readonly focus: NodeId
}

/**
 * A new pane put alongside a node, taking half its share.
 *
 * Where the parent divides its length the way the side asks, the pane joins
 * it as a neighbour; where it does not, the node is wrapped in a branch, which
 * lands one level deeper and divides the other way. A node with no parent is
 * the root, and the workspace turns its axis to suit.
 */
function beside(
  root: WorkspaceNode,
  onto: NodeId,
  tab: TabId,
  side: Side,
  axis: Orientation,
  id: Naming,
): Landed {
  const wanted = orientationOf(side)
  const path = pathTo(root, onto)
  if (!wanted || !path) return { root, axis, focus: onto }

  const made = pane(id(), [tab])
  const index = path[path.length - 1]
  const above = path.slice(0, -1)

  if (index === undefined) {
    const pair = leads(side) ? [made, root] : [root, made]
    return { root: branch(id(), pair, [0.5, 0.5]), axis: wanted, focus: made.id }
  }

  const parent = nodeAt(root, above)
  if (!parent || !isBranch(parent)) return { root, axis, focus: onto }

  if (orientationAt(axis, above.length) === wanted) {
    const at = leads(side) ? index : index + 1
    const children = [...parent.children.slice(0, at), made, ...parent.children.slice(at)]
    const sizes = insert(parent.sizes, index, leads(side))
    return {
      root: replaceAt(root, above, withChildren(parent, children, sizes)),
      axis,
      focus: made.id,
    }
  }

  const target = nodeAt(root, path)
  if (!target) return { root, axis, focus: onto }

  const pair = leads(side) ? [made, target] : [target, made]
  return {
    root: replaceAt(root, path, branch(id(), pair, [0.5, 0.5])),
    axis,
    focus: made.id,
  }
}

/**
 * The tab taken off whichever pane holds it. The pane is left where it is,
 * empty if that was its last tab, and `settle` is what clears it away.
 */
function detach(root: WorkspaceNode, tab: TabId): WorkspaceNode {
  const holder = paneWithTab(root, tab)
  if (!holder) return root

  return mapPane(root, holder.id, (held) => {
    const at = held.tabs.indexOf(tab)
    const tabs = held.tabs.filter((each) => each !== tab)
    const active = held.active === tab ? (tabs[at] ?? tabs[at - 1] ?? null) : held.active
    return { ...held, tabs, active }
  })
}

/** The canonical tree, with the focus on a pane that survived it. */
function settle(workspace: Workspace, preferred: NodeId): Workspace {
  const settled = normalize(workspace)
  const panes = panesOf(settled.root)
  const kept =
    panes.find((each) => each.id === preferred) ??
    panes.find((each) => each.id === workspace.focus) ??
    panes[0]

  return kept ? { ...settled, focus: kept.id } : settled
}

function mapPane(
  root: WorkspaceNode,
  id: NodeId,
  change: (pane: Pane) => Pane,
): WorkspaceNode {
  const path = pathTo(root, id)
  if (!path) return root

  const node = nodeAt(root, path)
  if (!node || !isPane(node)) return root

  return replaceAt(root, path, change(node))
}
