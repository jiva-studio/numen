/**
 * A step of a command: what it asks for, and the answers it is reached by. The
 * names below are what an item of a step is chosen under, so the view and the
 * navigation agree on one word for one answer.
 */
import type { Command, CommandPrompt, CommandTarget, PromptStep } from '../../types'

/**
 * One step of a command: what it asks for, and what it is over. An `arguments`
 * step asks for the prompt at `promptIndex` and holds the `values` of the earlier
 * prompts.
 */
export interface PendingStep {
  readonly step: PromptStep
  readonly command: Command
  readonly on: CommandTarget
  readonly promptIndex?: number
  readonly values?: Readonly<Record<string, string>>
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

/** The prompt an `arguments` step asks for, and nothing for any other step. */
export const getPrompt = (step: PendingStep): CommandPrompt | undefined =>
  step.step === 'arguments' ? step.command.prompts?.[step.promptIndex ?? 0] : undefined
