/** What the application answers about the preset one deck is scheduled by. */
import { StopReason } from '@numen/protocol'
import type { Refusal } from '@numen/protocol'
import { goalOf, formatErrorCodeMessage } from '@numen/wire'

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
    refusal?: Refusal | undefined
  }>
}

/** What was answered about one deck's preset. */
export interface DeckPresetResult {
  readonly deck: string
  /** The preset, and null where it was not read. */
  readonly held: {
    path: string
    name: string
    settings: Settings
    problems: readonly string[]
    stopsOn: StopReason
  } | null
  /** Why it was not read, in the words to show, and empty where it was. */
  readonly refused: string
}

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
      return { deck, held: null, refused: formatErrorCodeMessage(answer.refusal) || UNREAD }
    }
    return {
      deck,
      held: {
        path: answer.preset.path,
        name: answer.preset.title,
        settings: { ...settings, goal: goalOf[settings.goal] ?? 'minutes' },
        problems: answer.preset.problems,
        stopsOn: answer.preset.stopsOn,
      },
      refused: '',
    }
  } catch {
    // A preset that could not be asked for is refused in the same words as one
    // whose settings would not read, and the refusal is what is drawn.
    return { deck, held: null, refused: UNREAD }
  }
}
