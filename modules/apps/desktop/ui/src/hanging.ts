/**
 * Whether a node in the plex hangs the parts of its note under the box.
 *
 * One setting, offered as two rows. The row in force is the one the setting
 * holds, so the list opens on what this installation is doing, and choosing the
 * other writes it. The plex reads the setting as it draws, so the picture
 * answers the choice as it is made.
 */
import { ref } from 'vue'
import type { Offering } from './commanding'
import type { Says } from './telling'

/** The command whose step offers the two. */
export const HANGING = 'hanging'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/** Everything this says in the window's voice. */
export interface Words {
  /** The band the setting is drawn in, and the two it is. */
  readonly hangingBand: string
  readonly on: string
  readonly off: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}

/** What this asks of the vault. */
export interface Called {
  /** Whether a node hangs the parts of its note, as the settings hold it. */
  hanging(): Promise<boolean>
  /** The setting written. What could not be written, and nothing where it was. */
  choosesHanging(hangs: boolean): Promise<string | null>
}

export function hanging(core: Called, words: Words, said: Says) {
  /**
   * Whether a node hangs the parts of its note. It opens on what an
   * installation nobody has configured does, and is asked of the vault as the
   * window opens.
   */
  const hangs = ref(true)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    try {
      hangs.value = await core.hanging()
    } catch {
      return
    }
  }

  /** The two, in one band named for the setting they are of. */
  const offers = (): readonly Offering[] => {
    const row = (id: string, title: string, inForce: boolean) => ({
      id,
      title,
      ...(inForce ? { detail: words.current, inForce: true } : {}),
    })
    return [
      {
        id: HANGING,
        title: words.hangingBand,
        items: [row(ON, words.on, hangs.value), row(OFF, words.off, !hangs.value)],
      },
    ]
  }

  /**
   * The row chosen, written into the settings. A setting that could not be
   * written is said, and the window goes back to what the settings hold.
   */
  const chooses = async (item: string): Promise<void> => {
    if (item !== ON && item !== OFF) return
    const was = hangs.value
    const now = item === ON
    if (now === was) return
    said('')
    hangs.value = now

    const failed = await core.choosesHanging(now)
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'refusal')
    hangs.value = was
  }

  return { hangs, start, offers, chooses }
}
