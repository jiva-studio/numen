/**
 * What an agent tab saves across restarts: the question being written, the turns that settled, and the
 * connection and the model the conversation chose. The agent itself does not keep the conversation, so
 * the turns are shown and a new question starts a new conversation with the agent.
 */
import type { JsonValue } from '@numen/plugin-sdk'
import type { Turn, Voice } from '../types'

/** One settled turn, as an agent tab saves it. */
type SavedTurn = {
  readonly id: string
  readonly voice: Voice
  readonly text: string
}

/** The connection of the agent and the model a conversation chose. */
export type SavedModel = {
  readonly connectionId: string
  readonly modelId: string
}

/** What an agent tab saves. `model` is null for a conversation that chose none. */
export type SavedConversation = {
  readonly question: string
  readonly turns: readonly SavedTurn[]
  readonly model: SavedModel | null
}

/** The voices a saved conversation keeps; the work lines of an answer are not kept. */
const KEPT: ReadonlySet<Voice> = new Set<Voice>(['prompt', 'response'])

/** The state an agent tab saves, from its question, its turns and the model it chose. */
export const createSavedConversation = (
  question: string,
  turns: readonly Turn[],
  model: SavedModel | null = null,
): SavedConversation => ({
  question,
  turns: turns
    .filter((turn) => KEPT.has(turn.voice) && (turn.state ?? 'settled') === 'settled')
    .map(({ id, voice, text }) => ({ id, voice, text })),
  model: model && { connectionId: model.connectionId, modelId: model.modelId },
})

const isObject = (value: JsonValue | undefined): value is { readonly [key: string]: JsonValue } =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isSavedTurn = (value: JsonValue): value is SavedTurn =>
  isObject(value) &&
  typeof value['id'] === 'string' &&
  typeof value['text'] === 'string' &&
  KEPT.has(value['voice'] as Voice)

const parseSavedModel = (value: JsonValue | undefined): SavedModel | null => {
  if (!isObject(value)) return null
  const connectionId = value['connectionId']
  const modelId = value['modelId']
  if (typeof connectionId !== 'string' || !connectionId || typeof modelId !== 'string') return null
  return { connectionId, modelId }
}

/** The conversation an agent tab saved, and an empty one where the value is not one. */
export const parseSavedConversation = (saved: JsonValue | undefined): SavedConversation => {
  const value = isObject(saved) ? saved : {}
  const turns = value['turns']
  const question = value['question']
  return {
    question: typeof question === 'string' ? question : '',
    turns: Array.isArray(turns) ? turns.filter(isSavedTurn) : [],
    model: parseSavedModel(value['model']),
  }
}
