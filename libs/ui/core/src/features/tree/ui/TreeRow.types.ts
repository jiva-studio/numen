import type { ShownRow } from '../lib/row'

export interface TreeRowProps {
  row: ShownRow
  name: string
  isSelected: boolean
  isLifted: boolean
  isDropInside: boolean
  isDropAbove: boolean
  isTabStop: boolean
  isRenaming: boolean
  mark: Record<string, string>
}
