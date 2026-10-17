/**
 * The latest question wins.
 */
export interface Question {
  readonly isCurrent: boolean
  claim(): boolean
}

export type AnswerGuard = ReturnType<typeof createAnswerGuard>

export function createAnswerGuard() {
  let questionSeq = 0
  let renderedSeq = 0
  let isListening = true

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

  const drop = () => {
    questionSeq += 1
    renderedSeq = questionSeq
  }

  const close = () => {
    isListening = false
  }

  const open = () => isListening

  return { ask, drop, close, open }
}
