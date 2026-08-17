/**
 * The one tree an arrangement of the screen has.
 *
 * An edit may leave a group holding nothing, or a branch holding one child.
 * Neither draws anything, and both let two trees stand for the same picture.
 * This puts the tree back into the single form that picture has.
 */
import { branch, orthogonal, type Workspace, type WorkspaceNode } from './node'
import { fit } from './shares'
import { groupsOf, isBranch, isGroup, withChildren } from './tree'

/** What a node comes to, and the shares those pieces take of its length. */
interface Slice {
  readonly nodes: readonly WorkspaceNode[]
  readonly sizes: readonly number[]
}

/**
 * A branch that keeps one child hands the child's own children up to its
 * parent. Two levels turn a half, so those children already divide their
 * length the way the parent does, and the picture is unchanged. A single
 * child that is a group goes up as it is, having no orientation to keep.
 */
function reduce(node: WorkspaceNode): Slice {
  if (isGroup(node)) {
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

/** A group with nothing in it, kept so that a workspace always has a root. */
const emptied = (root: WorkspaceNode): WorkspaceNode => {
  const first = groupsOf(root)[0]
  return { kind: 'group', id: first?.id ?? root.id, tabs: [], active: null }
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
