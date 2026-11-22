import type { Summary } from '../../types'

/** What the chip in the title bar shows. The count is the tasks of the groups that are shown. */
export interface Chip {
  readonly isVisible: boolean
  readonly count: number
}

/** Which groups the popover lists and the chip counts. */
export interface ShownGroups {
  readonly overdue: boolean
  readonly today: boolean
  readonly nextSevenDays: boolean
  readonly noDate: boolean
}

const ALL_SHOWN: ShownGroups = {
  overdue: true,
  today: true,
  nextSevenDays: true,
  noDate: true,
}

const HIDDEN: Chip = { isVisible: false, count: 0 }

/** Gives the chip for the counts, or the hidden chip before the counts are ready. */
export function getChip(summary: Summary | null, shown: ShownGroups = ALL_SHOWN): Chip {
  if (!summary?.isReady) return HIDDEN
  const count =
    (shown.overdue ? summary.overdueCount : 0) +
    (shown.today ? summary.todayCount : 0) +
    (shown.nextSevenDays ? summary.nextSevenDaysCount : 0) +
    (shown.noDate ? summary.noDateCount : 0)
  return count === 0 ? HIDDEN : { isVisible: true, count }
}
