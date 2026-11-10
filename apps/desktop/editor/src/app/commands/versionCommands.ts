/**
 * The command of the window over the text versions of a file: Delete text, which offers every version
 * of the file the core lists and has the core delete the one picked. The core serves it with no plugin.
 */
import type { CommandItem } from '@numen/plugin-loader'
import { registerCommand } from '@/features/command-palette'
import type { Command, CommandInvocation, CommandTarget } from '@/entities/command'
import type { DeclaredCommandOutcome } from '@/entities/plugin'
import { COMMAND } from '@/shared/commandIds'

/** The plugin id the core serves the commands of the application under: none. */
const CORE_PLUGIN = ''

const VERSION_WORDS = { deleteText: 'Delete text' }

export interface VersionCommandsDeps {
  /** The items of a command the core serves, over the file. */
  listItems(
    command: { readonly pluginId: string; readonly id: string },
    path: string,
  ): Promise<readonly CommandItem[]>
  /** Runs a command the core serves over the file, with the item picked. */
  run(command: string, path: string, itemId: string): Promise<DeclaredCommandOutcome>
}

const getFile = (target: Pick<CommandTarget, 'file' | 'path'>): string => target.file || target.path

export function createVersionCommands(deps: VersionCommandsDeps) {
  const list: readonly Command[] = [
    {
      id: COMMAND.deleteText,
      text: VERSION_WORDS.deleteText,
      group: 'file',
      input: 'items',
      isAvailable: (target) => getFile(target) !== '' && target.isFolder !== true,
      listItems: (target) =>
        deps.listItems({ pluginId: CORE_PLUGIN, id: COMMAND.deleteText }, getFile(target)),
    },
  ]

  /** Registers the run of the command; the result takes it back. */
  const register = (): (() => void) =>
    registerCommand(COMMAND.deleteText, async (invocation: CommandInvocation, on) => {
      if (invocation.name === '') return
      const outcome = await deps.run(COMMAND.deleteText, getFile(invocation), invocation.name)
      if (!outcome.ok) on.writeMessage(outcome.error, 'error')
    })

  return { commands: list, register }
}
