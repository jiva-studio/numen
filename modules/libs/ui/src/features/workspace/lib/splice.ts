/** Changes to a workspace tree: a pane put alongside a node, a tab taken off, a pane replaced, and the canonical form restored. */
import {
  branch,
  isLeading,
  orientationAt,
  orientationOf,
  pane,
  type NodeId,
  type Orientation,
  type Pane,
  type Side,
  type TabId,
  type Workspace,
  type WorkspaceNode,
} from './node'
import { normalize } from './normalize'
import {
  isBranch,
  isPane,
  nodeAt,
  paneWithTab,
  panesOf,
  pathTo,
  replaceAt,
  withChildren,
} from './tree'
import { insert } from './shares'

/** Where an identity for a pane or a branch a gesture makes comes from. */
export type NodeIdFactory = () => NodeId

export interface Landing {
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
export function putBeside(
  root: WorkspaceNode,
  onto: NodeId,
  tab: TabId,
  side: Side,
  axis: Orientation,
  id: NodeIdFactory,
): Landing {
  const wanted = orientationOf(side)
  const path = pathTo(root, onto)
  if (!wanted || !path) return { root, axis, focus: onto }

  const made = pane(id(), [tab])
  const index = path[path.length - 1]
  const above = path.slice(0, -1)

  if (index === undefined) {
    const pair = isLeading(side) ? [made, root] : [root, made]
    return { root: branch(id(), pair, [0.5, 0.5]), axis: wanted, focus: made.id }
  }

  const parent = nodeAt(root, above)
  if (!parent || !isBranch(parent)) return { root, axis, focus: onto }

  if (orientationAt(axis, above.length) === wanted) {
    const at = isLeading(side) ? index : index + 1
    const children = [...parent.children.slice(0, at), made, ...parent.children.slice(at)]
    const sizes = insert(parent.sizes, index, isLeading(side))
    return {
      root: replaceAt(root, above, withChildren(parent, children, sizes)),
      axis,
      focus: made.id,
    }
  }

  const target = nodeAt(root, path)
  if (!target) return { root, axis, focus: onto }

  const pair = isLeading(side) ? [made, target] : [target, made]
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
export function detach(root: WorkspaceNode, tab: TabId): WorkspaceNode {
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
export function settle(workspace: Workspace, focus: NodeId): Workspace {
  const settled = normalize(workspace)
  const panes = panesOf(settled.root)
  const kept =
    panes.find((each) => each.id === focus) ??
    panes.find((each) => each.id === workspace.focus) ??
    panes[0]

  return kept ? { ...settled, focus: kept.id } : settled
}

export function mapPane(
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
