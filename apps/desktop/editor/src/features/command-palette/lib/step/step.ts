/**
 * A step of a command: the input it requires. The step reads its own kind; the rows it
 * draws carry the command as their action.
 */
import type { Command, CommandPrompt, PromptStep } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'

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

/** The prompt an `arguments` step requires; undefined for any other step. */
export const getPrompt = (step: PendingStep): CommandPrompt | undefined =>
  step.step === 'arguments' ? step.command.prompts?.[step.promptIndex ?? 0] : undefined
