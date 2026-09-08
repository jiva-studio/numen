/** Finding a node in a workspace, and putting one back. */
import { fit } from './shares'
import type { Branch, NodeId, Pane, TabId, WorkspaceNode } from './node'

/** Child indices from the root down to a node. The root's own path is empty. */
export type Path = readonly number[]

export const isBranch = (node: WorkspaceNode): node is Branch => node.kind === 'branch'

export const isPane = (node: WorkspaceNode): node is Pane => node.kind === 'pane'

/** Every pane in the tree, in the order they are drawn. */
export function panesOf(node: WorkspaceNode): readonly Pane[] {
  return isPane(node) ? [node] : node.children.flatMap(panesOf)
}

export function nodeAt(root: WorkspaceNode, path: Path): WorkspaceNode | null {
  let node: WorkspaceNode | undefined = root
  for (const index of path) {
    if (!node || !isBranch(node)) return null
    node = node.children[index]
  }
  return node ?? null
}

export function pathTo(root: WorkspaceNode, id: NodeId): Path | null {
  if (root.id === id) return []
  if (!isBranch(root)) return null

  for (let i = 0; i < root.children.length; i++) {
    const child = root.children[i]
    if (!child) continue
    const below = pathTo(child, id)
    if (below) return [i, ...below]
  }
  return null
}

export function paneById(root: WorkspaceNode, id: NodeId): Pane | null {
  return panesOf(root).find((pane) => pane.id === id) ?? null
}

export function paneWithTab(root: WorkspaceNode, tab: TabId): Pane | null {
  return panesOf(root).find((pane) => pane.tabs.includes(tab)) ?? null
}

/** A branch with different children, its shares made to sum to one. */
export const withChildren = (
  branch: Branch,
  children: readonly WorkspaceNode[],
  sizes: readonly number[],
): Branch => ({ ...branch, children, sizes: fit(sizes, children.length) })

/** The tree with the node at `path` replaced. */
export function replaceAt(
  root: WorkspaceNode,
  path: Path,
  replacement: WorkspaceNode,
): WorkspaceNode {
  const [index, ...below] = path
  if (index === undefined) return replacement
  if (!isBranch(root)) return root

  const child = root.children[index]
  if (!child) return root

  const children = [...root.children]
  children[index] = replaceAt(child, below, replacement)
  return { ...root, children }
}
