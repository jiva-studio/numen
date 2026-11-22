/**
 * The text of the recovery words typed to restore an identity; typing clears the problem shown.
 */
import { ref, type Ref } from 'vue'

export function useRestoreText(problem: Ref<string>) {
  const restoreText = ref('')

  const setRestoreText = (value: string): void => {
    restoreText.value = value
    problem.value = ''
  }

  return { restoreText, setRestoreText }
}
