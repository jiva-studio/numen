/**
 * One note of the vault as the note editor view holds it: read once, and written after each edit.
 */
import { ref } from 'vue'
import type { VaultNotes } from '@numen/plugin-sdk'

export type NoteFileState = ReturnType<typeof useNoteFile>

const getErrorMessage = (error: unknown): string =>
  error instanceof Error ? error.message : String(error)

/**
 * An edit made while the note is still being read keeps what was typed. The error shown is the one
 * of the latest write.
 */
export function useNoteFile(path: string, notes: Pick<VaultNotes, 'read' | 'write'>) {
  const body = ref('')
  const errorMessage = ref('')
  const isLoading = ref(true)
  let hasEdited = false
  let writes = 0

  const write = (text: string) => {
    const turn = ++writes
    notes.write(path, text).then(
      () => {
        if (turn === writes) errorMessage.value = ''
      },
      (error: unknown) => {
        if (turn === writes) errorMessage.value = getErrorMessage(error)
      },
    )
  }

  notes.read(path).then(
    (result) => {
      isLoading.value = false
      if (!hasEdited) body.value = result.body
      if (result.error) errorMessage.value = result.error
    },
    (error: unknown) => {
      isLoading.value = false
      errorMessage.value = getErrorMessage(error)
    },
  )

  return {
    path,
    body,
    errorMessage,
    isLoading,
    updateBody: (text: string) => {
      hasEdited = true
      body.value = text
      write(text)
    },
    save: () => {
      if (isLoading.value && !hasEdited) return
      write(body.value)
    },
  }
}
