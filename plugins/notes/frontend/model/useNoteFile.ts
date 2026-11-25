import { formatErrorMessage } from '@numen/plugin-sdk'
import { ref } from 'vue'
import type { VaultNotes } from '@numen/plugin-sdk'
const getErrorMessage = (error: unknown): string => formatErrorMessage(error)

/** Each save starts from the last version read or successfully written by this editor. */
export function useNoteFile(path: string, notes: Pick<VaultNotes, 'read' | 'write'>) {
  const body = ref('')
  const errorMessage = ref('')
  const isLoading = ref(true)
  let observed: { body: string; at: string } | undefined
  let isSaving = false

  async function save() {
    if (isSaving || !observed) return
    isSaving = true
    try {
      while (body.value !== observed.body) {
        const text: string = body.value
        const at = await notes.write(path, text, observed)
        observed = { body: text, at }
        errorMessage.value = ''
      }
    } catch (error: unknown) {
      errorMessage.value = getErrorMessage(error)
    } finally {
      isSaving = false
    }
  }

  notes.read(path).then(
    (result) => {
      isLoading.value = false
      body.value = result.body
      if (result.error || !result.at) {
        errorMessage.value = result.error || 'The version of this note could not be read.'
        return
      }
      observed = { body: result.body, at: result.at }
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
      body.value = text
      return save()
    },
    save,
  }
}
