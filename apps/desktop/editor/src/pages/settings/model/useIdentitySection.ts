/**
 * The state of the identity section: showing the recovery words of a new identity and checking
 * three of them in two steps, restoring from the twelve words in a dialog, and deleting the identity from this device. The words
 * live in this state only while a setup is open; leaving the page or cancelling drops them.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'
import { useIdentityDelete } from './useIdentityDelete'
import { useIdentityCall } from './useIdentityCall'
import { useRestoreWords } from './useRestoreWords'
import { useWordsConfirmation } from './useWordsConfirmation'
import { useSetupStage } from './useSetupStage'

type Step = 'idle' | 'setup' | 'restore'

export function useIdentitySection(getIdentity: () => InstallationIdentity) {
  const identity = computed(() => getIdentity().identity.value)
  const step = ref<Step>('idle')
  const call = useIdentityCall()
  const { problem, isBusy, run } = call
  const setup = useSetupStage(problem)
  const { stage, recoveryWords } = setup
  const confirmation = useWordsConfirmation(recoveryWords)
  const restoring = useRestoreWords(problem)
  const deletion = useIdentityDelete(getIdentity, run, () => isBusy.value)

  onMounted(() => {
    runInBackground(getIdentity().load(), (caughtError) => {
      problem.value = formatError(caughtError)
    })
  })

  onBeforeUnmount(() => {
    reset()
  })

  function reset(): void {
    step.value = 'idle'
    setup.clear()
    confirmation.clear()
    restoring.clear()
    problem.value = ''
    deletion.cancelDelete()
  }

  const startSetup = async (): Promise<void> => {
    if (isBusy.value) return
    const answer = await run(() => getIdentity().generateWords())
    if (!answer.ok) return
    recoveryWords.value = answer.value
    confirmation.pick()
    stage.value = 'words'
    step.value = 'setup'
  }

  const startRestore = (): void => {
    reset()
    step.value = 'restore'
  }

  const setEntered = (position: number, value: string): void => {
    confirmation.setEntered(position, value)
    problem.value = ''
  }

  const keep = async (): Promise<void> => {
    if (isBusy.value) return
    if (!confirmation.verify()) {
      problem.value = words.identityNotValid
      return
    }
    const answer = await run(() => getIdentity().create(recoveryWords.value))
    if (answer.ok) reset()
  }

  const restore = async (): Promise<void> => {
    if (isBusy.value) return
    const answer = await run(() => getIdentity().create(restoring.readWords()))
    if (answer.ok) reset()
  }

  return {
    identity,
    step,
    setup,
    call,
    confirmation,
    restoring,
    deletion,
    actions: {
      startSetup,
      startRestore,
      showCheck: setup.showCheck,
      showWords: setup.showWords,
      cancel: reset,
      setEntered,
      keep,
      restore,
    },
  }
}
