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
export interface TextEditorTabDeps {
  /** The settings file as its person wrote it, and where it stands. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file written whole, presenting the file it was last read as. A
   * file it cannot read is refused, and a file standing at anything else is
   * answered `changed` with nothing written.
   */
  saveSettingsFile(
    written: string,
    seen: string | null,
  ): Promise<{ readonly changed: boolean }>
}

/** What one tab of the settings file holds. */
export type TextEditorTabState = ReturnType<typeof useTextEditor>

/**
 * The file as it stands, what is typed over it, and what is wrong with what was
 * typed. It is kept the way a note is kept: a keep presents the file the tab
 * last read, and a file that moved past it stands stale until the person
 * keeps theirs or takes the file's.
 */
export function useTextEditor(core: TextEditorTabDeps, readSettings: () => void) {
  /** The bytes the file held when it was last read. */
  const held = ref('')
  const typed = ref('')
  const wrong = ref('')

  /** Whether what is typed differs from what the file held. */
  const changed = computed(() => typed.value !== held.value)

  /** Whether the file has been read at all. */
  const read = ref(false)

  /** Whether the file moved past what was last read, so keeping it stopped. */
  const isStale = ref(false)

  const again = async (): Promise<void> => {
    let answer: Awaited<ReturnType<TextEditorTabDeps['getSettingsFile']>>
    try {
      answer = await core.getSettingsFile()
    } catch (thrown) {
      wrong.value = `${words.unread} ${formatErrorMessage(thrown)}`
      return
    }
    held.value = answer.written
    typed.value = answer.written
    wrong.value = ''
    read.value = true
    isStale.value = false
  }

  /**
   * What was typed written into the file, presenting what it is given. A file
   * the settings cannot be read out of is refused, and every row of the settings
   * page is read again once one has been written.
   */
  const writes = async (baseline: string | null): Promise<void> => {
    if (!read.value) return
    let answer: Awaited<ReturnType<TextEditorTabDeps['saveSettingsFile']>>
    try {
      answer = await core.saveSettingsFile(typed.value, baseline)
    } catch (thrown) {
      wrong.value = `${words.unwritten} ${formatErrorMessage(thrown)}`
      return
    }
    // Nothing was written, and the tab stands stale until the person says
    // which of the two is theirs.
    if (answer.changed) {
      isStale.value = true
      return
    }
    held.value = typed.value
    wrong.value = ''
    isStale.value = false
    readSettings()
  }

  /** What was typed written into the file the tab read. */
  const save = (): Promise<void> => writes(held.value)

  /** Keep: what is typed goes to the file, whatever the file now holds. */
  const keep = async (): Promise<void> => {
    if (!isStale.value) return
    await writes(null)
  }

  return {
    /** What stands in the editor. It is moved by `type` and nothing else. */
    text: readonly(typed),
    type: (said: string) => void (typed.value = said),
    /** What is wrong, and empty where nothing is. */
    errorMessage: readonly(wrong),
    changed,
    /** Whether the file has been read at all. */
    read: readonly(read),
    /** Whether the file moved past what was read. */
    isStale: readonly(isStale),
    again,
    save,
    keep,
    /** Take: the file is read again, and that read replaces what is typed. */
    take: again,
  }
}
