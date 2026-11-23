/**
 * The check of two sheets against the account of this device: the fields shown, and whether the
 * last check said the sheets fit.
 */
import { ref } from 'vue'
import { WORDS as words } from '../words'
import type { InstallationIdentity } from '../types'
import type { useIdentityCall } from './useIdentityCall'

interface SheetFields {
  readSheets(): string[][]
  clear(): void
}

export function useKitCheck(
  getIdentity: () => InstallationIdentity,
  call: ReturnType<typeof useIdentityCall>,
  fields: SheetFields,
) {
  const { problem, run } = call
  const isChecking = ref(false)
  const fits = ref(false)

  const reset = (): void => {
    isChecking.value = false
    fits.value = false
  }

  const forget = (): void => {
    fits.value = false
  }

  const open = (): void => {
    problem.value = ''
    fields.clear()
    fits.value = false
    isChecking.value = true
  }

  const cancel = (): void => {
    problem.value = ''
    fields.clear()
    reset()
  }

  const check = async (): Promise<void> => {
    const sheets = fields.readSheets()
    if (sheets.length === 0) {
      problem.value = words.kitOneMoreSheet
      return
    }
    const answer = await run(() => getIdentity().checkKit(sheets))
    fits.value = answer.ok
  }

  return { isChecking, fits, reset, forget, open, cancel, check }
}
