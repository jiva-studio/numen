/**
 * What an agent tab saves across restarts: the question being written and the turns that settled.
 * The agent itself does not keep the conversation, so the turns are shown and a new question
 * starts a new conversation with the agent.
 */
import type { Turn, Voice } from '../thread/lib/turn'

/** One settled turn, as an agent tab saves it. */
export interface SavedTurn {
  readonly id: string
  readonly voice: Voice
  readonly text: string
}

/** What an agent tab saves. */
export interface SavedConversation {
  readonly question: string
  readonly turns: readonly SavedTurn[]
}

/** The voices a saved conversation keeps; the work lines of an answer are not kept. */
const KEPT: ReadonlySet<Voice> = new Set<Voice>(['prompt', 'response'])

/** The state an agent tab saves, from its question and its turns. */
export const createSavedConversation = (
  question: string,
  turns: readonly Turn[],
): SavedConversation => ({
  question,
  turns: turns
    .filter((turn) => KEPT.has(turn.voice) && (turn.state ?? 'settled') === 'settled')
    .map(({ id, voice, text }) => ({ id, voice, text })),
})

/** The conversation an agent tab saved, and an empty one where the value is not one. */
export const parseSavedConversation = (saved: unknown): SavedConversation => {
  const value = saved as Partial<SavedConversation> | null | undefined
  const turns = Array.isArray(value?.turns) ? value.turns : []
  return {
    question: typeof value?.question === 'string' ? value.question : '',
    turns: turns.filter(
      (turn): turn is SavedTurn =>
        typeof turn?.id === 'string' &&
        typeof turn.text === 'string' &&
        KEPT.has(turn.voice as Voice),
    ),
  }
}
