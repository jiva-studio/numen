/**
 * Whether a node in the plex nests the headings of its note under the box, and how
 * many of them stand there at once.
 *
 * Two settings, offered as two lists. A list opens on the row the setting
 * holds, choosing another writes it, and the plex reads both as it draws.
 */
import { ref } from 'vue'
import type { StepGroup, StepRow } from '@/features/command-palette/@x/settings-commands'
import type { MessageWriter } from '@/shared/notices/messages'
import { DEFAULT_PARTS, nestingLadder as ladder } from '@/entities/settings'
import type { NestingSettings } from '@/entities/settings'
import type { NestingWords } from '../words'

/** The command whose step offers the two, and the one that offers the counts. */
export const NESTING = 'nesting'
export const PARTS = 'parts'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/** What this asks of the vault. */
export interface NestingDeps {
  /** The two settings, as the settings file holds them. */
  getNestingSettings(): Promise<NestingSettings>
  /**
   * The settings written. What could not be written, and nothing where it was.
   * A count left out stands as it is.
   */
  setNestingSettings(isNesting: boolean, parts?: number): Promise<string | null>
}

export function useNestingSetting(core: NestingDeps, words: NestingWords, write: MessageWriter) {
  /**
   * The two settings. They open on what an installation nobody has configured
   * does, and are asked of the vault as the window opens.
   */
  const isNesting = ref(true)
  const parts = ref(DEFAULT_PARTS)

  /**
   * How many the vault takes, which it says when it is asked what it holds.
   * Nothing is offered until it has, so no count is offered that is refused.
   */
  const ends = ref({ least: DEFAULT_PARTS, most: DEFAULT_PARTS })

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    let settings: NestingSettings
    try {
      settings = await core.getNestingSettings()
    } catch {
      // A vault that cannot be asked leaves both settings where they stand.
      return
    }
    isNesting.value = settings.isNesting
    parts.value = settings.parts
    ends.value = { least: settings.least, most: settings.most }
  }

  /** One row of a list, saying whether it is the value in force. */
  const row = (id: string, title: string, current: boolean): StepRow => ({
    id,
    title,
    ...(current ? { detail: words.current, isCurrent: true } : {}),
  })

  /** The two, in one group named for the setting they are of. */
  const getNestingGroups = (): readonly StepGroup[] => [
    {
      id: NESTING,
      title: words.nestingGroup,
      items: [row(ON, words.on, isNesting.value), row(OFF, words.off, !isNesting.value)],
    },
  ]

  /** The counts, in one group of their own. */
  const getPartsGroups = (): readonly StepGroup[] => [
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
  const choose = async (item: string): Promise<void> => {
    if (item !== ON && item !== OFF) return
    const was = isNesting.value
    const now = item === ON
    if (now === was) return
    write('')
    isNesting.value = now

    const failed = await core.setNestingSettings(now)
    if (!failed) return
    write(`${words.unturned} ${failed}`, 'error')
    isNesting.value = was
  }

  /**
   * The count chosen, written into the settings beside the switch the window
   * already knows. A setting that could not be written is said, and the window
   * goes back to what the settings hold.
   */
  const chooseCount = async (item: string): Promise<void> => {
    const now = Number(item)
    const was = parts.value
    const { least, most } = ends.value
    if (!Number.isInteger(now) || now < least || now > most || now === was) return
    write('')
    parts.value = now

    const failed = await core.setNestingSettings(isNesting.value, now)
    if (!failed) return
    write(`${words.unturned} ${failed}`, 'error')
    parts.value = was
  }

  return { isNesting, parts, ends, start, getNestingGroups, getPartsGroups, choose, chooseCount }
}
