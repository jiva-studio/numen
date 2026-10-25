/**
 * The commands the enabled plugins declare in their manifests, which the
 * plugins' backends run. The application checks a run against the declaration
 * before it reaches the plugin.
 */
import {
  CommandArgumentType,
  CommandErrorCode,
  type PluginCommand as WireCommand,
} from '@numen/protocol'
import { pluginCommands } from '@/shared/clients'

export type DeclaredArgumentType = 'string' | 'number' | 'boolean'

export interface DeclaredArgument {
  readonly name: string
  readonly title: string
  readonly type: DeclaredArgumentType
  readonly isRequired: boolean
}

/** One command a plugin declares. A command with no `fileExtensions` runs over no file. */
export interface DeclaredCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly description: string
  readonly fileExtensions: readonly string[]
  readonly arguments: readonly DeclaredArgument[]
}

/** One run of a command: the file it is over, and the values of its arguments by name. */
export interface DeclaredCommandRun {
  readonly path: string
  readonly values: Readonly<Record<string, string>>
}

/** What a run came to: what the plugin said, or why it did not succeed. */
export type DeclaredCommandOutcome = { ok: true; value: string } | { ok: false; error: string }

const ARGUMENT_TYPES: Record<CommandArgumentType, DeclaredArgumentType> = {
  [CommandArgumentType.UNSPECIFIED]: 'string',
  [CommandArgumentType.STRING]: 'string',
  [CommandArgumentType.NUMBER]: 'number',
  [CommandArgumentType.BOOLEAN]: 'boolean',
}

const readCommand = (wire: WireCommand): DeclaredCommand => ({
  pluginId: wire.pluginId,
  id: wire.id,
  title: wire.title,
  description: wire.description,
  fileExtensions: wire.fileExtensions,
  arguments: wire.arguments.map((one) => ({
    name: one.name,
    title: one.title,
    type: ARGUMENT_TYPES[one.type],
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
    path: command.fileExtensions.length > 0 ? run.path : '',
    arguments: { ...run.values },
  })
  if (answer.error === CommandErrorCode.UNSPECIFIED) return { ok: true, value: answer.message }
  return { ok: false, error: answer.message }
}

/** Whether the command runs over the file at the path, by its extension. */
export function canRunOn(command: DeclaredCommand, path: string): boolean {
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dot = name.lastIndexOf('.')
  if (dot < 0) return false
  return command.fileExtensions.includes(name.slice(dot + 1).toLowerCase())
}
