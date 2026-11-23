/**
 * The restore of an identity on a device without one: from the 12 words typed by the person, or
 * from two typed sheets of a recovery kit.
 */
import { WORDS as words } from '../words'
import type { InstallationIdentity } from '../types'
import type { useIdentityCall } from './useIdentityCall'
import { splitSheetText } from './useRecoveryKit'
import { useRestoreText } from './useRestoreText'

interface SheetFields {
  setText(index: number, value: string): void
  readSheets(): string[][]
  clear(): void
}

export function useIdentityRestore(
  getIdentity: () => InstallationIdentity,
  call: ReturnType<typeof useIdentityCall>,
  fields: SheetFields,
  onRestored: () => void,
) {
  const { problem, isBusy, run } = call
  const typing = useRestoreText(problem)

  const clear = (): void => {
    typing.restoreText.value = ''
    fields.clear()
  }

  const setSheetText = (index: number, value: string): void => {
    fields.setText(index, value)
    problem.value = ''
  }

  const restoreWords = async (): Promise<void> => {
    if (isBusy.value) return
    const typed = splitSheetText(typing.restoreText.value)
    const answer = await run(() => getIdentity().restore(typed))
    if (answer.ok) onRestored()
  }

  const restoreSheets = async (): Promise<void> => {
    if (isBusy.value) return
    const sheets = fields.readSheets()
    if (sheets.length === 0) {
      problem.value = words.kitOneMoreSheet
      return
    }
    const answer = await run(() => getIdentity().restoreFromKit(sheets))
    if (answer.ok) onRestored()
  }

  return { typing, clear, setSheetText, restoreWords, restoreSheets }
}
