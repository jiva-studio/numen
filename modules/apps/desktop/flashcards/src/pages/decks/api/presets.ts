/** What the application answers about the preset one deck is scheduled by. */
import { StopReason } from '@numen/protocol'
import type { ErrorCode } from '@numen/protocol'
import { asFailure, asValue, goalOf, formatErrorCodeMessage } from '@numen/wire'
import type { Result } from '@numen/wire'

import type { Settings, SettingsMessage } from '../types'

/**
 * What the application answers about the presets of a vault. Every question
 * names the vault it is about, the same way an answer names the one it is
 * written to.
 */
export interface PresetsClient {
  getVaultDeckPreset(said: { vault: string; deck: string }): Promise<{
    preset?:
      | {
          path: string
          title: string
          settings?: SettingsMessage | undefined
          problems: readonly string[]
          /** Why it schedules nothing on the day it was read in. */
          stopsOn: StopReason
        }
      | undefined
    error?: ErrorCode | undefined
  }>
}

/** What was answered about one deck's preset. */
export interface DeckPreset {
  readonly deck: string
  readonly held: {
    path: string
    name: string
    settings: Settings
    problems: readonly string[]
    stopsOn: StopReason
  }
}

/** The preset a deck is scheduled by, or why it was not read, in words to show. */
export type DeckPresetResult = Result<DeckPreset, string>

/** What is shown of a preset the window has no other reason to give for. */
export const UNREAD = 'the settings of this preset could not be read'

/** The preset one deck is scheduled by, or why it could not be read. */
export const readDeckPreset = async (
  presets: PresetsClient,
  vault: string,
  deck: string,
): Promise<DeckPresetResult> => {
  try {
    const answer = await presets.getVaultDeckPreset({ vault, deck })
    const settings = answer.preset?.settings
    if (!answer.preset || !settings) {
      return asFailure(formatErrorCodeMessage(answer.error) || UNREAD)
    }
    return asValue({
      deck,
      held: {
        path: answer.preset.path,
        name: answer.preset.title,
        settings: { ...settings, goal: goalOf[settings.goal] ?? 'minutes' },
        problems: answer.preset.problems,
        stopsOn: answer.preset.stopsOn,
      },
    })
  } catch {
    // A preset that could not be asked for is refused in the same words as one
    // whose settings would not read, and that is what is drawn.
    return asFailure(UNREAD)
  }
}
