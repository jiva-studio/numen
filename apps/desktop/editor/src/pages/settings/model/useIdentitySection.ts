/**
 * The state of the identity section: showing the recovery words of a new identity and checking
 * three of them in two steps, restoring from typed words, and deleting the identity from this device. The words
 * live in this state only while a setup is open; leaving the page or cancelling drops them.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'
import { useIdentityDelete } from './useIdentityDelete'
import { useIdentityCall } from './useIdentityCall'
import { useRestoreText } from './useRestoreText'
import { useWordsConfirmation } from './useWordsConfirmation'

type Step = 'idle' | 'setup' | 'restore'
type SetupStage = 'words' | 'check'

export function useIdentitySection(getIdentity: () => InstallationIdentity) {
  const identity = computed(() => getIdentity().identity.value)
  const step = ref<Step>('idle')
  const stage = ref<SetupStage>('words')
  const recoveryWords = ref<readonly string[]>([])
  const call = useIdentityCall()
  const { problem, isBusy, run } = call
  const confirmation = useWordsConfirmation(recoveryWords)
  const typing = useRestoreText(problem)
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
    stage.value = 'words'
    recoveryWords.value = []
    confirmation.clear()
    typing.restoreText.value = ''
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

  const showCheck = (): void => {
    problem.value = ''
    stage.value = 'check'
  }

  const showWords = (): void => {
    problem.value = ''
    stage.value = 'words'
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
    if (!confirmation.isConfirmed()) {
      problem.value = words.identityNotValid
      return
    }
    const answer = await run(() => getIdentity().create(recoveryWords.value))
    if (answer.ok) reset()
  }

  const restore = async (): Promise<void> => {
    if (isBusy.value) return
    const typed = typing.restoreText.value.split(/\s+/).filter((word) => word !== '')
    const answer = await run(() => getIdentity().create(typed))
    if (answer.ok) reset()
  }

  return {
    identity,
    step,
    stage,
    recoveryWords,
    call,
    confirmation,
    typing,
    deletion,
    actions: {
      startSetup,
      startRestore,
      showCheck,
      showWords,
      cancel: reset,
      setEntered,
      keep,
      restore,
    },
  }
}
