/** Workspaces to draw and to test against. */
import {
  branch,
  group,
  groupsOf,
  type NodeId,
  type Orientation,
  type Workspace,
  type WorkspaceNode,
} from '../model'
import { even } from '../model/shares'
import type { Naming } from '../edit'

/** A stack of tabs. */
export const stack = (id: NodeId, ...tabs: string[]): WorkspaceNode => group(id, tabs)

/** A row or a column, in equal shares when none are given. */
export const split = (
  id: NodeId,
  children: readonly WorkspaceNode[],
  sizes?: readonly number[],
): WorkspaceNode => branch(id, children, sizes ?? even(children.length))

export const workspaceOf = (
  root: WorkspaceNode,
  axis: Orientation = 'horizontal',
  focus?: NodeId,
): Workspace => ({
  root,
  axis,
  focus: focus ?? groupsOf(root)[0]?.id ?? root.id,
})

/** Identities that count up, so a test can name what a gesture made. */
export function naming(prefix = 'made'): Naming {
  let made = 0
  return () => `${prefix}-${++made}`
}

/** What the desktop opens with: the plex, and the agent beside it. */
export const sideBySide = (): Workspace =>
  workspaceOf(split('root', [stack('main', 'plex'), stack('aside', 'chat')], [0.72, 0.28]))

/** One stack holding both, so that only one shows at a time. */
export const oneStack = (): Workspace => workspaceOf(stack('main', 'plex', 'chat'))

/** Two stacks, the second with a tab to spare, so that neither empties. */
export const withSpare = (): Workspace =>
  workspaceOf(split('root', [stack('main', 'plex'), stack('aside', 'chat', 'notes')], [0.7, 0.3]))

/** Four levels, each turning a quarter from the one above. */
export const deep = (): Workspace =>
  workspaceOf(
    split('root', [
      stack('a', 'one'),
      split('down', [
        stack('b', 'two'),
        split('across', [stack('c', 'three'), split('again', [stack('d', 'four'), stack('e', 'five')])]),
      ]),
    ]),
  )

/** A stack with more tabs than a narrow strip can show. */
export const crowded = (): Workspace =>
  workspaceOf(
    split(
      'root',
      [
        stack('main', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight'),
        stack('aside', 'chat'),
      ],
      [0.7, 0.3],
    ),
  )

/** Nothing open. */
export const empty = (): Workspace => workspaceOf(group('main', []))
