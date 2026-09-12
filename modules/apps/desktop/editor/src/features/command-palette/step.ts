/**
 * A step of a command: what it asks for, and the answers it is reached by.
 *
 * The palette stands on a step, and what is drawn there is `view.ts`. The
 * names below are what an item of a step is chosen under, so the two agree on
 * one word for one answer.
 */
import type { Command, CommandTarget, PromptStep } from './target'

/** One step of a command: what it asks for, and what it is over. */
export interface PendingStep {
  readonly step: PromptStep
  readonly command: Command
  readonly on: CommandTarget
}

/** The one item of a step that asks for one thing. */
export const NAME = 'name'
export const PICK = 'pick'
export const OPEN = 'open'
export const EXACT = 'exactly'

/** The one thing every item of a list the window holds can be asked. */
export const CHOSEN = 'chosen'

/** The two answers of the step that confirms. */
export const NO = 'no'
export const YES = 'yes'
