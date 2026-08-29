/**
 * What one face of a card comes to as it is read: the halves it draws, and the
 * words it is drawn with. No DOM, no measurement, no clock.
 */

import type { Half } from './order'

/** The words a card is drawn with, declared once. */
export interface FaceWords {
  /** What is said in place of a half with nothing in it. */
  readonly silence: string
  /** What the button turning the card says. */
  readonly turning: string
}

export const FACE_WORDS: FaceWords = {
  silence: 'Nothing here',
  turning: 'Turn',
}

/** One half of a card as it is drawn. */
export interface Part {
  readonly half: Half
  /** Markdown, assembled already. */
  readonly text: string
  /** Nothing but space stands in it. */
  readonly blank: boolean
}

/**
 * What a face draws, in the order it draws it. The back stands once the card
 * is turned and not before.
 */
export function parts(front: string, back: string, turned: boolean): readonly Part[] {
  const part = (half: Half, text: string): Part => ({
    half,
    text,
    blank: text.trim() === '',
  })
  return turned ? [part('front', front), part('back', back)] : [part('front', front)]
}
