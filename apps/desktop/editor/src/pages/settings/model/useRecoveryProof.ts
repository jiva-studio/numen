/**
 * What proves the account in a dialog of the identity: one text, the 12 recovery words or two sheets
 * of a recovery kit one to a line, read as the person types. Typing clears the problem shown.
 */
import { computed, ref, type Ref } from 'vue'
import type { KitProof } from '@/entities/identity'
import { readProof } from './readProof'

export function useRecoveryProof(problem: Ref<string>) {
  const text = ref('')
  const reading = computed(() => readProof(text.value))

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
