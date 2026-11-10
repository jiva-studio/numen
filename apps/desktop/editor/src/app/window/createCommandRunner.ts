/**
 * How a command of the window is run: from the palette and the keys, or to its end for a plugin that
 * awaits it.
 */
import type { PluginHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import { runInvocation, type createCommandPalette } from '@/features/command-palette'
import type { createCommandNoteReader } from '@/features/plugin-commands'
import type { ResultChoice } from '@/features/command-palette'
import type { CommandDeps, CommandInvocation, CommandTarget } from '@/entities/command'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'

export type Values = Readonly<Record<string, string>>

export const getValues = (values: Values | undefined): { values?: Values } =>
  values && Object.keys(values).length > 0 ? { values } : {}

export const withValues = (
  invocation: CommandInvocation,
  values: Values | undefined,
): CommandInvocation => ({
  ...invocation,
  ...getValues(values),
})

export async function runInvocationToEnd(
  invocation: CommandInvocation,
  deps: CommandDeps,
): Promise<string> {
  const errors: string[] = []
  const writeMessage: MessageWriter = (text, severity) => {
    if (severity === 'error') errors.push(text)
    else deps.writeMessage(text, severity)
  }
  const made = await runInvocation(invocation, { ...deps, writeMessage })
  if (errors[0] !== undefined) throw new Error(errors[0])
  return made
}

/** What the runs of the window go through: the palettes, the plugin host and the deps of a command. */
interface CommandRunnerParts {
  readonly commands: ReturnType<typeof createCommandPalette>
  readonly palette: { setOpen(isOpen: boolean): void }
  readonly pluginHost: PluginHost | undefined
  readonly getCommandNote: ReturnType<typeof createCommandNoteReader>
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
  const { commands, palette, pluginHost, getCommandNote } = parts

  /** The run of the command over the target, or the objection to it where it cannot start. */
  const startRun = (id: string, target: CommandTarget, way: RunWay) => {
    if (pluginHost && (pluginHost.hasCommand(id) || pluginHost.isSharedCommand(id))) {
      commands.palette.setOpen(false)
      return getCommandNote(target).then((note) => {
        const run = { ...getValues(way.values), ...(way.isAwaited ? { isAwaited: true } : {}) }
        return Object.keys(run).length > 0
          ? pluginHost.executeCommand(id, note, run)
          : pluginHost.executeCommand(id, note)
      })
    }
    const { values } = getValues(way.values)
    const invocation = values
      ? commands.palette.startCommand(id, target, way.input, values)
      : way.input === undefined
        ? commands.palette.startCommand(id, target)
        : commands.palette.startCommand(id, target, way.input)
    if (invocation) {
      const deps = parts.getCommandDeps()
      const run = withValues(invocation, way.values)
      return way.isAwaited ? runInvocationToEnd(run, deps) : runInvocation(run, deps)
    }
    if (commands.palette.open.value) {
      palette.setOpen(false)
      return Promise.resolve()
    }
    return commands.palette.getObjection(id, target)
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

const toResultTarget = (place: NonNullable<ResultChoice['over']>) => ({
  tab: '',
  kind: null,
  path: place.path,
  file: place.file,
  title: place.title,
  ...(place.line === undefined ? {} : { line: place.line }),
  ...(place.selector === undefined ? {} : { selector: place.selector }),
})

/** Runs the command of a chosen action over what it was chosen on, with the text typed for it. */
export const createChoiceRunner =
  (
    runCommand: (id: string, target: CommandTarget, input?: string, values?: Values) => void,
    getTarget: () => CommandTarget,
  ) =>
  (choice: ResultChoice): void =>
    runCommand(
      choice.command,
      choice.over ? { ...getTarget(), ...toResultTarget(choice.over) } : getTarget(),
      choice.input || undefined,
      choice.values,
    )
