import { describe, expect, it } from 'vitest'
import { BudgetUnit as BudgetUnits, Goal as Goals, Rule as Rules } from '@numen/protocol'

import { DEFAULT_SETTINGS, formatSettings, parseSettings } from './settings'
import type { Settings, SettingsMessage } from './settings'

const message = (more: Partial<SettingsMessage> = {}): SettingsMessage => ({
  goal: Goals.RETENTION,
  byDate: '2026-09-30',
  minutesADay: 30,
  newADay: 5,
  reviewsADay: 80,
  retention: 0.85,
  counts: BudgetUnits.SHOWS,
  backlog: 40,
  load: { mon: 25 },
  hasEvenLoad: false,
  learned: Rules.RETENTION,
  interval: 14,
  ...more,
})

describe('the settings of a preset', () => {
  it('are read field by field from the message', () => {
    expect(parseSettings(message())).toStrictEqual({
      goal: 'retention',
      byDate: '2026-09-30',
      minutesADay: 30,
      newADay: 5,
      reviewsADay: 80,
      retention: 0.85,
      counts: 'shows',
      backlog: 40,
      load: { mon: 25 },
      hasEvenLoad: false,
      learned: 'retention',
      interval: 14,
    })
  })

  it('are the defaults where the preset carries none', () => {
    expect(parseSettings(undefined)).toBe(DEFAULT_SETTINGS)
  })

  it('take the default for a value the message is empty for', () => {
    const read = parseSettings(
      message({
        goal: Goals.UNSPECIFIED,
        counts: BudgetUnits.UNSPECIFIED,
        learned: Rules.UNSPECIFIED,
      }),
    )
    expect(read.goal).toBe(DEFAULT_SETTINGS.goal)
    expect(read.counts).toBe(DEFAULT_SETTINGS.counts)
    expect(read.learned).toBe(DEFAULT_SETTINGS.learned)
  })

  it('carry whether the load is even under the name the schema gives it', () => {
    expect(parseSettings(message({ hasEvenLoad: true })).hasEvenLoad).toBe(true)
    expect(formatSettings({ ...DEFAULT_SETTINGS, hasEvenLoad: false }).hasEvenLoad).toBe(false)
  })

  it('come back unchanged from the message they are written to', () => {
    const settings: Settings = {
      ...DEFAULT_SETTINGS,
      goal: 'date',
      counts: 'shows',
      learned: 'retention',
    }
    expect(parseSettings(formatSettings(settings))).toStrictEqual(settings)
  })
})
