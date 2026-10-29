/**
 * The commands the enabled plugins declare in their manifests, which the
 * plugins' backends run. The application checks a run against the declaration
 * before it is sent to the plugin.
 */
import { COMMAND_ERROR_CODE } from '@numen/plugin-sdk'
import { pluginCommands } from '@/shared/clients'

interface WireCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly contentTypes: readonly string[]
  readonly extensions: readonly string[]
  readonly arguments: readonly {
    readonly name: string
    readonly title: string
    readonly isRequired: boolean
  }[]
}

interface DeclaredArgument {
  readonly name: string
  readonly title: string
  readonly isRequired: boolean
}

/** The extension the core lists for a command over a file of any type. */
const ANY_FILE_EXTENSION = '*'

/** One command a plugin declares. A command with no `contentTypes` runs over no file. */
export interface DeclaredCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly contentTypes: readonly string[]
  /** The extensions, without the dot, that give a file one of the types, as the core lists them. */
  readonly extensions: readonly string[]
  readonly arguments: readonly DeclaredArgument[]
}

/** One run of a command: the file it is over, and the values of its arguments by name. */
export interface DeclaredCommandRun {
  readonly path: string
  readonly values: Readonly<Record<string, string>>
}

/**
 * The outcome of a run: the plugin's message and the file to open after the run,
 * either of them empty, or the reason the run failed.
 */
export type DeclaredCommandOutcome =
  | { ok: true; value: { readonly message: string; readonly openedPath: string } }
  | { ok: false; error: string }

const readCommand = (wire: WireCommand): DeclaredCommand => ({
  pluginId: wire.pluginId,
  id: wire.id,
  title: wire.title,
  ...(wire.icon ? { icon: wire.icon } : {}),
  contentTypes: wire.contentTypes,
  extensions: wire.extensions,
  arguments: wire.arguments.map((one) => ({
    name: one.name,
    title: one.title,
    isRequired: one.isRequired,
  })),
})

/** Every command the enabled plugins declare, in the order of the plugins' ids. */
export async function listDeclaredCommands(): Promise<readonly DeclaredCommand[]> {
  const answer = await pluginCommands.listCommands({})
  return answer.commands.map(readCommand)
}

/** Runs the command in the backend of the plugin that declares it. */
export async function runDeclaredCommand(
  command: DeclaredCommand,
  run: DeclaredCommandRun,
): Promise<DeclaredCommandOutcome> {
  const answer = await pluginCommands.runPluginCommand({
    pluginId: command.pluginId,
    command: command.id,
    path: command.contentTypes.length > 0 ? run.path : '',
    arguments: { ...run.values },
  })
  if (answer.error === COMMAND_ERROR_CODE.UNSPECIFIED) {
    return { ok: true, value: { message: answer.message, openedPath: answer.openedPath } }
  }
  return { ok: false, error: answer.message }
}

/** Whether the command runs over the file at the path, by its extension. */
export function canRunOn(command: DeclaredCommand, path: string): boolean {
  const name = path.slice(path.lastIndexOf('/') + 1)
  if (command.extensions.includes(ANY_FILE_EXTENSION)) return name !== ''
  const dot = name.lastIndexOf('.')
  if (dot < 0) return false
  return command.extensions.includes(name.slice(dot + 1).toLowerCase())
}
