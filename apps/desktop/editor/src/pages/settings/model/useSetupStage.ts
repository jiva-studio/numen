/**
 * The two steps of an identity setup: the recovery words shown once, then the check of three of
 * them. Moving between the steps clears the problem of the step left.
 */
import { ref, type Ref } from 'vue'

type SetupStage = 'words' | 'check'

export function useSetupStage(problem: Ref<string>) {
  const stage = ref<SetupStage>('words')
  const recoveryWords = ref<readonly string[]>([])

  const showCheck = (): void => {
    problem.value = ''
    stage.value = 'check'
  }

  const showWords = (): void => {
    problem.value = ''
    stage.value = 'words'
  }

  const clear = (): void => {
    stage.value = 'words'
    recoveryWords.value = []
  }

  return { stage, recoveryWords, showCheck, showWords, clear }
}
