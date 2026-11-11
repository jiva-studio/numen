/**
 * How a command of the window is run: from the palette and the keys, or to its end for a plugin that
 * awaits it.
 */
import { runInBackground } from '@numen/ui'
import {
  getValues,
  withValues,
  type CommandDeps,
  type CommandTarget,
  type Values,
} from '@/shared/model/command'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { Commands } from '../palette/palette'
import { runInvocation } from '../handlers/handlers'
import { runInvocationToEnd } from './runInvocationToEnd'

/** The commands a plugin declares, which the plugin host runs. */
interface PluginCommandPort {
  isPluginCommand(id: string): boolean
  run(
    id: string,
    target: CommandTarget,
    way: { readonly values?: Values; readonly isAwaited?: true },
  ): Promise<string | boolean | void>
}

/** What the runs of the window go through. */
export interface CommandRunnerParts {
  readonly commands: Pick<Commands, 'palette'>
  readonly closeSearch: () => void
  readonly pluginCommands: PluginCommandPort
  readonly getCommandDeps: () => CommandDeps
  readonly writeMessage: MessageWriter
}

/** How a command is run: with the text typed for it, and whether its caller awaits it. */
interface RunWay {
  readonly input?: string | undefined
  readonly values?: Values | undefined
  readonly isAwaited: boolean
}

/**
 * The two runs of a command of the window. The run of the palette and the shortcuts writes its
 * error in the window. The run to its end, which a plugin awaits through `context.commands.run`,
 * writes nothing: it settles when the run ends with the path of the note or folder it made, empty
 * where it made none, and rejects with its error.
 */
export function createCommandRunner(parts: CommandRunnerParts) {
  const { palette } = parts.commands

  /** The run of the command over the target, or the objection to it where it cannot start. */
  const startRun = (
    id: string,
    target: CommandTarget,
    way: RunWay,
  ): Promise<string | boolean | void> | string => {
    if (parts.pluginCommands.isPluginCommand(id)) {
      palette.setOpen(false)
      return parts.pluginCommands.run(id, target, {
        ...getValues(way.values),
        ...(way.isAwaited ? { isAwaited: true as const } : {}),
      })
    }
    const invocation = palette.startCommand(id, target, way.input, getValues(way.values).values)
    if (invocation) {
      const deps = parts.getCommandDeps()
      const run = withValues(invocation, way.values)
      return way.isAwaited ? runInvocationToEnd(run, deps) : runInvocation(run, deps)
    }
    if (palette.open.value) {
      parts.closeSearch()
      return Promise.resolve()
    }
    return palette.getObjection(id, target)
  }

  return {
    runCommand: (id: string, target: CommandTarget, input?: string, values?: Values): void => {
      const run = startRun(id, target, { input, values, isAwaited: false })
      if (typeof run === 'string') parts.writeMessage(run, 'error')
      else runInBackground(run, (error) => parts.writeMessage(formatError(error), 'error'))
    },
    runCommandToEnd: async (
      id: string,
      target: CommandTarget,
      values?: Values,
    ): Promise<string> => {
      const run = startRun(id, target, { values, isAwaited: true })
      if (typeof run === 'string') throw new Error(run)
      const made = await run
      return typeof made === 'string' ? made : ''
    },
  }
}
