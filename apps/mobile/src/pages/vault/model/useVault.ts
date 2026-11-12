import { onMounted, ref } from 'vue'
import { startCore, type Core } from '@/entities/core'
import { formatError } from '@/shared/api/errors'

/**
 * The core the page started and the note it opens on. A failure to start the core or to read the
 * opening note is reported as `errorMessage`.
 */
export function useVault() {
  const core = ref<Core | null>(null)
  const errorMessage = ref('')

  /** The note the vault opens on, and nothing while it has none. */
  const openingPath = ref<string | null>(null)

  /** The note being written, and nothing while none is. */
  const editingPath = ref<string | null>(null)

  onMounted(async () => {
    try {
      const started = await startCore()
      core.value = started
      openingPath.value = await started.getOpeningPath()
      editingPath.value = openingPath.value
    } catch (why) {
      errorMessage.value = formatError(why)
    }
  })

  return { core, errorMessage, openingPath, editingPath }
}
