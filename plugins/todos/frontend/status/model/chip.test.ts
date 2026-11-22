import { describe, expect, it } from 'vitest'
import type { Summary } from '../../types'
import { ALL_SHOWN, getChip } from './chip'

const summary = (overrides: Partial<Summary>): Summary => ({
  isReady: true,
  overdueCount: 0,
  todayCount: 0,
  nextSevenDaysCount: 0,
  noDateCount: 0,
  unreadableCount: 0,
  ...overrides,
})

describe('the chip', () => {
  it('counts the tasks of every shown group', () => {
    const chip = getChip(
      summary({ overdueCount: 2, todayCount: 3, nextSevenDaysCount: 4, noDateCount: 7 }),
    )

    expect(chip).toEqual({ isVisible: true, count: 16 })
  })

  it('leaves out the groups that are switched off', () => {
    const chip = getChip(
      summary({ overdueCount: 2, todayCount: 3, nextSevenDaysCount: 4, noDateCount: 7 }),
      { overdue: true, today: false, nextSevenDays: true, noDate: false },
    )

    expect(chip).toEqual({ isVisible: true, count: 6 })
  })

  it('counts the tasks due today when no task is overdue', () => {
    expect(getChip(summary({ todayCount: 1 }))).toEqual({ isVisible: true, count: 1 })
  })

  it('is hidden when only a switched off group has tasks', () => {
    const chip = getChip(summary({ noDateCount: 4 }), { ...ALL_SHOWN, noDate: false })

    expect(chip.isVisible).toBe(false)
  })

  it('is hidden when no open task exists', () => {
    expect(getChip(summary({})).isVisible).toBe(false)
  })

  it('is hidden before the counts are read and while the first scan has not ended', () => {
    expect(getChip(null).isVisible).toBe(false)
    expect(getChip(summary({ isReady: false, overdueCount: 3 })).isVisible).toBe(false)
  })
})
