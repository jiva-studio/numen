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

const isOverFile = (target: CommandTarget): boolean =>
  target.isReady && (target.file || target.path) !== ''

const getSharedPrompts = (id: string): readonly CommandPrompt[] =>
  (SHARED_COMMANDS.find((one) => one.id === id)?.arguments ?? [])
    .filter((one) => one.isRequired)
    .map(({ name, title, isRequired }) => ({ name, title, isRequired }))

export function createSharedCommands(
  getHandlers: () => readonly HandlerDeclaration[] | undefined,
  isWhenAvailable: (when: string | undefined, target: CommandTarget) => boolean,
) {
  /** The shared commands an enabled plugin handles. */
  const getSharedCommands = (): readonly PluginPaletteCommand[] => {
    const handlers = getHandlers() ?? []
    return SHARED_COMMANDS.filter((shared) =>
      handlers.some((one) => one.command === shared.id),
    ).map(({ id, title }) => ({ id, title }))
  }

  /** Whether a handler of the command is available over the target. */
  const isHandled = (id: string, target: CommandTarget): boolean =>
    (getHandlers() ?? []).some((one) => one.command === id && isWhenAvailable(one.when, target))

  /** A shared command as the palette lists it. */
  const createSharedEntry = (cmd: PluginPaletteCommand): Command => {
    const prompts = getSharedPrompts(cmd.id)
    if (prompts.length === 0) {
      return { id: cmd.id, text: cmd.title, group: 'note', input: 'items', isAvailable: isOverFile }
    }
    return {
      id: cmd.id,
      text: cmd.title,
      group: 'window',
      input: 'arguments',
      prompts,
      isAvailable: (target) => target.isReady && isHandled(cmd.id, target),
    }
  }

  return {
    getSharedCommands,
    getSharedEntries: (): readonly Command[] => getSharedCommands().map(createSharedEntry),
  }
}
