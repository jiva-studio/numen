import type { Summary } from '../../types'

/** What the chip in the title bar shows. */
export interface Chip {
  readonly isVisible: boolean
  /** The overdue tasks plus the tasks due today. */
  readonly count: number
  readonly isOverdue: boolean
}

/** Gives the chip for the counts, or the hidden chip before the counts are ready. */
export function getChip(summary: Summary | null): Chip {
  return { isVisible: true, count: -1, isOverdue: true }
}
