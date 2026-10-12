import { describe, expect, it } from 'vitest'
import { ErrorCode, Goal, StopReason } from '@numen/protocol'

import { readDeckPreset, PRESET_UNREADABLE_MESSAGE } from './presets'
import type { PresetsClient } from './presets'
import type { SettingsMessage } from '../types'

const settings = (goal: Goal): SettingsMessage => ({
  goal,
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  load: {},
  hasEvenLoad: true,
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

  it('is refused with the error given where the preset came without settings', async () => {
    const one = await readDeckPreset(
      answering({ preset: preset(), error: ErrorCode.NOT_A_NOTE }),
      '01A',
      'decks/Words.md',
    )

    expect(one).toMatchObject({ ok: false, error: 'that file is not a note' })
  })

  it('is refused as unread where no preset and no error came', async () => {
    const one = await readDeckPreset(answering({}), '01A', 'decks/Words.md')

    expect(one).toMatchObject({ ok: false, error: PRESET_UNREADABLE_MESSAGE })
  })
})
