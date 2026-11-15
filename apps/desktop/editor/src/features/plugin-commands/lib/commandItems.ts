/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and the run of one with the item picked.
 */
import {
  isSharedCommand,
  type CommandItem,
  type CommandNote,
  type CommandTarget as RunTarget,
  type PluginHost,
} from '@numen/plugin-loader'
import { type Command } from '@/entities/command'
import { getValues, type CommandTarget } from '@/shared/model/command'
import { getNamedTarget } from '@/entities/plugin'

type Values = Readonly<Record<string, string>>

interface CommandItemsDeps {
  readonly getAppCommands: () => readonly Command[]
  readonly pluginCommands: Pick<
    PluginHost['commands'],
    'hasCommandItems' | 'listCommandItems' | 'executeCommand'
  >
  readonly getTarget: () => CommandTarget
  readonly runApp: (
    id: string,
    target: CommandTarget,
    itemId: string,
    values?: Values,
  ) => Promise<void>
  readonly getCommandNote: (target: RunTarget) => Promise<CommandNote>
}

export function createCommandItems(deps: CommandItemsDeps) {
  const findAppCommand = (id: string) =>
    deps.getAppCommands().find((one) => one.id === id && one.listItems !== undefined)

  const toAppTarget = (target: RunTarget): CommandTarget => getNamedTarget(deps.getTarget(), target)

  /** The target with the type and properties of its note, which the handlers of a shared command read. */
  const readNoteTarget = async (target: RunTarget): Promise<RunTarget> => {
    const { type, properties } = await deps.getCommandNote(target)
    return { ...target, noteType: type, ...(properties ? { properties } : {}) }
  }

  const runPlugin = async (
    id: string,
    target: RunTarget,
    itemId: string,
    values?: Values,
  ): Promise<void> => {
    const note = await deps.getCommandNote(target)
    await deps.pluginCommands.executeCommand(id, note, {
      itemId,
      ...getValues(values),
      isAwaited: true,
    })
  }

  return {
    hasCommandItems: (id: string): boolean =>
      findAppCommand(id) !== undefined || deps.pluginCommands.hasCommandItems(id),
    listCommandItems: async (id: string, target: RunTarget): Promise<readonly CommandItem[]> => {
      const command = findAppCommand(id)
      if (command) return (await command.listItems?.(toAppTarget(target))) ?? []
      if (!deps.pluginCommands.hasCommandItems(id)) return []
      const noteTarget = isSharedCommand(id) ? await readNoteTarget(target) : target
      return deps.pluginCommands.listCommandItems(id, noteTarget)
    },
    runWithItem: (
      id: string,
      target: RunTarget,
      itemId: string,
      values?: Values,
    ): Promise<void> => {
      const extra = values === undefined ? [] : [values]
      return findAppCommand(id)
        ? deps.runApp(id, toAppTarget(target), itemId, ...extra)
        : runPlugin(id, target, itemId, ...extra)
    },
  }
}
