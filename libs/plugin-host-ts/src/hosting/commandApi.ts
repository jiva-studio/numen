/**
 * The commands the plugins registered, as the window reads and runs them: the palette's list, the
 * choices a command offers, running one with its choice, and the plugins' shortcuts.
 */
import type { CommandChoice, CommandNote, CommandTarget } from '@numen/plugin-sdk'
import { createPaletteCommand } from './available'
import { readAvailableCommands } from './availableReader'
import { reportError, runCommand } from './events'
import type { HostRegistry } from './registry'
import { matchesShortcut } from './shortcuts'
import type { PluginHostDeps } from './types'

/** The commands the plugins registered: the palette's list, running one, and the plugins' shortcuts. */
export function createCommandApi(deps: PluginHostDeps, registry: HostRegistry) {
  const getAvailable = () => readAvailableCommands(deps, registry)
  const findAvailable = (id: string) => getAvailable().find((one) => one.id === id)
  return {
    getPaletteCommands: () => getAvailable().map(createPaletteCommand),
    hasCommand: (id: string): boolean => findAvailable(id)?.entry !== undefined,
    hasCommandChoices: (id: string): boolean =>
      registry.commands.value.get(id)?.reg.getChoices !== undefined,
    getCommandChoices: async (
      id: string,
      target: CommandTarget,
    ): Promise<readonly CommandChoice[]> => {
      const entry = registry.commands.value.get(id)
      try {
        return (await entry?.reg.getChoices?.(target)) ?? []
      } catch (error) {
        if (entry) reportError(deps, entry.pluginId, error)
        return []
      }
    },
    executeCommand: async (id: string, note: CommandNote, choice?: string): Promise<boolean> => {
      const command = findAvailable(id)
      if (!command) return false
      let entry = command.entry
      if (!entry) {
        await deps.waitForPlugin?.(command.declaration.pluginId)
        entry = registry.commands.value.get(id)
      }
      if (!entry) return false
      await runCommand(deps, entry, { note, choice })
      return true
    },
    handleShortcut: (event: KeyboardEvent, getNote: () => Promise<CommandNote>): boolean => {
      for (const { entry, declaration } of getAvailable()) {
        if (entry && matchesShortcut(event, declaration.shortcut)) {
          getNote()
            .then((note) => {
              if (declaration.scope === 'note' && note.path === '') return
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
