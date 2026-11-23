/**
 * The state of the identity section: showing the recovery words of a new identity and checking
 * three of them in two steps, opening a restore, and deleting the identity from this device. The
 * words live in this state only while a setup is open; leaving the page or cancelling drops them,
 * and a kept identity hands them on to the recovery kit once.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'
import { useIdentityDelete } from './useIdentityDelete'
import type { useIdentityCall } from './useIdentityCall'
import { useIdentityRestore } from './useIdentityRestore'
import { useWordsConfirmation } from './useWordsConfirmation'
import { useSetupStage } from './useSetupStage'

type Step = 'idle' | 'setup' | 'restore' | 'restoreSheets'

interface SheetFields {
  setText(index: number, value: string): void
  readSheets(): string[][]
  clear(): void
}

export function useIdentitySection(
  getIdentity: () => InstallationIdentity,
  call: ReturnType<typeof useIdentityCall>,
  fields: SheetFields,
  onKept: (recoveryWords: readonly string[]) => Promise<void>,
) {
  const identity = computed(() => getIdentity().identity.value)
  const step = ref<Step>('idle')
  const { problem, isBusy, run } = call
  const setup = useSetupStage(problem)
  const { stage, recoveryWords } = setup
  const confirmation = useWordsConfirmation(recoveryWords)
  const restoring = useIdentityRestore(getIdentity, call, fields, reset)
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

  const startRestore = (kind: 'restore' | 'restoreSheets'): void => {
    reset()
    step.value = kind
  }

  const setEntered = (position: number, value: string): void => {
    confirmation.setEntered(position, value)
    problem.value = ''
  }

  const keep = async (): Promise<void> => {
    if (isBusy.value) return
    if (!confirmation.isConfirmed()) {
      confirmation.markWrong()
      problem.value = words.identityNotValid
      return
    }
    const kept = recoveryWords.value
    const answer = await run(() => getIdentity().create(kept))
    if (!answer.ok) return
    reset()
    await onKept(kept)
  }

  return {
    identity,
    step,
    setup,
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
    },
  }
}
