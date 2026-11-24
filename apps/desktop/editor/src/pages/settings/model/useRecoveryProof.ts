/**
 * What proves the account in a dialog of the identity: one text read as the person types, by default
 * as the 12 recovery words or two sheets of a recovery kit one to a line. Typing clears the problem shown.
 */
import { computed, ref, type Ref } from 'vue'
import type { KitProof } from '@/entities/identity'
import { readProof, type ProofReading } from './readProof'

export function useRecoveryProof(
  problem: Ref<string>,
  readText: (text: string) => ProofReading = readProof,
) {
  const text = ref('')
  const reading = computed(() => readText(text.value))

  const setText = (value: string): void => {
    text.value = value
    problem.value = ''
  }

  const clear = (): void => {
    text.value = ''
  }

  const read = (): KitProof | undefined => reading.value.proof

  return { text, reading, setText, clear, read }
}
