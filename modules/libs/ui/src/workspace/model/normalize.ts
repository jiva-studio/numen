/**
 * The one tree an arrangement of the screen has.
 *
 * A pane holding nothing goes, and so does a branch holding one child.
 */
import { branch, orthogonal, type Workspace, type WorkspaceNode } from './node'
import { fit } from './shares'
import { isBranch, isPane, panesOf, withChildren } from './tree'

/** What a node comes to, and the shares those pieces take of its length. */
interface Slice {
  readonly nodes: readonly WorkspaceNode[]
  readonly sizes: readonly number[]
}

/**
 * A branch left with one child hands that child's own children up to its
 * parent: two levels turn a half, so they already divide their length the way
 * the parent does. A single child that is a pane goes up as it is.
 */
function reduce(node: WorkspaceNode): Slice {
  if (isPane(node)) {
    return node.tabs.length > 0 ? { nodes: [node], sizes: [1] } : { nodes: [], sizes: [] }
  }

  const nodes: WorkspaceNode[] = []
  const sizes: number[] = []
  const own = fit(node.sizes, node.children.length)

  node.children.forEach((child, index) => {
    const share = own[index] ?? 0
    const slice = reduce(child)
    slice.nodes.forEach((kept, at) => {
      nodes.push(kept)
      sizes.push((slice.sizes[at] ?? 0) * share)
    })
  })

  if (nodes.length === 0) return { nodes: [], sizes: [] }

  if (nodes.length === 1) {
    const only = nodes[0]
    if (!only) return { nodes: [], sizes: [] }
    return isBranch(only)
      ? { nodes: only.children, sizes: fit(only.sizes, only.children.length) }
      : { nodes: [only], sizes: [1] }
  }

  return { nodes: [withChildren(node, nodes, sizes)], sizes: [1] }
}

/** A pane with nothing in it, kept so that a workspace always has a root. */
const emptied = (root: WorkspaceNode): WorkspaceNode => {
  const first = panesOf(root)[0]
  return { kind: 'pane', id: first?.id ?? root.id, tabs: [], active: null }
}

export function normalize(workspace: Workspace): Workspace {
  const slice = reduce(workspace.root)

  const only = slice.nodes[0]
  if (!only) return { ...workspace, root: emptied(workspace.root) }
  if (slice.nodes.length === 1) return { ...workspace, root: only }

  // The root itself kept one child and handed its children up. There is no
  // parent to take them, so they become the root and the axis turns with them.
  return {
    ...workspace,
    root: branch(workspace.root.id, slice.nodes, fit(slice.sizes, slice.nodes.length)),
    axis: orthogonal(workspace.axis),
  }
}
