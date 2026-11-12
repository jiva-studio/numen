/**
 * The command of the window over the text versions of a file: Delete text, which offers every version
 * of the file the core lists and has the core delete the one picked. The core serves it with no plugin.
 */
import type { CommandItem } from '@numen/plugin-loader'
import { type Command, type FeatureCommands } from '@/entities/command'
import { COMMAND, type CommandInvocation, type CommandTarget } from '@/shared/model/command'
import { runDeclaredHandler } from '@/entities/plugin'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { VERSION_WORDS } from '../words'

/** The plugin id the core serves the commands of the application under: none. */
const CORE_PLUGIN = ''

export interface VersionCommandsDeps {
  /** The items of a command the core serves, over the file. */
  listItems(
    command: { readonly pluginId: string; readonly id: string },
    path: string,
  ): Promise<readonly CommandItem[]>
  readonly writeMessage: MessageWriter
}

const getFile = (target: Pick<CommandTarget, 'file' | 'path'>): string => target.file || target.path

const deleteText = async (
  invocation: CommandInvocation,
  deps: VersionCommandsDeps,
): Promise<void> => {
  if (invocation.name === '') return
  const outcome = await runDeclaredHandler(
    { pluginId: CORE_PLUGIN, command: COMMAND.deleteText },
    { path: getFile(invocation.target), values: {}, itemId: invocation.name },
  )
  if (!outcome.ok) deps.writeMessage(outcome.error, 'error')
}

const createRows = (deps: VersionCommandsDeps): readonly Command[] => [
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

/** The commands over the text versions of a file. */
export const createVersionCommands = (deps: VersionCommandsDeps): FeatureCommands => {
  const handlers: FeatureCommands['handlers'] = {
    [COMMAND.deleteText]: (invocation) => deleteText(invocation, deps),
  }
  return { commands: createRows(deps), handlers }
}
