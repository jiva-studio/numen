/**
 * The latest question wins.
 *
 * Several answers can be on their way at once, and each question takes a turn.
 * A caller draws the answer to the turn last asked for, or draws answers in the
 * order they land and never goes back.
 */

/** One question asked, measured against everything asked after it. */
export interface Question {
  /** Whether the answer to this one is still the answer to draw. */
  readonly isCurrent: boolean
  /**
   * Whether this answer may be drawn over what is drawn already. An answer
   * older than one that has landed is let go of.
   */
  claim(): boolean
}

export type AnswerGuard = ReturnType<typeof createAnswerGuard>

export function createAnswerGuard() {
  let questionSeq = 0
  let renderedSeq = 0
  let isListening = true

  /** A turn for one question. What is already on its way is let go of. */
  const ask = (): Question => {
    const mine = ++questionSeq
    return {
      get isCurrent() {
        return isListening && mine === questionSeq
      },
      claim() {
        if (!isListening || mine < renderedSeq) return false
        renderedSeq = mine
        return true
      },
    }
  }

  /** Nothing already asked for will be drawn. */
  const drop = () => {
    questionSeq += 1
    renderedSeq = questionSeq
  }

  /** Nothing will be drawn from here on, whenever it lands. */
  const close = () => {
    isListening = false
  }

  /** Whether answers are still being drawn. */
  const open = () => isListening

  return { ask, drop, close, open }
}
