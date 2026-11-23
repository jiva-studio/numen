/**
 * What proves the account in a dialog of the identity: the twelve recovery words, one field each, or
 * two sheets of a recovery kit. Changing the choice or typing clears the problem shown.
 */
import { ref, type Ref } from 'vue'
import type { KitProof } from '@/entities/identity'
import { useRestoreWords } from './useRestoreWords'
import { useSheetFields } from './useSheetFields'

export type ProofKind = 'words' | 'sheets'

export function useRecoveryProof(problem: Ref<string>) {
  const kind = ref<ProofKind>('words')
  const words = useRestoreWords(problem)
  const sheets = useSheetFields()

  const setKind = (value: ProofKind): void => {
    kind.value = value
    problem.value = ''
  }

  const setSheet = (index: number, value: string): void => {
    sheets.setText(index, value)
    problem.value = ''
  }

  const clear = (): void => {
    kind.value = 'words'
    words.clear()
    sheets.clear()
  }

  /** The proof of the kind chosen, or undefined when no sheet holds a word. */
  const read = (): KitProof | undefined => {
    if (kind.value === 'words') return { words: words.readWords() }
    const typed = sheets.readSheets()
    return typed.length === 0 ? undefined : { sheets: typed }
  }

  return { kind, words, sheets, setKind, setSheet, clear, read }
}
