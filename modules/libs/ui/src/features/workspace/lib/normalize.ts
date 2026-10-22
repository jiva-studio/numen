/**
 * The one tree an arrangement of the screen has.
 *
 * A pane holding nothing goes, and so does a branch holding one child.
 */
import { branch, orthogonal, type Workspace, type WorkspaceNode } from './node'
import { fit } from './shares'
import { isBranch, isPane, getPanes, withChildren } from './tree'

/** One node a node evaluates to, and the share it takes of that node's length. */
interface Piece {
  readonly node: WorkspaceNode
  readonly share: number
}

/**
 * A branch left with one child hands that child's own children up to its
 * parent: two levels turn a half, so they already divide their length the way
 * the parent does. A single child that is a pane goes up as it is.
 */
function reduce(node: WorkspaceNode): readonly Piece[] {
  if (isPane(node)) return node.tabs.length > 0 ? [{ node, share: 1 }] : []

  const own = fit(node.sizes, node.children.length)
  const pieces = node.children.flatMap((child, index) =>
    reduce(child).map((piece) => ({ node: piece.node, share: piece.share * (own[index] ?? 0) })),
  )

  const [first, ...others] = pieces
  if (!first) return []

  if (others.length === 0) {
    if (!isBranch(first.node)) return [{ node: first.node, share: 1 }]
    const shares = fit(first.node.sizes, first.node.children.length)
    return first.node.children.map((child, index) => ({ node: child, share: shares[index] ?? 0 }))
  }

  return [
    {
      node: withChildren(
        node,
        pieces.map((piece) => piece.node),
        pieces.map((piece) => piece.share),
      ),
      share: 1,
    },
  ]
}

/** A pane with nothing in it, kept so that a workspace always has a root. */
const emptyPane = (root: WorkspaceNode): WorkspaceNode => {
  const first = getPanes(root)[0]
  return { kind: 'pane', id: first?.id ?? root.id, tabs: [], active: null }
}

export function normalize(workspace: Workspace): Workspace {
  const pieces = reduce(workspace.root)

  const [only, ...others] = pieces
  if (!only) return { ...workspace, root: emptyPane(workspace.root) }
  if (others.length === 0) return { ...workspace, root: only.node }

  // The root itself kept one child and handed its children up. There is no
  // parent to take them, so they become the root and the axis turns with them.
  return {
    ...workspace,
    root: branch(
      workspace.root.id,
      pieces.map((piece) => piece.node),
      fit(
        pieces.map((piece) => piece.share),
        pieces.length,
      ),
    ),
    axis: orthogonal(workspace.axis),
  }
}
