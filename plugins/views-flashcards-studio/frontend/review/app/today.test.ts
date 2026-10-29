/**
 * The day the window weighs a goal against is the review day, which begins at
 * the hour the settings name. Between midnight and that hour the calendar has
 * moved on and the review day has not, and a preset due today would be called
 * over while the core still schedules it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StopReason } from '../../api/types.js'
import { DEFAULT_SETTINGS, formatSettings } from '../../cards/wire/settings'
import type { SettingsMessage } from '../../cards/wire/settings'

import { useReviewCounter } from '../entities/vault'
import type { CardsDueClient, VaultCardsDue } from '../entities/vault'
import { getDayName } from '../../lib/day'
import { useVaultPresets } from '../pages/decks'
import type { PresetsClient } from '../pages/decks'

const createSettings = (day: string) =>
  formatSettings({ ...DEFAULT_SETTINGS, goal: 'date', byDate: day, reviewsADay: 200 })

const vault: VaultCardsDue = {
  vault: '01A',
  name: 'Vault',
  path: '/vaults/01A',
  isCounted: true,
  reviewCards: 4,
  due: 3,
  new: 1,
  decks: [{ deck: 'decks/Words.md', reviewCards: 4, due: 3, new: 1, learned: 2, unbegun: 1 }],
  presets: [],
  unread: '',
  isIndexing: false,
}

const createPresets = (settings: SettingsMessage): PresetsClient => ({
  async getVaultDeckPreset() {
    return {
      preset: {
        path: 'Sanskrit.md',
        title: 'Sanskrit',
        settings,
        problems: [],
        stopsOn: StopReason.NOTHING,
      },
    }
  },
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the day a goal is weighed against', () => {
  it('is the one the application counted', async () => {
    const cards: CardsDueClient = {
      async *watchCardsDue() {
        yield { day: '2026-09-04', vaults: [] }
      },
    }
    const card = useReviewCounter({ cards, reportError: () => {} })

    await card.count()

    expect(card.day.value).toBe('2026-09-04')
  })

  it('leaves a preset due today running in the hours past midnight', async () => {
    // One in the morning: the calendar shows the fifth, and the review day that
    // began at four on the fourth is still running.
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 5, 1, 0, 0))
    expect(getDayName(new Date())).toBe('2026-09-05')

    const cards: CardsDueClient = {
      async *watchCardsDue() {
        yield { day: '2026-09-04', vaults: [] }
      },
    }
    const card = useReviewCounter({ cards, reportError: () => {} })
    await card.count()

    const one = useVaultPresets({ presets: createPresets(createSettings('2026-09-04')) })
    await one.read(vault, card.day.value)

    expect(one.presets.value[0]?.paused).toBe('')
    expect(one.presets.value[0]?.cards).toBe(4)
  })
})
