/**
 * The recovery kit on the identity section: the kit of three sheets while it is shown and the proof
 * asked for before a later kit. The kit and the proof live in this state only; closing the step or
 * leaving the page drops them.
 */
import { onBeforeUnmount, ref, shallowRef, type Ref } from 'vue'
import type { RecoveryKit } from '@/entities/identity'
import type { Printer } from '@/pages/settings/api/print'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'
import type { useIdentityCall } from './useIdentityCall'
import { useKitCheck } from './useKitCheck'

type Step = 'closed' | 'proof' | 'shown'

interface SheetFields {
  readonly texts: Readonly<Ref<readonly [string, string]>>
  setText(index: number, value: string): void
  readSheets(): string[][]
  clear(): void
}

/** The words of a sheet or of the recovery words as typed: whole words, lower-cased. */
export const splitSheetText = (text: string): string[] =>
  text
    .split(/\s+/)
    .filter((word) => word !== '')
    .map((word) => word.toLowerCase())

export function useRecoveryKit(
  getIdentity: () => InstallationIdentity,
  getPrinter: () => Printer,
  call: ReturnType<typeof useIdentityCall>,
  fields: SheetFields,
) {
  const { problem, run } = call
  const checking = useKitCheck(getIdentity, call, fields)
  const step = ref<Step>('closed')
  const kit = shallowRef<RecoveryKit | undefined>(undefined)
  const wordsText = ref('')
  const isFirst = ref(false)

  onBeforeUnmount(() => {
    close()
  })

  function close(): void {
    step.value = 'closed'
    kit.value = undefined
    wordsText.value = ''
    checking.reset()
    fields.clear()
    problem.value = ''
  }

  const show = (made: RecoveryKit, first: boolean): void => {
    close()
    kit.value = made
    isFirst.value = first
    step.value = 'shown'
  }

  const showAfterSetup = async (recoveryWords: readonly string[]): Promise<void> => {
    close()
    const answer = await run(() => getIdentity().createKit({ words: recoveryWords }))
    if (answer.ok) show(answer.value, true)
  }

  const openProof = (): void => {
    close()
    step.value = 'proof'
  }

  const makeKit = async (): Promise<void> => {
    const typed = splitSheetText(wordsText.value)
    const sheets = fields.readSheets()
    if (typed.length === 0 && sheets.length === 0) {
      problem.value = words.kitOneMoreSheet
      return
    }
    const proof = typed.length > 0 ? { words: typed } : { sheets }
    const answer = await run(() => getIdentity().createKit(proof))
    if (answer.ok) show(answer.value, false)
  }

  const print = (): void => {
    getPrinter().print()
  }

  const setWordsText = (value: string): void => {
    wordsText.value = value
    problem.value = ''
  }

  const setSheetText = (index: number, value: string): void => {
    fields.setText(index, value)
    problem.value = ''
    checking.forget()
  }

  return {
    step,
    kit,
    wordsText,
    isFirst,
    checking,
    actions: { showAfterSetup, openProof, makeKit, print, close },
    edit: { setWordsText, setSheetText },
  }
}
