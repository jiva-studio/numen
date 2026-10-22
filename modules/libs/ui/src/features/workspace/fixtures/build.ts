/** Workspaces to draw and to test against. */
import {
  branch,
  pane,
  type NodeId,
  type Orientation,
  type Workspace,
  type WorkspaceNode,
} from '../lib/node'
import { getPanes } from '../lib/tree'
import { even } from '../lib/shares'
import type { NodeIdFactory } from '../lib/edit'

/** A stack of tabs. */
export const stack = (id: NodeId, ...tabs: string[]): WorkspaceNode => pane(id, tabs)

/** A row or a column, in equal shares when none are given. */
const splitBranch = (
  id: NodeId,
  children: readonly WorkspaceNode[],
  sizes?: readonly number[],
): WorkspaceNode => branch(id, children, sizes ?? even(children.length))

export const createWorkspace = (
  root: WorkspaceNode,
  axis: Orientation = 'horizontal',
  focus?: NodeId,
): Workspace => ({
  root,
  axis,
  focus: focus ?? getPanes(root)[0]?.id ?? root.id,
})

/** Identities that count up, so a test can name what a gesture made. */
export function createIdFactory(prefix = 'made'): NodeIdFactory {
  let createdItem = 0
  return () => `${prefix}-${++createdItem}`
}

/** What the desktop opens with: the plex, and the agent beside it. */
export const sideBySide = (): Workspace =>
  createWorkspace(
    splitBranch('root', [stack('main', 'plex'), stack('aside', 'chat')], [0.72, 0.28]),
  )

/** One stack holding both, so that only one shows at a time. */
export const oneStack = (): Workspace => createWorkspace(stack('main', 'plex', 'chat'))

/** Two stacks, the second with a tab to spare, so that neither empties. */
export const withSpare = (): Workspace =>
  createWorkspace(
    splitBranch('root', [stack('main', 'plex'), stack('aside', 'chat', 'notes')], [0.7, 0.3]),
  )

/** Four levels, each turning a quarter from the one above. */
export const deep = (): Workspace =>
  createWorkspace(
    splitBranch('root', [
      stack('a', 'one'),
      splitBranch('down', [
        stack('b', 'two'),
        splitBranch('across', [
          stack('c', 'three'),
          splitBranch('again', [stack('d', 'four'), stack('e', 'five')]),
        ]),
      ]),
    ]),
  )

/** A stack with more tabs than a narrow strip can show. */
export const manyTabs = (): Workspace =>
  createWorkspace(
    splitBranch(
      'root',
      [
        stack('main', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight'),
        stack('aside', 'chat'),
      ],
      [0.7, 0.3],
    ),
  )

/** Nothing open. */
export const empty = (): Workspace => createWorkspace(pane('main', []))

export { splitBranch as split }
