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
import { DEFAULT_HEADINGS, nestingLadder as ladder } from '@/entities/settings'
import type { NestingSettings } from '@/entities/settings'
import type { NestingWords } from '../words'
import { createWriteGuard } from '../lib/writeGuard'

/** The command whose step offers the two, and the one that offers the counts. */
export const NESTING = 'nesting'
export const HEADINGS = 'headings'

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
  setNestingSettings(isNesting: boolean, headings?: number): Promise<string | null>
}

function createStepRow(
  id: string,
  title: string,
  ...[current, currentLabel]: [current: boolean, currentLabel: string]
): StepRow {
  return {
    id,
    title,
    ...(current ? { detail: currentLabel, isCurrent: true } : {}),
  }
}

function createNestingGroups(isNesting: boolean, words: NestingWords): readonly StepGroup[] {
  return [
    {
      id: NESTING,
      title: words.nestingGroup,
      items: [
        createStepRow(ON, words.on, isNesting, words.current),
        createStepRow(OFF, words.off, !isNesting, words.current),
      ],
    },
  ]
}

function createHeadingsGroups(
  ends: { least: number; most: number },
  headings: number,
  words: NestingWords,
): readonly StepGroup[] {
  return [
    {
      id: HEADINGS,
      title: words.headingsGroup,
      items: ladder(ends.least, ends.most).map((count) =>
        createStepRow(`${count}`, `${count}`, count === headings, words.current),
      ),
    },
  ]
}

export function useNestingSetting(core: NestingDeps, words: NestingWords, write: MessageWriter) {
  const isNesting = ref(true)
  const headings = ref(DEFAULT_HEADINGS)
  const ends = ref({ least: DEFAULT_HEADINGS, most: DEFAULT_HEADINGS })

  const guard = createWriteGuard()

  const start = async (): Promise<void> => {
    let settings: NestingSettings
    try {
      settings = await core.getNestingSettings()
    } catch {
      return
    }
    if (guard.isIdle()) {
      isNesting.value = settings.isNesting
      headings.value = settings.headings
    }
    ends.value = { least: settings.least, most: settings.most }
  }

  const getNestingGroups = (): readonly StepGroup[] => createNestingGroups(isNesting.value, words)
  const getHeadingsGroups = (): readonly StepGroup[] =>
    createHeadingsGroups(ends.value, headings.value, words)

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
    const { failed, isLatest } = await guard.run(guard.beginChoice(), () => core.setNestingSettings(now))
    if (!failed) return
    write(`${words.unturned} ${failed}`, 'error')
    if (isLatest) isNesting.value = was
  }

  /**
   * The count chosen, written into the settings beside the switch the window
   * already knows. A setting that could not be written is said, and the window
   * goes back to what the settings hold.
   */
  const chooseCount = async (item: string): Promise<void> => {
    const now = Number(item)
    const was = headings.value
    const { least, most } = ends.value
    if (!Number.isInteger(now) || now < least || now > most || now === was) return
    write('')
    headings.value = now
    const { failed, isLatest } = await guard.run(guard.beginChoice(), () =>
      core.setNestingSettings(isNesting.value, now),
    )
    if (!failed) return
    write(`${words.unturned} ${failed}`, 'error')
    if (isLatest) headings.value = was
  }

  return {
    isNesting,
    headings,
    ends,
    start,
    getNestingGroups,
    getHeadingsGroups,
    choose,
    chooseCount,
  }
}
