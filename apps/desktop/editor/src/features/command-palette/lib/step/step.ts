/**
 * A step of a command: the input it requires, and the answers it accepts. The names
 * below identify an item of a step, so the view and the navigation agree on
 * one name per answer.
 */
import type { Command, CommandPrompt, CommandTarget, PromptStep } from '../../types'

/**
 * One step of a command: the input it requires, and what it applies to. An `arguments`
 * step requires the prompt at `promptIndex` and carries the `values` of the earlier
 * prompts.
 */
export interface PendingStep {
  readonly step: PromptStep
  readonly command: Command
  readonly on: CommandTarget
  readonly promptIndex?: number
  readonly values?: Readonly<Record<string, string>>
}

/** The item of a step that requires one value. */
export const NAME = 'name'
export const PICK = 'pick'
export const OPEN = 'open'
export const EXACT = 'exactly'

/** The single action available on every item of a list the window holds. */
export const CHOSEN = 'chosen'

/** The two answers of the step that confirms. */
export const NO = 'no'
export const YES = 'yes'

/** The prompt an `arguments` step requires; undefined for any other step. */
export const getPrompt = (step: PendingStep): CommandPrompt | undefined =>
  step.step === 'arguments' ? step.command.prompts?.[step.promptIndex ?? 0] : undefined
