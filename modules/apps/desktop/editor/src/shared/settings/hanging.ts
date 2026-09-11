/**
 * Whether a node in the plex hangs the parts of its note under the box, and how
 * many of them stand there at once.
 *
 * Two settings, offered as two lists. A list opens on the row the setting
 * holds, choosing another writes it, and the plex reads both as it draws.
 */
import { ref } from 'vue'
import type { StepGroup, StepRow } from '../command/lists'
import type { MessageWriter } from '../notices/messages'

/** The command whose step offers the two, and the one that offers the counts. */
export const HANGING = 'hanging'
export const PARTS = 'parts'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/** How many a node hangs where the settings name no number. */
export const DEFAULT_PARTS = 6

/** Everything this says in the window's voice. */
export interface Words {
  /** The group the setting is drawn in, and the two it is. */
  readonly hangingGroup: string
  readonly on: string
  readonly off: string
  /** The group the counts are drawn in. */
  readonly partsGroup: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}

/** Whether a node hangs the parts of its note, and how many stand at once. */
export interface HangingSettings {
  readonly hangs: boolean
  readonly parts: number
  /** How many the vault takes, at each end. A count outside them is refused. */
  readonly least: number
  readonly most: number
}

/** What this asks of the vault. */
export interface HangingDeps {
  /** The two settings, as the settings file holds them. */
  hanging(): Promise<HangingSettings>
  /**
   * The settings written. What could not be written, and nothing where it was.
   * A count left out stands as it is.
   */
  choosesHanging(hangs: boolean, parts?: number): Promise<string | null>
}

/** The counts offered, from one end of what the setting takes to the other. */
const ladder = (least: number, most: number): readonly number[] =>
  Array.from({ length: Math.max(0, most - least + 1) }, (_, at) => least + at)

export function useHangingSetting(core: HangingDeps, words: Words, said: MessageWriter) {
  /**
   * The two settings. They open on what an installation nobody has configured
   * does, and are asked of the vault as the window opens.
   */
  const hangs = ref(true)
  const parts = ref(DEFAULT_PARTS)

  /**
   * How many the vault takes, which it says when it is asked what it holds.
   * Nothing is offered until it has, so no count is offered that is refused.
   */
  const ends = ref({ least: DEFAULT_PARTS, most: DEFAULT_PARTS })

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    let held: HangingSettings
    try {
      held = await core.hanging()
    } catch {
      // A vault that cannot be asked leaves both settings where they stand.
      return
    }
    hangs.value = held.hangs
    parts.value = held.parts
    ends.value = { least: held.least, most: held.most }
  }

  /** One row of a list, saying whether it is the value in force. */
  const row = (id: string, title: string, inForce: boolean): StepRow => ({
    id,
    title,
    ...(inForce ? { detail: words.current, inForce: true } : {}),
  })

  /** The two, in one group named for the setting they are of. */
  const offers = (): readonly StepGroup[] => [
    {
      id: HANGING,
      title: words.hangingGroup,
      items: [row(ON, words.on, hangs.value), row(OFF, words.off, !hangs.value)],
    },
  ]

  /** The counts, in one group of their own. */
  const counts = (): readonly StepGroup[] => [
    {
      id: PARTS,
      title: words.partsGroup,
      items: ladder(ends.value.least, ends.value.most).map((count) =>
        row(`${count}`, `${count}`, count === parts.value),
      ),
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
    said(`${words.unturned} ${failed}`, 'error')
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
    const { least, most } = ends.value
    if (!Number.isInteger(now) || now < least || now > most || now === was) return
    said('')
    parts.value = now

    const failed = await core.choosesHanging(hangs.value, now)
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'error')
    parts.value = was
  }

  return { hangs, parts, ends, start, offers, counts, chooses, choosesCount }
}
