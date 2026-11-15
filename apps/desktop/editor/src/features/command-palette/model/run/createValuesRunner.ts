import { runInBackground } from '@numen/ui'
import {
  withValues,
  type CommandDeps,
  type CommandTarget,
  type Values,
} from '@/shared/model/command'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { createInvocation } from '../../lib/entries/invocation'
import { runInvocation } from '../handlers/handlers'

/**
 * Runs a command of the window over a target with the values of a keybinding. The command is not
 * asked whether it is offered: the reveal of a note is offered only as the rows that carry its values,
 * and the condition of the key was judged when the key matched.
 */
export const createValuesRunner =
  (deps: CommandDeps, writeMessage: MessageWriter) =>
  (command: string, target: CommandTarget, values: Values): void =>
    runInBackground(
      runInvocation(withValues(createInvocation(command, target), values), deps),
      (error) => writeMessage(formatError(error), 'error'),
    )
