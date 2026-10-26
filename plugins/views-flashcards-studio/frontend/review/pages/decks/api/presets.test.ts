import { describe, expect, it } from 'vitest'
import { ErrorCode } from '../../../../gen/numen/flashcards/v1/errors_pb.js'
import { Goal, StopReason } from '../../../../gen/numen/flashcards/v1/presets_pb.js'
import { DEFAULT_SETTINGS, formatSettings } from '../../../../cards/wire/settings'
import type { SettingsMessage } from '../../../../cards/wire/settings'

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

    expect(one.ok && one.value.activePreset.settings.goal).toBe('retention')
  })

  it('is steered by its minutes where the settings name no goal', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset({ settings: settings(Goal.UNSPECIFIED) }) }),
      '01A',
      'decks/Words.md',
    )

    expect(one.ok && one.value.activePreset.settings.goal).toBe('minutes')
  })

  it('fails with the error given where the preset came without settings', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset(), error: ErrorCode.NOT_A_NOTE }),
      '01A',
      'decks/Words.md',
    )

    expect(one).toMatchObject({ ok: false, error: 'not a note' })
  })

  it('fails as unread where no preset and no error came', async () => {
    const one = await readDeckPreset(answering({}), '01A', 'decks/Words.md')

    expect(one).toMatchObject({ ok: false, error: PRESET_UNREADABLE_MESSAGE })
  })
})
