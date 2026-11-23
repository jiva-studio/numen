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
import { getEvenShares } from '../lib/getEvenShares'
import type { NodeIdFactory } from '../lib/edit'

export const createStack = (id: NodeId, ...tabs: string[]): WorkspaceNode => pane(id, tabs)

const splitBranch = (
  id: NodeId,
  children: readonly WorkspaceNode[],
  sizes?: readonly number[],
): WorkspaceNode => branch(id, children, sizes ?? getEvenShares(children.length))

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
export const createSideBySide = (): Workspace =>
  createWorkspace(
    splitBranch('root', [createStack('main', 'plex'), createStack('aside', 'chat')], [0.72, 0.28]),
  )

/** One stack holding both tabs, of which only one is visible at a time. */
export const createOneStack = (): Workspace => createWorkspace(createStack('main', 'plex', 'chat'))

/** Two stacks, the second with an extra tab, so neither becomes empty. */
export const withSpare = (): Workspace =>
  createWorkspace(
    splitBranch(
      'root',
      [createStack('main', 'plex'), createStack('aside', 'chat', 'notes')],
      [0.7, 0.3],
    ),
  )

/** Four levels, each turning a quarter from the one above. */
export const createDeepWorkspace = (): Workspace =>
  createWorkspace(
    splitBranch('root', [
      createStack('a', 'one'),
      splitBranch('down', [
        createStack('b', 'two'),
        splitBranch('across', [
          createStack('c', 'three'),
          splitBranch('again', [createStack('d', 'four'), createStack('e', 'five')]),
        ]),
      ]),
    ]),
  )

/** A stack with more tabs than a narrow strip can show. */
export const createManyTabsWorkspace = (): Workspace =>
  createWorkspace(
    splitBranch(
      'root',
      [
        createStack('main', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight'),
        createStack('aside', 'chat'),
      ],
      [0.7, 0.3],
    ),
  )

/** Nothing open. */
export const createEmptyWorkspace = (): Workspace => createWorkspace(pane('main', []))

export { splitBranch as split }
