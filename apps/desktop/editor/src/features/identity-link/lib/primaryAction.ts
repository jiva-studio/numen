import { WORDS as words } from '../words'
import type { LinkView } from './phases'

/** The one step forward of a link dialog: its label and whether it can be pressed now. */
export interface PrimaryAction {
  label: string
  isDisabled: boolean
}

/** What the dialog knows around its view: a call is running, the field holds text, the link is pasted. */
export interface PrimaryState {
  isBusy: boolean
  hasInput: boolean
  isPasting: boolean
}

const LABELS: Partial<Record<LinkView, string>> = {
  peer: words.linkApprove,
  account: words.linkJoin,
  stalled: words.linkTryAgain,
  refused: words.linkTryAgain,
  failed: words.linkTryAgain,
}

export function getPrimaryAction(view: LinkView, state: PrimaryState): PrimaryAction | undefined {
  if (view === 'scan') {
    return state.isPasting
      ? { label: words.linkUse, isDisabled: state.isBusy || !state.hasInput }
      : undefined
  }
  const label = LABELS[view]
  if (label === undefined) return undefined
  return { label, isDisabled: state.isBusy }
}
