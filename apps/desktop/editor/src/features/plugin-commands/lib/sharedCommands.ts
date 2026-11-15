/**
 * The shared commands in the palette: one entry for each shared command an enabled plugin handles. A
 * command that declares required arguments asks for each in a step of its own, as a command of a
 * manifest does; any other runs over a file, and its handlers are its items.
 */
import {
  SHARED_COMMANDS,
  type HandlerDeclaration,
  type PluginPaletteCommand,
} from '@numen/plugin-loader'
import type { Command, CommandPrompt } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'

type SharedCommand = (typeof SHARED_COMMANDS)[number]

const isOverFile = (target: CommandTarget): boolean =>
  target.isReady && (target.file || target.path) !== ''

const getSharedPrompts = (shared: SharedCommand): readonly CommandPrompt[] =>
  shared.arguments
    .filter((one) => one.isRequired)
    .map(({ name, title, isRequired }) => ({ name, title, isRequired }))

export function createSharedCommands(
  getHandlers: () => readonly HandlerDeclaration[] | undefined,
  isWhenAvailable: (when: string | undefined, target: CommandTarget) => boolean,
  sharedCommands: readonly SharedCommand[] = SHARED_COMMANDS,
) {
  const getHandledShared = (): readonly SharedCommand[] => {
    const handlers = getHandlers()
    if (!handlers) return []
    return sharedCommands.filter((shared) => handlers.some((one) => one.command === shared.id))
  }

  /** The shared commands an enabled plugin handles. */
  const getSharedCommands = (): readonly PluginPaletteCommand[] =>
    getHandledShared().map(({ id, title }) => ({ id, title }))

  /** Whether a handler of the command is available over the target. */
  const isHandled = (id: string, target: CommandTarget): boolean => {
    const handlers = getHandlers()
    return (
      handlers !== undefined &&
      handlers.some((one) => one.command === id && isWhenAvailable(one.when, target))
    )
  }

  /** A shared command as the palette lists it. */
  const createSharedEntry = (shared: SharedCommand): Command => {
    const prompts = getSharedPrompts(shared)
    if (prompts.length === 0) {
      return {
        id: shared.id,
        text: shared.title,
        group: 'note',
        input: 'items',
        isAvailable: isOverFile,
      }
    }
    return {
      id: shared.id,
      text: shared.title,
      group: 'window',
      input: 'arguments',
      prompts,
      isAvailable: (target) => target.isReady && isHandled(shared.id, target),
    }
  }

  return {
    getSharedCommands,
    getSharedEntries: (): readonly Command[] => getHandledShared().map(createSharedEntry),
  }
}
