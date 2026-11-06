/**
 * The commands the plugins registered, as the window reads and runs them: the palette's list, the
 * items a command offers, running one with its argument, and the plugins' shortcuts.
 */
import type { CommandItem, CommandNote, CommandTarget } from '@numen/plugin-sdk'
import { createPaletteCommand } from './available'
import { readAvailableCommands } from './availableReader'
import { isCommandAvailable } from './commandWhen'
import { createHandlerApi, isSharedCommand } from './handlers'
import { reportError, runCommand } from './events'
import type { HostRegistry } from './registry'
import { matchesShortcut } from './shortcuts'
import type { PluginHostDeps } from './types'

/** The note a place's target is, by the note type the place read. */
const toNote = (target: CommandTarget): CommandNote => ({ ...target, type: target.noteType ?? '' })

/**
 * The commands the plugins registered: the palette's list, running one, and the plugins' shortcuts.
 * A shared command offers and runs the handlers the plugins declare.
 */
export function createCommandApi(deps: PluginHostDeps, registry: HostRegistry) {
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
    executeCommand: async (id: string, note: CommandNote, argument?: string): Promise<boolean> => {
      if (isSharedCommand(id)) return handlers.runSharedCommand(id, note, argument)
      const command = findAvailable(id)
      if (!command || !isCommandAvailable(command.declaration, note, deps)) return false
      let entry = command.entry
      if (!entry) {
        await deps.waitForPlugin?.(command.declaration.pluginId)
        entry = registry.commands.value.get(id)
      }
      if (!entry) return false
      await runCommand(deps, entry, { note, argument })
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
