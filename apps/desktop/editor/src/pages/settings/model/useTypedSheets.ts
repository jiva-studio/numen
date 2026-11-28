/**
 * Two sheets of a recovery kit typed, the words of one sheet in each field, read as the person types.
 */
import { computed, ref, type Ref } from 'vue'
import type { KitProof } from '@/entities/identity'
import { readTypedSheets } from './readProof'

export function useTypedSheets(problem: Ref<string>) {
  const sheets = ref<[string, string]>(['', ''])
  const reading = computed(() => readTypedSheets(sheets.value[0], sheets.value[1]))

  const setSheet = (position: 0 | 1, value: string): void => {
    sheets.value = position === 0 ? [value, sheets.value[1]] : [sheets.value[0], value]
    problem.value = ''
  }

  const clear = (): void => {
    sheets.value = ['', '']
  }

  const read = (): KitProof | undefined => reading.value.proof

  return { sheets, reading, setSheet, clear, read }
}
