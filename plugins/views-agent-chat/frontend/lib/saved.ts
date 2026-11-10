/**
 * What an agent tab saves across restarts: the question being written and the turns that settled.
 * The agent itself does not keep the conversation, so the turns are shown and a new question
 * starts a new conversation with the agent.
 */
import type { JsonValue } from '@numen/plugin-sdk'
import type { Turn, Voice } from '../types'

/** One settled turn, as an agent tab saves it. */
type SavedTurn = {
  readonly id: string
  readonly voice: Voice
  readonly text: string
}

/** What an agent tab saves. */
export type SavedConversation = {
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

const isObject = (value: JsonValue | undefined): value is { readonly [key: string]: JsonValue } =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isSavedTurn = (value: JsonValue): value is SavedTurn =>
  isObject(value) &&
  typeof value['id'] === 'string' &&
  typeof value['text'] === 'string' &&
  KEPT.has(value['voice'] as Voice)

/** The conversation an agent tab saved, and an empty one where the value is not one. */
export const parseSavedConversation = (saved: JsonValue | undefined): SavedConversation => {
  const value = isObject(saved) ? saved : {}
  const turns = value['turns']
  const question = value['question']
  return {
    question: typeof question === 'string' ? question : '',
    turns: Array.isArray(turns) ? turns.filter(isSavedTurn) : [],
  }
}
