import type { ShownRow } from '../lib/row'

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
}
