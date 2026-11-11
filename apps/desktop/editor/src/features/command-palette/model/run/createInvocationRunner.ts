import type { CommandDeps, CommandTarget } from '@/entities/command'
import { createInvocation } from '../../lib/entries/invocation'
import { runInvocation } from '../handlers/handlers'

/** Runs a command of the window over a target with the name typed for it, and settles when it ends. */
export const createInvocationRunner =
  (getDeps: () => CommandDeps) =>
  (id: string, target: CommandTarget, name: string): Promise<void> =>
    runInvocation(createInvocation(id, target, name), getDeps()).then(() => undefined)
