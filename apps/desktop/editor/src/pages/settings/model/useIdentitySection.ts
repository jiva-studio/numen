/**
 * The state of the identity section: showing the recovery words of a new identity and checking
 * three of them in two steps, restoring from the twelve words, from two scanned sheets or from two typed sheets in a dialog of its own,
 * and deleting the identity from this device. The words live in this state only while a setup is
 * open; leaving the page or cancelling drops them, and a kept identity hands them to `onKept` once.
 */
import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'
import { useIdentityDelete } from './useIdentityDelete'
import { useIdentityCall } from './useIdentityCall'
import { useRestoreSources } from './useRestoreSources'
import { useWordsConfirmation } from './useWordsConfirmation'
import { useSetupStage } from './useSetupStage'

type Step = 'idle' | 'setup' | 'restore-words' | 'restore-sheets' | 'restore-typed'

function useLoadOnMount(getIdentity: () => InstallationIdentity, problem: Ref<string>): void {
  onMounted(() => {
    runInBackground(getIdentity().load(), (caughtError) => {
      problem.value = formatError(caughtError)
    })
  })
}

export function useIdentitySection(
  getIdentity: () => InstallationIdentity,
  onKept: (recoveryWords: readonly string[]) => void,
) {
  const identity = computed(() => getIdentity().identity.value)
  const step = ref<Step>('idle')
  const call = useIdentityCall()
  const { problem, isBusy, run } = call
  const setup = useSetupStage(problem)
  const confirmation = useWordsConfirmation(setup.recoveryWords)
  const deletion = useIdentityDelete(getIdentity, run, () => isBusy.value)
  const restore = useRestoreSources({
    getIdentity,
    run,
    problem,
    isBusy: () => isBusy.value,
    onRestored: () => reset(),
  })

  useLoadOnMount(getIdentity, problem)
  onBeforeUnmount(reset)

  let generation = 0

  function reset(): void {
    generation += 1
    step.value = 'idle'
    setup.clear()
    confirmation.clear()
    restore.clear()
    problem.value = ''
    deletion.cancelDelete()
  }

  const startSetup = async (): Promise<void> => {
    if (isBusy.value) return
    const started = generation
    const answer = await run(() => getIdentity().recovery.generateWords())
    if (!answer.ok || started !== generation) return
    setup.recoveryWords.value = answer.value
    confirmation.pick()
    setup.stage.value = 'words'
    step.value = 'setup'
  }

  const startRestore = (kind: 'restore-words' | 'restore-sheets' | 'restore-typed'): void => {
    reset()
    step.value = kind
  }

  const setEntered = (position: number, value: string): void => {
    confirmation.setEntered(position, value)
    problem.value = ''
  }

  const keep = async (): Promise<void> => {
    if (isBusy.value) return
    if (!confirmation.check()) {
      problem.value = words.identityNotValid
      return
    }
    const kept = setup.recoveryWords.value
    const answer = await run(() => getIdentity().create(kept))
    if (!answer.ok) return
    reset()
    onKept(kept)
  }

  return {
    identity,
    step,
    setup,
    call,
    confirmation,
    restore,
    deletion,
    actions: {
      startSetup,
      startRestore,
      cancel: reset,
      setEntered,
      keep,
      restore: restore.restoreWords,
      restoreTyped: restore.restoreTyped,
    },
  }
}
