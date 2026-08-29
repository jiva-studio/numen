/**
 * Whether a node in the plex hangs the parts of its note under the box, and how
 * many of them stand there at once.
 *
 * Two settings, offered as two lists: the first is two rows, the second a
 * ladder from one end of what the setting takes to the other. The row in force
 * is the one the setting holds, so a list opens on what this installation is
 * doing, and choosing another writes it. The plex reads both as it draws, so
 * the picture answers the choice as it is made.
 */
import { ref } from 'vue'
import type { Offered, Offering } from './commanding'
import type { Says } from './telling'

/** The command whose step offers the two, and the one that offers the counts. */
export const HANGING = 'hanging'
export const PARTS = 'parts'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/**
 * How many parts a node may be asked to hang, at each end. A node hangs at
 * least one, and twelve of them reach the foot of a window the plex is drawn in.
 */
const LEAST = 1
const MOST = 12

/** Everything this says in the window's voice. */
export interface Words {
  /** The band the setting is drawn in, and the two it is. */
  readonly hangingBand: string
  readonly on: string
  readonly off: string
  /** The band the counts are drawn in. */
  readonly partsBand: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}

/** Whether a node hangs the parts of its note, and how many stand at once. */
export interface Hanging {
  readonly hangs: boolean
  readonly parts: number
}

/** What this asks of the vault. */
export interface Called {
  /** The two settings, as the settings file holds them. */
  hanging(): Promise<Hanging>
  /**
   * The settings written. What could not be written, and nothing where it was.
   * A count left out stands as it is.
   */
  choosesHanging(hangs: boolean, parts?: number): Promise<string | null>
}

/** The counts offered, from one end of what the setting takes to the other. */
const ladder = (): readonly number[] =>
  Array.from({ length: MOST - LEAST + 1 }, (_, at) => LEAST + at)

export function hanging(core: Called, words: Words, said: Says) {
  /**
   * The two settings. They open on what an installation nobody has configured
   * does, and are asked of the vault as the window opens.
   */
  const hangs = ref(true)
  const parts = ref(6)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    let held: Hanging
    try {
      held = await core.hanging()
    } catch {
      return
    }
    hangs.value = held.hangs
    parts.value = held.parts
  }

  /** One row of a list, saying whether it is the value in force. */
  const row = (id: string, title: string, inForce: boolean): Offered => ({
    id,
    title,
    ...(inForce ? { detail: words.current, inForce: true } : {}),
  })

  /** The two, in one band named for the setting they are of. */
  const offers = (): readonly Offering[] => [
    {
      id: HANGING,
      title: words.hangingBand,
      items: [row(ON, words.on, hangs.value), row(OFF, words.off, !hangs.value)],
    },
  ]

  /** The counts, in one band of their own. */
  const counts = (): readonly Offering[] => [
    {
      id: PARTS,
      title: words.partsBand,
      items: ladder().map((count) => row(`${count}`, `${count}`, count === parts.value)),
    },
  ]

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

  /**
   * The count chosen, written into the settings beside the switch the window
   * already knows. A setting that could not be written is said, and the window
   * goes back to what the settings hold.
   */
  const choosesCount = async (item: string): Promise<void> => {
    const now = Number(item)
    const was = parts.value
    if (!Number.isInteger(now) || now < LEAST || now > MOST || now === was) return
    said('')
    parts.value = now

    const failed = await core.choosesHanging(hangs.value, now)
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'refusal')
    parts.value = was
  }

  return { hangs, parts, start, offers, counts, chooses, choosesCount }
}
