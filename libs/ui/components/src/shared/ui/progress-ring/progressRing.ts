/** The ring is drawn on a path of length 100, so the filled share of it is a percent. */
export const RING_LENGTH = 100
const ARC_LENGTH = 25

export interface RingView {
  readonly isKnown: boolean
  readonly percent: number
  readonly dash: string
}

/** What the ring draws for a progress from 0 to 1; an undefined or invalid progress is a short arc. */
export function getRingView(value: number | undefined): RingView {
  if (value === undefined || !Number.isFinite(value)) {
    return { isKnown: false, percent: 0, dash: `${ARC_LENGTH} ${RING_LENGTH - ARC_LENGTH}` }
  }
  const share = Math.min(1, Math.max(0, value))
  const percent = Math.round(share * RING_LENGTH)
  return { isKnown: true, percent, dash: `${percent} ${RING_LENGTH}` }
}
