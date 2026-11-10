import { ref } from 'vue'
import { getInitialOpenPath } from '@/entities/note'

/** The note the vault opens with, for whatever has nowhere else to start. */
export function useDisplayOpening() {
  /** Whether the vault holds a note to show at all. */
  const hasNote = ref(false)
  const path = ref('')

  /** Reads which note the vault opens with, and returns its path, empty for none. */
  const read = async (): Promise<string> => {
    const note = await getInitialOpenPath()
    hasNote.value = note !== null
    path.value = note?.path ?? ''
    return path.value
  }

  return { hasNote, path, read }
}
