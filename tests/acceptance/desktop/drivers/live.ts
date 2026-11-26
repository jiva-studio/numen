import { isStackRequired, NO_STACK_MESSAGE, readStack } from './sync-stack.js'

/** The tags of the scenarios that reach real things. `@live` is on every one of them; `@model` is on those that also need a real model, which is skipped without its key. */
export const LIVE_TAG = '@live'
export const MODEL_TAG = '@model'
export const ENV_MODEL_KEY = 'NUMEN_ACCEPTANCE_TESTS_OPENROUTER_KEY'
export const NO_MODEL_KEY_MESSAGE = `no key of the real model: set ${ENV_MODEL_KEY}`

export const isModelKeySet = (env: Readonly<Record<string, string | undefined>> = process.env): boolean => (env[ENV_MODEL_KEY] ?? '') !== ''

/** Why a scenario of these tags cannot run here, or undefined when it can: a model scenario needs its key, any other @live scenario the real sync server (a run that requires the server fails in global-setup instead). */
export const readSkipReason = (tags: readonly string[]): string | undefined => {
  if (tags.includes(MODEL_TAG)) return isModelKeySet() ? undefined : NO_MODEL_KEY_MESSAGE
  if (tags.includes(LIVE_TAG) && readStack() === undefined && !isStackRequired()) return NO_STACK_MESSAGE
  return undefined
}
