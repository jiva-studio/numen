import type { Summary } from '../../types'

/** What the chip in the title bar shows. The count is the overdue tasks plus the tasks due today. */
export interface Chip {
  readonly isVisible: boolean
  readonly count: number
  readonly isOverdue: boolean
}

const HIDDEN: Chip = { isVisible: false, count: 0, isOverdue: false }

/** Gives the chip for the counts, or the hidden chip before the counts are ready. */
export function getChip(summary: Summary | null): Chip {
  if (!summary?.isReady) return HIDDEN
  const { overdueCount, todayCount, noDateCount } = summary
  if (overdueCount + todayCount + noDateCount === 0) return HIDDEN
  return { isVisible: true, count: overdueCount + todayCount, isOverdue: overdueCount > 0 }
}
