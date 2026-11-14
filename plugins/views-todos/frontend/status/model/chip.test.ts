import { describe, expect, it } from 'vitest'
import type { Summary } from '../../types'
import { getChip } from './chip'

const summary = (overrides: Partial<Summary>): Summary => ({
  isReady: true,
  overdueCount: 0,
  todayCount: 0,
  noDateCount: 0,
  unreadableCount: 0,
  ...overrides,
})

describe('the chip', () => {
  it('counts the overdue tasks plus the tasks due today', () => {
    const chip = getChip(summary({ overdueCount: 2, todayCount: 3, noDateCount: 7 }))

    expect(chip).toEqual({ isVisible: true, count: 5, isOverdue: true })
  })

  it('is not overdue when no task is overdue', () => {
    expect(getChip(summary({ todayCount: 1 }))).toEqual({
      isVisible: true,
      count: 1,
      isOverdue: false,
    })
  })

  it('is visible with a count of zero while only tasks with no date exist', () => {
    expect(getChip(summary({ noDateCount: 4 }))).toEqual({
      isVisible: true,
      count: 0,
      isOverdue: false,
    })
  })

  it('is hidden when no open task exists', () => {
    expect(getChip(summary({})).isVisible).toBe(false)
  })

  it('is hidden before the counts are read and while the first scan has not ended', () => {
    expect(getChip(null).isVisible).toBe(false)
    expect(getChip(summary({ isReady: false, overdueCount: 3 })).isVisible).toBe(false)
  })
})
