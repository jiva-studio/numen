/** What a workspace is made of, as plain values. */

/** A tab's identity. What it shows is the application's to decide. */
export type TabId = string

export type NodeId = string

export type Orientation = 'horizontal' | 'vertical'

/** Where a dragged tab lands on a pane. */
export type Side = 'left' | 'right' | 'top' | 'bottom' | 'center'

/** What a tab is called, and what it carries besides, for a strip to show. */
export interface Tab {
  readonly id: TabId
  readonly title: string
  /**
   * A word for what the tab is holding: unsaved work, or why it is stuck. The
   * strip draws it beside the title and says it aloud.
   */
  readonly mark?: string
}

/** A stack of tabs with one of them showing. */
export interface Pane {
  readonly kind: 'pane'
  readonly id: NodeId
  readonly tabs: readonly TabId[]
  /** Null while the pane holds nothing. */
  readonly active: TabId | null
}

/**
 * A row or a column of nodes.
 *
 * Which one it is comes from its depth, so `sizes` are shares of the length it
 * divides: one per child, summing to one.
 */
export interface Branch {
  readonly kind: 'branch'
  readonly id: NodeId
  readonly children: readonly WorkspaceNode[]
  readonly sizes: readonly number[]
}

export type WorkspaceNode = Branch | Pane

/**
 * A tree of splits with a stack of tabs at every leaf.
 *
 * `axis` is how the root divides its length, and every level below turns a
 * quarter from the one above: a row holds columns, and those columns hold
 * rows. One arrangement of the screen therefore has one tree, and `normalize`
 * restores that form after every edit.
 */
export interface Workspace {
  readonly root: WorkspaceNode
  readonly axis: Orientation
  /** The pane a tab opens into. */
  readonly focus: NodeId
}

export const orthogonal = (orientation: Orientation): Orientation =>
  orientation === 'horizontal' ? 'vertical' : 'horizontal'

/** How a branch at this depth divides its length. */
export const orientationAt = (axis: Orientation, depth: number): Orientation =>
  depth % 2 === 0 ? axis : orthogonal(axis)

/** The orientation a split towards a side needs. Center splits nothing. */
export const orientationOf = (side: Side): Orientation | null => {
  if (side === 'center') return null
  return side === 'left' || side === 'right' ? 'horizontal' : 'vertical'
}

/** Whether a side puts what lands on it ahead of what it landed on. */
export const leads = (side: Side): boolean => side === 'left' || side === 'top'

export const pane = (id: NodeId, tabs: readonly TabId[], active?: TabId): Pane => ({
  kind: 'pane',
  id,
  tabs,
  active: active ?? tabs[0] ?? null,
})

export const branch = (
  id: NodeId,
  children: readonly WorkspaceNode[],
  sizes: readonly number[],
): Branch => ({ kind: 'branch', id, children, sizes })
