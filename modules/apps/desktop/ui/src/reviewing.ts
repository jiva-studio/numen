/**
 * The hour a day of review begins at, on the clock on the wall.
 *
 * One setting, held as the hour the settings file holds. An answer given before
 * that hour is written into the day before, so the window opens on what this
 * installation is doing and choosing another hour writes it.
 */
import { ref } from 'vue'
import { troubleWords } from '@numen/wire'
import type { ReviewSettings } from './core'
import type { Voice } from './telling'

/** The hour an installation nobody has configured begins the day at. */
export const DEFAULT_STARTS = '04:00'

/** Everything this says in the window's voice. */
export interface Words {
  /** The setting could not be written. */
  readonly unturned: string
}

/** What this asks of the vault. */
export interface ReviewingDeps {
  /** The hour as the settings file holds it, and the latest the vault takes. */
  reviewing(): Promise<ReviewSettings>
  /** The hour written. What could not be written, and nothing where it was. */
  choosesReviewing(starts: string): Promise<string | null>
}

export function reviewing(core: ReviewingDeps, words: Words, said: Voice) {
  /** The hour in force. It opens where an installation nobody has configured begins. */
  const starts = ref(DEFAULT_STARTS)

  /**
   * How late in the day the vault takes an hour, which it says when it is asked
   * what it holds. Until it has, an hour is asked for no later than the one in
   * force, so none is offered that is refused.
   */
  const latest = ref(DEFAULT_STARTS)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    try {
      const held = await core.reviewing()
      starts.value = held.starts
      latest.value = held.latest
    } catch {
      // A vault that cannot be asked leaves the hour where it stands.
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
      failed = troubleWords(thrown)
    }
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'refusal')
    starts.value = was
  }

  return { starts, latest, start, chooses }
}
