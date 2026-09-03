/**
 * The settings file itself, opened whole in a tab of its own.
 *
 * The tab holds the bytes the file stands as, and hands them back as they were
 * typed. A file the settings cannot be read out of is refused with what is
 * wrong with it, and the file is left as it was.
 */
import { computed, ref } from 'vue'
import type { Host, Kind } from '../windowing'
import { CONFIGURATION } from '../workspace'
import ConfigurationTab from './ConfigurationTab.vue'
import { WORDS as words } from './words'

/** What this asks of the vault. */
export interface Called {
  /** The settings file as its person wrote it, and where it stands. */
  settingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /** The settings file written whole. A file it cannot read is refused. */
  writesSettingsFile(written: string): Promise<void>
}

/** What one tab of the settings file holds. */
export type Held = ReturnType<typeof holding>

/** What is wrong, as a person reads it. */
const reason = (thrown: unknown): string =>
  thrown instanceof Error ? thrown.message : `${thrown}`

/**
 * The file as it stands, what is typed over it, and what is wrong with what was
 * typed. It is kept the way a note is kept, and keeping it reads the file
 * again, so what the tab holds is what the file holds.
 */
export function holding(core: Called, reads: () => void) {
  /** The bytes the file held when it was last read. */
  const held = ref('')
  const typed = ref('')
  const wrong = ref('')

  /** Whether what is typed differs from what the file held. */
  const changed = computed(() => typed.value !== held.value)

  /** Whether the file has been read at all. */
  const read = ref(false)

  const again = async (): Promise<void> => {
    let answer: Awaited<ReturnType<Called['settingsFile']>>
    try {
      answer = await core.settingsFile()
    } catch (thrown) {
      wrong.value = `${words.unread} ${reason(thrown)}`
      return
    }
    held.value = answer.written
    typed.value = answer.written
    wrong.value = ''
    read.value = true
  }

  /**
   * What was typed written into the file. A file the settings cannot be read
   * out of is refused, and every row of the settings page is read again once
   * one has been written.
   */
  const keeps = async (): Promise<void> => {
    if (!read.value) return
    try {
      await core.writesSettingsFile(typed.value)
    } catch (thrown) {
      wrong.value = `${words.unwritten} ${reason(thrown)}`
      return
    }
    held.value = typed.value
    wrong.value = ''
    reads()
  }

  return {
    /** What stands in the editor, and what is typed into it. */
    text: () => typed.value,
    types: (said: string) => void (typed.value = said),
    /** What is wrong, and empty where nothing is. */
    saying: () => wrong.value,
    changed: () => changed.value,
    /** Whether the file has been read at all. */
    read: () => read.value,
    again,
    keeps,
  }
}

/**
 * The settings file's tab. There is one file, so opening it again is the tab it
 * already stands in.
 */
export function configuring(host: Host, core: Called, reads: () => void) {
  const kind: Kind<Held> = {
    kind: CONFIGURATION,
    opens: () => {
      const held = holding(core, reads)
      void held.again()
      return held
    },
    called: () => words.called,
    marked: (held) => (held.changed() ? '•' : undefined),
    draws: ConfigurationTab,
    identity: () => CONFIGURATION,
  }

  /** The file put in front of the person, beside what they were looking at. */
  const shows = (): void => void host.beside(CONFIGURATION)

  return { kind, shows }
}
