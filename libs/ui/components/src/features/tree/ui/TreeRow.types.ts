import type { VNode } from 'vue'
import type { RowId, ShownRow } from '../lib/row'

/** Props for a single tree row component. */
export interface TreeRowProps {
  row: ShownRow
  name: string
  isSelected: boolean
  isDragged: boolean
  isDropInside: boolean
  isDropAbove: boolean
  isTabStop: boolean
  isRenaming: boolean
  mark: Record<string, string>
  /** What the end of the row draws when the row has no `end` slot. */
  end?: ((props: { id: RowId }) => VNode[]) | undefined
}
