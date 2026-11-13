/**
 * The commands the plugins registered, as the window reads and runs them: the palette's list, the
 * items a command offers, running one with an item, and the keys that run them.
 */
import type { CommandItem, CommandNote, CommandTarget } from '@numen/plugin-sdk'
import { createPaletteCommand } from './available'
import { readAvailableCommands } from './availableReader'
import { isCommandAvailable } from './commandWhen'
import { isSharedCommand } from '@numen/plugin-sdk'
import { createHandlerApi } from './handlers'
import { reportError, requestPluginStart, runCommand } from './events'
import type { HostRegistry } from './registry'
import type { KeybindingRegistry } from '../keybindings'
import { createShortcutHandler } from './shortcuts'
import type { CommandRun, HostDeps } from './types'

const toNote = (target: CommandTarget): CommandNote => ({ ...target, type: target.noteType ?? '' })

/**
 * The commands the plugins registered: the palette's list, running one, and the keys that run them.
 * A shared command offers and runs the handlers the plugins declare, which `handlers` lists.
 */
export function createCommandApi(
  deps: HostDeps,
  registry: HostRegistry,
  keybindings: KeybindingRegistry,
) {
  const getAvailableCommands = () => readAvailableCommands(deps, registry)
  const findAvailableCommand = (id: string) => getAvailableCommands().find((one) => one.id === id)
  const handlers = createHandlerApi(deps, registry)
  const executeCommand = async (
    id: string,
    note: CommandNote,
    run: CommandRun = {},
  ): Promise<boolean> => {
    const { itemId } = run
    if (isSharedCommand(id)) return handlers.runSharedCommand(id, { note, ...run })
    const command = findAvailableCommand(id)
    if (!command || !isCommandAvailable(command.declaration, note, deps)) return false
    let entry = command.entry
    if (!entry) {
      await deps.startPlugin?.(command.declaration.pluginId)
      entry = registry.commands.value.get(id)
    }
    if (!entry) return false
    await runCommand(deps, entry, { note, itemId, values: run.values, isAwaited: run.isAwaited })
    return true
  }
  return {
    handlers,
    getPaletteCommands: () => getAvailableCommands().map((one) => createPaletteCommand(one)),
    hasCommand: (id: string): boolean => findAvailableCommand(id)?.entry !== undefined,
    /** Whether the command offers items, as known without starting its plugin; reactive. */
    hasKnownItems: (id: string): boolean =>
      isSharedCommand(id) || registry.commands.value.get(id)?.reg.listItems !== undefined,
    /**
     * Whether the command offers items; reactive. Only the plugin knows, so a plugin that has not
     * started is started by the question and answers once it has.
     */
    hasCommandItems: (id: string): boolean => {
      if (isSharedCommand(id)) return true
      const entry = registry.commands.value.get(id)
      if (entry) return entry.reg.listItems !== undefined
      const command = findAvailableCommand(id)
      if (command) requestPluginStart(deps, command.declaration.pluginId)
      return false
    },
    listCommandItems: async (
      id: string,
      target: CommandTarget,
    ): Promise<readonly CommandItem[]> => {
      if (isSharedCommand(id)) return handlers.listHandlerItems(id, toNote(target))
      const pluginId = findAvailableCommand(id)?.declaration.pluginId
      if (!registry.commands.value.has(id) && pluginId) await deps.startPlugin?.(pluginId)
      const entry = registry.commands.value.get(id)
      try {
        return (await entry?.reg.listItems?.(target)) ?? []
      } catch (error) {
        if (entry) reportError(deps, entry.pluginId, error)
        return []
      }
    },
    /** Runs the command with the item picked; the handler gets the values of its arguments. */
    executeCommand,
    /**
     * Runs the command of the key event over the note, and returns whether the event is such a key.
     * The caller prevents the default of the event on true; `target` is what is in front, for the
     * condition of the command. A key of an application command is given to `runApp`.
     */
    handleShortcut: createShortcutHandler({ deps, keybindings, getAvailableCommands, executeCommand }),
  }
}
