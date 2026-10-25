import type { Notice } from '../lib/notice'

export interface NoticesProps {
  /** What the window has to say, in the order it is drawn. */
  notices?: readonly Notice[]
  /** What the corner is announced as. */
  name?: string
  /** What the way to dismiss one is called. */
  dismiss?: string
  /** What the ones folded away behind the rest are counted as. */
  more?: string
  /** How long work runs before it is worth a card. */
  wait?: number
  /** How many cards stand at once. */
  room?: number
  /** What the moment is. The window's clock by default. */
  clock?: () => number
  /** Whether nobody is looking. The window's own answer by default. */
  hidden?: () => boolean
}
