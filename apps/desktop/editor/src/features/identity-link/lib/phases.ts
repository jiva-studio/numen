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

const PHASE_VIEWS: Partial<Record<LinkPhase, LinkView>> = {
  waitingForChannel: 'preparing',
  waitingForAccept: 'waiting',
  confirmPeer: 'peer',
  linked: 'linked',
  failed: 'failed',
}

function getViewOfLink(link: LinkInfo, role: LinkRole): LinkView {
  if (link.phase === 'waitingForPeer') return link.mode === 'show' ? 'qr' : 'waiting'
  if (link.phase === 'confirmAccount') return role === 'joiner' ? 'account' : 'linking'
  return PHASE_VIEWS[link.phase] ?? 'linking'
}

export function getLinkView(state: LinkViewState): LinkView {
  if (state.isStalled) return 'stalled'
  if (state.link !== undefined) return getViewOfLink(state.link, state.role)
  if (state.isScanning) return 'scan'
  return state.hasProblem ? 'refused' : 'preparing'
}
