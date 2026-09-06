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
  readonly current: boolean
  /**
   * Whether this answer may be drawn over what is drawn already. An answer
   * older than one that has landed is let go of.
   */
  lands(): boolean
}

export type AnswerGuard = ReturnType<typeof answerGuard>

export function answerGuard() {
  let asked = 0
  let landed = 0
  let listening = true

  /** A turn for one question. What is already on its way is let go of. */
  const ask = (): Question => {
    const mine = ++asked
    return {
      get current() {
        return listening && mine === asked
      },
      lands() {
        if (!listening || mine < landed) return false
        landed = mine
        return true
      },
    }
  }

  /** Nothing already asked for will be drawn. */
  const drop = () => {
    asked += 1
    landed = asked
  }

  /** Nothing will be drawn from here on, whenever it lands. */
  const close = () => {
    listening = false
  }

  /** Whether answers are still being drawn. */
  const open = () => listening

  return { ask, drop, close, open }
}
