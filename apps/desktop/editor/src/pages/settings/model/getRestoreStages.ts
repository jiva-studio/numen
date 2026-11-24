import type { Stage, StageState } from '@numen/ui'
import { WORDS as words } from '../words'

/**
 * The three stages of the restore from sheets, always in this order. After a failed restore the sheets
 * are pending again, because the next scan starts over.
 */
export function getRestoreStages(
  slots: readonly (number | undefined)[],
  isBusy: boolean,
  hasFailed: boolean,
): Stage[] {
  const sheets = hasFailed ? [undefined, undefined] : slots
  const next = sheets.findIndex((sheet) => sheet === undefined)
  const getStageState = (at: number): StageState => {
    if (sheets[at] !== undefined) return 'done'
    return at === next ? 'active' : 'pending'
  }
  const sheetStages = [0, 1].map((at) => {
    const sheet = sheets[at]
    return {
      id: `sheet-${at + 1}`,
      label: words.restoreSheetSlot(at + 1),
      state: getStageState(at),
      ...(sheet === undefined ? {} : { detail: words.restoreSheetNumber(sheet) }),
    }
  })
  const restoreState: StageState = isBusy ? 'active' : 'pending'
  const restore = { id: 'restore', label: words.restoreAction, state: restoreState }
  return [...sheetStages, isBusy ? { ...restore, detail: words.restoringNow } : restore]
}
