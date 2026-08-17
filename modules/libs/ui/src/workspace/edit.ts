/**
 * What a workspace comes to after a gesture.
 *
 * Every function takes a workspace and gives back another; nothing is changed
 * in place. A gesture that asks for something the tree does not hold gives the
 * workspace back untouched.
 */
import {
  branch,
  group,
  groupById,
  groupWithTab,
  groupsOf,
  isBranch,
  isGroup,
  leads,
  nodeAt,
  normalize,
  orientationAt,
  orientationOf,
  pathTo,
  replaceAt,
  withChildren,
  type Group,
  type NodeId,
  type Orientation,
  type Side,
  type TabId,
  type Workspace,
  type WorkspaceNode,
} from './model'
import { insert } from './model/shares'

/** Where an identity for a group or a branch a gesture makes comes from. */
export interface Naming {
  readonly id: () => NodeId
}

export interface TabDrop {
  readonly tab: TabId
  /** The group it was let go on. */
  readonly onto: NodeId
  readonly side: Side
}

export function focusGroup(workspace: Workspace, id: NodeId): Workspace {
  return groupById(workspace.root, id) ? { ...workspace, focus: id } : workspace
}

export function activateTab(workspace: Workspace, tab: TabId): Workspace {
  const holder = groupWithTab(workspace.root, tab)
  if (!holder) return workspace

  return {
    ...workspace,
    root: mapGroup(workspace.root, holder.id, (held) => ({ ...held, active: tab })),
    focus: holder.id,
  }
}

/** A tab already open is shown where it is. */
export function openTab(
  workspace: Workspace,
  tab: TabId,
  into: NodeId = workspace.focus,
): Workspace {
  if (groupWithTab(workspace.root, tab)) return activateTab(workspace, tab)

  const target = groupById(workspace.root, into) ?? groupsOf(workspace.root)[0]
  if (!target) return workspace

  return {
    ...workspace,
    root: mapGroup(workspace.root, target.id, (held) => ({
      ...held,
      tabs: [...held.tabs, tab],
      active: tab,
    })),
    focus: target.id,
  }
}

export function closeTab(workspace: Workspace, tab: TabId): Workspace {
  const holder = groupWithTab(workspace.root, tab)
  if (!holder) return workspace
  return settle({ ...workspace, root: detach(workspace.root, tab) }, holder.id)
}

/**
 * A tab moved along its own strip. `slot` is counted in gaps between tabs, as
 * `slotAt` reports them.
 */
export function moveTabWithin(workspace: Workspace, tab: TabId, slot: number): Workspace {
  const holder = groupWithTab(workspace.root, tab)
  if (!holder) return workspace

  const from = holder.tabs.indexOf(tab)
  const rest = holder.tabs.filter((each) => each !== tab)
  const at = Math.max(0, Math.min(slot > from ? slot - 1 : slot, rest.length))
  const tabs = [...rest.slice(0, at), tab, ...rest.slice(at)]

  return {
    ...workspace,
    root: mapGroup(workspace.root, holder.id, (held) => ({ ...held, tabs })),
    focus: holder.id,
  }
}

/**
 * A tab let go on a group: it joins the stack, or it takes half the group and
 * the two share the room.
 *
 * A tab let go where it started, with nowhere else to go, is only shown.
 */
export function dropTab(workspace: Workspace, drop: TabDrop, naming: Naming): Workspace {
  const target = groupById(workspace.root, drop.onto)
  const source = groupWithTab(workspace.root, drop.tab)
  if (!target || !source) return workspace

  if (source.id === target.id && (drop.side === 'center' || source.tabs.length === 1)) {
    return activateTab(workspace, drop.tab)
  }

  const root = detach(workspace.root, drop.tab)

  if (drop.side === 'center') {
    const joined = mapGroup(root, target.id, (held) => ({
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
  if (!groupWithTab(workspace.root, tab) || side === 'center') return workspace

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
 * A new group put alongside a node, taking half its share.
 *
 * Where the parent divides its length the way the side asks, the group joins
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
  { id }: Naming,
): Landed {
  const wanted = orientationOf(side)
  const path = pathTo(root, onto)
  if (!wanted || !path) return { root, axis, focus: onto }

  const made = group(id(), [tab])
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
 * The tab taken off whichever group holds it. The group is left where it is,
 * empty if that was its last tab, and `settle` is what clears it away.
 */
function detach(root: WorkspaceNode, tab: TabId): WorkspaceNode {
  const holder = groupWithTab(root, tab)
  if (!holder) return root

  return mapGroup(root, holder.id, (held) => {
    const at = held.tabs.indexOf(tab)
    const tabs = held.tabs.filter((each) => each !== tab)
    const active = held.active === tab ? (tabs[at] ?? tabs[at - 1] ?? null) : held.active
    return { ...held, tabs, active }
  })
}

/** The canonical tree, with the focus on a group that survived it. */
function settle(workspace: Workspace, preferred: NodeId): Workspace {
  const settled = normalize(workspace)
  const groups = groupsOf(settled.root)
  const kept =
    groups.find((each) => each.id === preferred) ??
    groups.find((each) => each.id === workspace.focus) ??
    groups[0]

  return kept ? { ...settled, focus: kept.id } : settled
}

function mapGroup(
  root: WorkspaceNode,
  id: NodeId,
  change: (group: Group) => Group,
): WorkspaceNode {
  const path = pathTo(root, id)
  if (!path) return root

  const node = nodeAt(root, path)
  if (!node || !isGroup(node)) return root

  return replaceAt(root, path, change(node))
}
