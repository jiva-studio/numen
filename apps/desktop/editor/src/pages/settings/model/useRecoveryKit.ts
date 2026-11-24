/**
 * The recovery kit of the identity of this device: its download as a PDF. Right after the identity is
 * created its words are held here, so the first download asks for nothing; any other download asks
 * for the 12 words or two sheets of an earlier kit in a dialog. The core makes the sheets and saves
 * them; none of them reaches this state. Leaving the page drops the held words and the proof.
 */
import { onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import type { KitProof } from '@/entities/identity'
import type { InstallationIdentity } from '../types'
import { useIdentityCall } from './useIdentityCall'
import { useRecoveryProof } from './useRecoveryProof'

export function useRecoveryKit(getIdentity: () => InstallationIdentity) {
  const call = useIdentityCall()
  const { problem, isBusy, run } = call
  const proof = useRecoveryProof(problem)
  const isAsking = ref(false)
  const isSaved = ref(false)
  const held = shallowRef<readonly string[]>([])

  watch(
    () => getIdentity().identity.value?.deviceFingerprint,
    () => {
      held.value = []
    },
    { flush: 'sync' },
  )

  onBeforeUnmount(() => {
    held.value = []
    close()
  })

  function close(): void {
    isAsking.value = false
    proof.clear()
    problem.value = ''
  }

  /** Holds the words of an identity just created for the first download. */
  const hold = (recoveryWords: readonly string[]): void => {
    held.value = recoveryWords
  }

  const save = async (given: KitProof): Promise<void> => {
    isSaved.value = false
    const answer = await run(() => getIdentity().saveKit(given))
    if (!answer.ok) return
    isSaved.value = answer.value
    if (!answer.value) return
    held.value = []
    close()
  }

  const download = async (): Promise<void> => {
    if (isBusy.value) return
    if (held.value.length > 0) {
      await save({ words: held.value })
      return
    }
    close()
    isSaved.value = false
    isAsking.value = true
  }

  const submit = async (): Promise<void> => {
    if (isBusy.value) return
    const given = proof.read()
    if (given !== undefined) await save(given)
  }

  return {
    call,
    proof,
    isAsking,
    isSaved,
    actions: { hold, download, submit, close },
  }
}
