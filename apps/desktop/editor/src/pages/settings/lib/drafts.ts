/**
 * Text typed into the text settings and not written yet, by key.
 *
 * The settings are drawn again whenever they are read back, and a field bound to the written value
 * would lose what is typed before the field is left.
 */
import { ref } from 'vue'

export function createTextDrafts(write: (key: string, text: string) => void) {
  const drafts = ref<Readonly<Record<string, string>>>({})

  /** The text typed at a key, and nothing where none is waiting to be written. */
  const getDraft = (key: string): string | undefined => drafts.value[key]

  const setDraft = (key: string, text: string): void => {
    drafts.value = { ...drafts.value, [key]: text }
  }

  /** Writes the text at a key and drops it from what is waiting. */
  const writeDraft = (key: string, text: string): void => {
    const rest = { ...drafts.value }
    delete rest[key]
    drafts.value = rest
    write(key, text)
  }

  return { getDraft, setDraft, writeDraft }
}
