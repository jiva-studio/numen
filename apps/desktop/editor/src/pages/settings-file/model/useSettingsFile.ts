/**
 * The settings file, opened as raw text in its own tab.
 *
 * The tab keeps the file's bytes from the last read and saves the draft unchanged. When
 * the draft does not parse as settings, the save fails with the parse error and the file
 * stays unchanged.
 */
import { computed, readonly, ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { WORDS as words } from '../words'

/** The vault operations the tab uses. */
export interface SettingsFileTabDeps {
  /** The raw settings file text and its path. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * Writes the whole settings file when the file on disk still equals `expectedVersion`.
   * Text that does not parse throws; a file that changed on disk returns `isChanged`
   * and nothing is written.
   */
  saveSettingsFile(
    content: string,
    expectedVersion: string | null,
  ): Promise<{ readonly isChanged: boolean }>
}

/** The state of one settings file tab. */
export type SettingsFileTabState = ReturnType<typeof useSettingsFile>

/** The file's saved text, the edited draft, and the last read or write error. */
export function useSettingsFile(core: SettingsFileTabDeps, readSettings: () => void) {
  /** The file's bytes at the last read or save. */
  const savedText = ref('')
  const draft = ref('')
  const error = ref('')

  /** Whether the draft differs from the saved text. */
  const changed = computed(() => draft.value !== savedText.value)

  /** Whether the file has been read at all. */
  const isRead = ref(false)

  /** Whether the file changed on disk after the last read, which stops saving. */
  const isStale = ref(false)

  const reload = async (): Promise<void> => {
    let answer: Awaited<ReturnType<SettingsFileTabDeps['getSettingsFile']>>
    try {
      answer = await core.getSettingsFile()
    } catch (thrown) {
      error.value = `${words.unread} ${formatErrorMessage(thrown)}`
      return
    }
    savedText.value = answer.written
    draft.value = answer.written
    error.value = ''
    isRead.value = true
    isStale.value = false
  }

  /** Writes the draft to the file. */
  const write = async (baseline: string | null): Promise<void> => {
    if (!isRead.value) return
    let answer: Awaited<ReturnType<SettingsFileTabDeps['saveSettingsFile']>>
    try {
      answer = await core.saveSettingsFile(draft.value, baseline)
    } catch (thrown) {
      error.value = `${words.unwritten} ${formatErrorMessage(thrown)}`
      return
    }
    if (answer.isChanged) {
      isStale.value = true
      return
    }
    savedText.value = draft.value
    error.value = ''
    isStale.value = false
    readSettings()
  }

  /** Writes the draft when the file still equals the last read. */
  const save = (): Promise<void> => write(savedText.value)

  /** Writes the draft over a file that changed on disk. */
  const keep = async (): Promise<void> => {
    if (!isStale.value) return
    await write(null)
  }

  // eslint-disable-next-line numen/max-composable-return-properties
  return {
    text: readonly(draft),
    type: (content: string) => void (draft.value = content),
    errorMessage: readonly(error),
    changed,
    isRead: readonly(isRead),
    isStale: readonly(isStale),
    reload,
    save,
    keep,
    take: reload,
  }
}
