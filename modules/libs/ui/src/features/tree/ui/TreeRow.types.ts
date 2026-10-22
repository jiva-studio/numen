import type { ShownRow } from '../lib/row'

export interface TreeRowProps {
  row: ShownRow
  /** What the tree is announced as. */
  name: string
  /** The selection relies on it. */
  isSelected: boolean
  /** It is among the rows being dragged. */
  isLifted: boolean
  /** A drop would land inside it. */
  isDropInside: boolean
  /** A drop would land above it. */
  isDropAbove: boolean
  /** The one row the tab key reaches. */
  isTabStop: boolean
  /** Its name is in a field. */
  isRenaming: boolean
  /** What the row is marked with, written onto it currently. */
  mark: Record<string, string>
}
