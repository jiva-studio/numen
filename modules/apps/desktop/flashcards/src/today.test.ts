/**
 * The day the window weighs a goal against is the review day, which begins at
 * the hour the settings name. Between midnight and that hour the calendar has
 * moved on and the review day has not, and a preset due today would be called
 * over while the core still schedules it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Goal, Stopped } from '@numen/protocol'

import { counting } from './counting'
import type { Counts } from './counting'
import { named, scheduling } from './scheduling'
import type { Asks, SettingsMessage } from './scheduling'
import type { Owing } from './core'

const dated = (day: string): SettingsMessage => ({
  goal: Goal.BY_DATE,
  byDate: day,
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  load: {},
  evenLoad: true,
})

const vault: Owing = {
  vaultId: '01A',
  name: 'Vault',
  path: '/vaults/01A',
  counted: true,
  faces: 4,
  due: 3,
  new: 1,
  decks: [{ deck: 'decks/Words.md', faces: 4, due: 3, new: 1, learned: 2, unbegun: 1 }],
  presets: [],
  unread: '',
  reading: false,
}

const answering = (settings: SettingsMessage): Asks => ({
  async scheduling() {
    return {
      preset: {
        path: 'Sanskrit.md',
        title: 'Sanskrit',
        settings,
        problems: [],
        stopsOn: Stopped.NOTHING,
      },
    }
  },
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the day a goal is weighed against', () => {
  it('is the one the application counted', async () => {
    const cards: Counts = {
      async *owing() {
        yield { day: '2026-09-04', vaults: [] }
      },
    }
    const held = counting({ cards, failed: () => {} })

    await held.count()

    expect(held.day.value).toBe('2026-09-04')
  })

  it('leaves a preset due today running in the hours past midnight', async () => {
    // One in the morning: the calendar says the fifth, and the review day that
    // began at four on the fourth is still running.
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 5, 1, 0, 0))
    expect(named(new Date())).toBe('2026-09-05')

    const cards: Counts = {
      async *owing() {
        yield { day: '2026-09-04', vaults: [] }
      },
    }
    const held = counting({ cards, failed: () => {} })
    await held.count()

    const one = scheduling({ presets: answering(dated('2026-09-04')) })
    await one.read(vault, held.day.value)

    expect(one.presets.value[0]?.paused).toBe('')
    expect(one.presets.value[0]?.cards).toBe(4)
  })
})
