import { withValues, type CommandDeps, type CommandTarget, type Values } from '@/entities/command'
import { createInvocation } from '../../lib/entries/invocation'
import { runInvocationToEnd } from './runInvocationToEnd'

/** Runs a command of the window with the item picked for it, and settles when the run ends. */
export const createAppItemRunner =
  (deps: CommandDeps) =>
  async (id: string, target: CommandTarget, itemId: string, values?: Values): Promise<void> => {
    await runInvocationToEnd(withValues(createInvocation(id, target, itemId), values), deps)
  }
