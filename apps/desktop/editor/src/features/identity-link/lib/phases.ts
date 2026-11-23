import type { LinkInfo, LinkPhase } from '@/entities/identity'

const ENDED: ReadonlySet<LinkPhase> = new Set(['linked', 'failed'])

/** Whether the link is over, so nothing more is read from the core. */
export const isEnded = (link: LinkInfo): boolean => ENDED.has(link.phase)
