import type { LinkInfo, LinkPhase, LinkRole } from '@/entities/identity'

const ENDED: ReadonlySet<LinkPhase> = new Set(['linked', 'failed'])

/** Whether the link is over, so nothing more is read from the core. */
export const isEnded = (link: LinkInfo): boolean => ENDED.has(link.phase)

/** What the link dialog draws: one view for each state a person can be in. */
export type LinkView =
  | 'scan'
  | 'preparing'
  | 'stalled'
  | 'refused'
  | 'qr'
  | 'waiting'
  | 'peer'
  | 'code'
  | 'account'
  | 'linking'
  | 'linked'
  | 'failed'

/** What decides the view: the link the core holds, and what the window knows around it. */
export interface LinkViewState {
  link: LinkInfo | undefined
  role: LinkRole
  isScanning: boolean
  isStalled: boolean
  hasProblem: boolean
}

export function getLinkView(state: LinkViewState): LinkView {
  const { link } = state
  if (state.isStalled) return 'stalled'
  if (link === undefined) {
    if (state.isScanning) return 'scan'
    return state.hasProblem ? 'refused' : 'preparing'
  }
  switch (link.phase) {
    case 'waitingForChannel':
      return 'preparing'
    case 'waitingForPeer':
      return link.mode === 'show' ? 'qr' : 'waiting'
    case 'waitingForAccept':
      return 'waiting'
    case 'confirmPeer':
      return 'peer'
    case 'enterCode':
      return 'code'
    case 'confirmAccount':
      return state.role === 'joiner' ? 'account' : 'linking'
    case 'linked':
      return 'linked'
    case 'failed':
      return 'failed'
    default:
      return 'linking'
  }
}
