/**
 * The hour a day of review begins at, on the clock on the wall.
 *
 * One setting, held as the hour the settings file holds. An answer given before
 * that hour is written into the day before, so the window opens on what this
 * installation is doing and choosing another hour writes it.
 */
import { ref } from 'vue'
import type { Says } from './telling'

/** The hour an installation nobody has configured begins the day at. */
export const DEFAULT_STARTS = '04:00'

/** How late in the day the setting takes an hour, which is noon. */
export const LATEST_STARTS = '12:00'

/** Everything this says in the window's voice. */
export interface Words {
  /** The setting could not be written. */
  readonly unturned: string
}

/** What this asks of the vault. */
export interface Called {
  /** The hour, as the settings file holds it. */
  reviewing(): Promise<string>
  /** The hour written. What could not be written, and nothing where it was. */
  choosesReviewing(starts: string): Promise<string | null>
}

export function reviewing(core: Called, words: Words, said: Says) {
  /** The hour in force. It opens where an installation nobody has configured begins. */
  const starts = ref(DEFAULT_STARTS)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    try {
      starts.value = await core.reviewing()
    } catch {
      return
    }
  }

  /**
   * The hour chosen, written into the settings. An hour that could not be
   * written is said, and the window goes back to what the settings hold.
   */
  const chooses = async (hour: string): Promise<void> => {
    const was = starts.value
    if (hour === was) return
    said('')
    starts.value = hour

    let failed: string | null
    try {
      failed = await core.choosesReviewing(hour)
    } catch (thrown) {
      failed = thrown instanceof Error ? thrown.message : `${thrown}`
    }
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'refusal')
    starts.value = was
  }

  return { starts, start, chooses }
}
