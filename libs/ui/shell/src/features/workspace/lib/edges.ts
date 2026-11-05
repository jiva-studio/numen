/** The panes of a workspace that touch its top edge and its top corners. */
import {
  getOrientation,
  type NodeId,
  type Orientation,
  type Workspace,
  type WorkspaceNode,
} from './node'

/** The room the window buttons take in the top corners of the window, in pixels. */
export interface WindowEdge {
  readonly leftInset: number
  readonly rightInset: number
}

export interface TopPanes {
  /** The panes whose top is the top of the workspace, in the order they are drawn. */
  readonly top: readonly NodeId[]
  /** The pane in the top-left corner. */
  readonly left: NodeId
  /** The pane in the top-right corner. */
  readonly right: NodeId
}

/** A branch whose children stand side by side gives each of them the top; a column, its first. */
function collectTop(node: WorkspaceNode, axis: Orientation, depth: number, out: NodeId[]): void {
  if (node.kind === 'pane') {
    out.push(node.id)
    return
  }
  const children =
    getOrientation(axis, depth) === 'horizontal' ? node.children : node.children.slice(0, 1)
  for (const child of children) collectTop(child, axis, depth + 1, out)
}

function findCorner(
  node: WorkspaceNode,
  axis: Orientation,
  depth: number,
  side: 'left' | 'right',
): NodeId {
  if (node.kind === 'pane') return node.id
  const isRow = getOrientation(axis, depth) === 'horizontal'
  const child = side === 'right' && isRow ? node.children.at(-1) : node.children[0]
  return child ? findCorner(child, axis, depth + 1, side) : node.id
}

/** What the strip of a pane takes from the top of the window. */
export interface StripEdge {
  /** Whether the free space of the strip moves the window. */
  readonly isDraggable: boolean
  /** The room kept free at the left of the strip, in pixels. */
  readonly leftInset: number
  /** The room kept free at the right of the strip, in pixels. */
  readonly rightInset: number
}

/** The edge of a strip that does not touch the top of the window. */
export const NO_STRIP_EDGE: StripEdge = { isDraggable: false, leftInset: 0, rightInset: 0 }

/**
 * What the strip of the pane takes from the top of the window: nothing while the workspace is not
 * the top of the window or the pane is not at its top edge, and the room of the corner it is in.
 */
export function getStripEdge(
  panes: TopPanes,
  edge: WindowEdge | undefined,
  pane: NodeId,
): StripEdge {
  if (!edge || !panes.top.includes(pane)) return NO_STRIP_EDGE
  return {
    isDraggable: true,
    leftInset: pane === panes.left ? edge.leftInset : 0,
    rightInset: pane === panes.right ? edge.rightInset : 0,
  }
}

/** The panes at the top edge and in the top corners of the workspace. */
export function findTopPanes(workspace: Workspace): TopPanes {
  const top: NodeId[] = []
  collectTop(workspace.root, workspace.axis, 0, top)
  return {
    top,
    left: findCorner(workspace.root, workspace.axis, 0, 'left'),
    right: findCorner(workspace.root, workspace.axis, 0, 'right'),
  }
}
