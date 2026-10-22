/** Changes to a workspace tree: a pane put alongside a node, a tab taken off, a pane replaced, and the canonical form restored. */
import {
  branch,
  isLeading,
  getOrientation,
  findSideOrientation,
  pane,
  type Branch,
  type NodeId,
  type Orientation,
  type Pane,
  type Side,
  type TabId,
  type Workspace,
  type WorkspaceNode,
} from './node'
import { normalize } from './normalize'
import { isPane, findNode, paneWithTab, getPanes, pathTo, replaceAt, withChildren } from './tree'
import { insert } from './shares'

/** Where an identity for a pane or a branch a gesture makes comes from. */
export type NodeIdFactory = () => NodeId

export interface SplitResult {
  readonly root: WorkspaceNode
  readonly axis: Orientation
  readonly focus: NodeId
}

export interface BesidePlacement {
  readonly onto: NodeId
  readonly tab: TabId
  readonly side: Side
  readonly axis: Orientation
}

interface JoinBeside {
  readonly index: number
  readonly side: Side
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
  placement: BesidePlacement,
  id: NodeIdFactory,
): SplitResult {
  const { onto, tab, side, axis } = placement
  const wanted = findSideOrientation(side)
  const path = pathTo(root, onto)
  if (!wanted || !path) return { root, axis, focus: onto }

  const createdItem = pane(id(), [tab])
  const index = path[path.length - 1]
  const above = path.slice(0, -1)

  if (index === undefined) {
    return {
      root: branch(id(), orderPair(createdItem, root, side), [0.5, 0.5]),
      axis: wanted,
      focus: createdItem.id,
    }
  }

  const parent = findNode(root, above) as Branch

  if (getOrientation(axis, above.length) === wanted) {
    return {
      root: replaceAt(root, above, joinBeside(parent, createdItem, { index, side })),
      axis,
      focus: createdItem.id,
    }
  }

  const target = findNode(root, path) as WorkspaceNode

  return {
    root: replaceAt(root, path, branch(id(), orderPair(createdItem, target, side), [0.5, 0.5])),
    axis,
    focus: createdItem.id,
  }
}

/** The new pane and a node as a branch's two children, in the order the side asks. */
function orderPair(newPane: Pane, node: WorkspaceNode, side: Side): WorkspaceNode[] {
  return isLeading(side) ? [newPane, node] : [node, newPane]
}

/** The branch with the new pane among its children, taking half the share of the child at `index`. */
function joinBeside(parent: Branch, newPane: Pane, join: JoinBeside): WorkspaceNode {
  const { index, side } = join
  const targetIndex = isLeading(side) ? index : index + 1
  const children = [
    ...parent.children.slice(0, targetIndex),
    newPane,
    ...parent.children.slice(targetIndex),
  ]
  return withChildren(parent, children, insert(parent.sizes, index, isLeading(side)))
}

/**
 * The tab taken off whichever pane holds it. The pane is left where it is,
 * empty if that was its last tab, and `commit` is what clears it away.
 */
export function detach(root: WorkspaceNode, tab: TabId): WorkspaceNode {
  const holder = paneWithTab(root, tab)
  if (!holder) return root

  return mapPane(root, holder.id, (target) => {
    const tabIndex = target.tabs.indexOf(tab)
    const tabs = target.tabs.filter((each) => each !== tab)
    const active =
      target.active === tab ? (tabs[tabIndex] ?? tabs[tabIndex - 1] ?? null) : target.active
    return { ...target, tabs, active }
  })
}

/** The canonical tree, with the focus on a pane that survived it. */
export function commit(workspace: Workspace, focus: NodeId): Workspace {
  const normalized = normalize(workspace)
  const panes = getPanes(normalized.root)
  const retainedBranch =
    panes.find((each) => each.id === focus) ??
    panes.find((each) => each.id === workspace.focus) ??
    panes[0]

  return retainedBranch ? { ...normalized, focus: retainedBranch.id } : normalized
}

export function mapPane(
  root: WorkspaceNode,
  id: NodeId,
  change: (pane: Pane) => Pane,
): WorkspaceNode {
  const path = pathTo(root, id)
  if (!path) return root

  const node = findNode(root, path)
  if (!node || !isPane(node)) return root

  return replaceAt(root, path, change(node))
}
