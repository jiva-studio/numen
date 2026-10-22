import { describe, expect, it } from 'vitest'
import { ErrorCode, Goal, StopReason } from '@numen/protocol'
import { DEFAULT_SETTINGS, formatSettings } from '@numen/wire'
import type { SettingsMessage } from '@numen/wire'

import { readDeckPreset, PRESET_UNREADABLE_MESSAGE } from './presets'
import type { PresetsClient } from './presets'

const settings = (goal: Goal): SettingsMessage => ({
  ...formatSettings(DEFAULT_SETTINGS),
  goal,
  reviewsADay: 45,
})

/** An application answering every deck with what is given here. */
const answering = (answer: Awaited<ReturnType<PresetsClient['getVaultDeckPreset']>>) => ({
  getVaultDeckPreset: async () => answer,
})

const preset = (more: { settings?: SettingsMessage } = {}) => ({
  path: 'Sanskrit.md',
  title: 'Sanskrit',
  problems: [],
  stopsOn: StopReason.NOTHING,
  ...more,
})

describe('the preset one deck is scheduled by', () => {
  it('carries the goal its settings name', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset({ settings: settings(Goal.RETENTION) }) }),
      '01A',
      'decks/Words.md',
    )

    expect(one.ok && one.value.held.settings.goal).toBe('retention')
  })

  it('is steered by its minutes where the settings name no goal', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset({ settings: settings(Goal.UNSPECIFIED) }) }),
      '01A',
      'decks/Words.md',
    )

    expect(one.ok && one.value.held.settings.goal).toBe('minutes')
  })

  it('fails with the error given where the preset came without settings', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset(), error: ErrorCode.NOT_A_NOTE }),
      '01A',
      'decks/Words.md',
    )

    expect(one).toMatchObject({ ok: false, error: 'that file is not a note' })
  })

  it('fails as unread where no preset and no error came', async () => {
    const one = await readDeckPreset(answering({}), '01A', 'decks/Words.md')

    expect(one).toMatchObject({ ok: false, error: PRESET_UNREADABLE_MESSAGE })
  })
})
