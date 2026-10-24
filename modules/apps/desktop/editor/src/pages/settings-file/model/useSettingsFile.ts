/**
 * The settings file itself, opened whole in a tab of its own.
 *
 * The tab holds the bytes the file stands as, and hands them back as they were
 * typed. A file the settings cannot be read out of is refused with what is
 * wrong with it, and the file is left as it was.
 */
import { computed, readonly, ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { WORDS as words } from '../words'

/** What this asks of the vault. */
export interface SettingsFileTabDeps {
  /** The settings file as its person wrote it, and where it stands. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file written whole, presenting the file it was last read as. A
   * file it cannot read is refused, and a file standing at anything else is
   * answered `changed` with nothing written.
   */
  saveSettingsFile(written: string, seen: string | null): Promise<{ readonly isChanged: boolean }>
}

/** What one tab of the settings file holds. */
export type SettingsFileTabState = ReturnType<typeof useSettingsFile>

/**
 * The file as it stands, what is typed over it, and what is wrong with what was
 * typed.
 */
export function useSettingsFile(core: SettingsFileTabDeps, readSettings: () => void) {
  /** The bytes the file held when it was last read. */
  const held = ref('')
  const typed = ref('')
  const wrong = ref('')

  /** Whether what is typed differs from what the file held. */
  const changed = computed(() => typed.value !== held.value)

  /** Whether the file has been read at all. */
  const isRead = ref(false)

  /** Whether the file moved past what was last read, so keeping it stopped. */
  const isStale = ref(false)

  const reload = async (): Promise<void> => {
    let answer: Awaited<ReturnType<SettingsFileTabDeps['getSettingsFile']>>
    try {
      answer = await core.getSettingsFile()
    } catch (thrown) {
      wrong.value = `${words.unread} ${formatErrorMessage(thrown)}`
      return
    }
    held.value = answer.written
    typed.value = answer.written
    wrong.value = ''
    isRead.value = true
    isStale.value = false
  }

  /**
   * What was typed written into the file.
   */
  const write = async (baseline: string | null): Promise<void> => {
    if (!isRead.value) return
    let answer: Awaited<ReturnType<SettingsFileTabDeps['saveSettingsFile']>>
    try {
      answer = await core.saveSettingsFile(typed.value, baseline)
    } catch (thrown) {
      wrong.value = `${words.unwritten} ${formatErrorMessage(thrown)}`
      return
    }
    if (answer.isChanged) {
      isStale.value = true
      return
    }
    held.value = typed.value
    wrong.value = ''
    isStale.value = false
    readSettings()
  }

  /** What was typed written into the file the tab read. */
  const save = (): Promise<void> => write(held.value)

  /** Keep: what is typed goes to the file, whatever the file now holds. */
  const keep = async (): Promise<void> => {
    if (!isStale.value) return
    await write(null)
  }

  // eslint-disable-next-line numen/max-composable-return-properties
  return {
    text: readonly(typed),
    type: (content: string) => void (typed.value = content),
    errorMessage: readonly(wrong),
    changed,
    isRead: readonly(isRead),
    isStale: readonly(isStale),
    reload,
    save,
    keep,
    take: reload,
  }
}
