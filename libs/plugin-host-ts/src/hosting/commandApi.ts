/**
 * The commands the plugins registered, as the window reads and runs them: the palette's list, the
 * items a command offers, running one with an item, and the plugins' shortcuts.
 */
import type { CommandItem, CommandNote, CommandTarget } from '@numen/plugin-sdk'
import { createPaletteCommand } from './available'
import { readAvailableCommands } from './availableReader'
import { isCommandAvailable } from './commandWhen'
import { isSharedCommand } from '@numen/plugin-sdk'
import { createHandlerApi } from './handlers'
import { reportError, runCommand } from './events'
import type { HostRegistry } from './registry'
import { matchesShortcut } from './shortcuts'
import type { CommandRun, HostDeps } from './types'

const toNote = (target: CommandTarget): CommandNote => ({ ...target, type: target.noteType ?? '' })

/**
 * The commands the plugins registered: the palette's list, running one, and the plugins' shortcuts.
 * A shared command offers and runs the handlers the plugins declare.
 */
export function createCommandApi(deps: HostDeps, registry: HostRegistry) {
  const getAvailable = () => readAvailableCommands(deps, registry)
  const findAvailable = (id: string) => getAvailable().find((one) => one.id === id)
  const handlers = createHandlerApi(deps, registry)
  return {
    ...handlers,
    getPaletteCommands: () => getAvailable().map(createPaletteCommand),
    hasCommand: (id: string): boolean => findAvailable(id)?.entry !== undefined,
    hasCommandItems: (id: string): boolean =>
      isSharedCommand(id) || registry.commands.value.get(id)?.reg.listItems !== undefined,
    listCommandItems: async (
      id: string,
      target: CommandTarget,
    ): Promise<readonly CommandItem[]> => {
      if (isSharedCommand(id)) return handlers.listHandlerItems(id, toNote(target))
      const entry = registry.commands.value.get(id)
      try {
        return (await entry?.reg.listItems?.(target)) ?? []
      } catch (error) {
        if (entry) reportError(deps, entry.pluginId, error)
        return []
      }
    },
    /** Runs the command with the item picked; the handler gets the values of its arguments. */
    executeCommand: async (
      id: string,
      note: CommandNote,
      run: CommandRun = {},
    ): Promise<boolean> => {
      const { itemId } = run
      if (isSharedCommand(id)) return handlers.runSharedCommand(id, { note, ...run })
      const command = findAvailable(id)
      if (!command || !isCommandAvailable(command.declaration, note, deps)) return false
      let entry = command.entry
      if (!entry) {
        await deps.waitForPlugin?.(command.declaration.pluginId)
        entry = registry.commands.value.get(id)
      }
      if (!entry) return false
      await runCommand(deps, entry, { note, itemId, values: run.values, isAwaited: run.isAwaited })
      return true
    },
    handleShortcut: (event: KeyboardEvent, getNote: () => Promise<CommandNote>): boolean => {
      for (const { entry, declaration } of getAvailable()) {
        if (entry && matchesShortcut(event, declaration.shortcut)) {
          getNote()
            .then((note) => {
              if (!isCommandAvailable(declaration, note, deps)) return
              return runCommand(deps, entry, { note })
            })
            .catch((error: unknown) => reportError(deps, entry.pluginId, error))
          return true
        }
      }
      return false
    },
  }
}
