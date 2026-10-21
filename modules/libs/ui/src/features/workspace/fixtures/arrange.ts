/**
 * Where every node of a workspace falls inside a box.
 *
 * Pure — the same tree and the same box give the same numbers on any machine.
 * Nothing here measures the DOM.
 */
import { getOrientation, type NodeId, type Workspace, type WorkspaceNode } from '../lib/node'
import type { Rect } from '../lib/rect'
import { isBranch } from '../lib/tree'
import { fit } from '../lib/shares'

export interface ArrangeOptions {
  /** What is left between two children for the handle between them. */
  readonly gap: number
}

export const DEFAULT_ARRANGE: ArrangeOptions = { gap: 0 }

export function arrangeWorkspace(
  workspace: Workspace,
  box: Rect,
  options: Partial<ArrangeOptions> = {},
): ReadonlyMap<NodeId, Rect> {
  const { gap } = { ...DEFAULT_ARRANGE, ...options }
  const boxes = new Map<NodeId, Rect>()
  place(workspace.root, box, workspace, 0, gap, boxes)
  return boxes
}

function place(
  node: WorkspaceNode,
  box: Rect,
  workspace: Workspace,
  depth: number,
  gap: number,
  into: Map<NodeId, Rect>,
): void {
  into.set(node.id, box)
  if (!isBranch(node)) return

  const isAcross = getOrientation(workspace.axis, depth) === 'horizontal'
  const length = isAcross ? box.width : box.height
  const free = Math.max(length - gap * Math.max(node.children.length - 1, 0), 0)
  const sizes = fit(node.sizes, node.children.length)

  let alongCoord = isAcross ? box.x : box.y
  node.children.forEach((child, index) => {
    const acquired = free * (sizes[index] ?? 0)
    const childBox: Rect = isAcross
      ? { x: alongCoord, y: box.y, width: acquired, height: box.height }
      : { x: box.x, y: alongCoord, width: box.width, height: acquired }
    // Stryker disable next-line ArithmeticOperator: depth - 1 has the parity of depth + 1, and only parity is read.
    place(child, childBox, workspace, depth + 1, gap, into)
    alongCoord += acquired + gap
  })
}
