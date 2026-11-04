/**
 * The commands the enabled plugins declare: how the palette and the welcome screen
 * list them, and the registration that runs them.
 */
import { computed, watch } from 'vue'
import type { CommandNote, MenuEntry, PluginHost, PluginPaletteCommand } from '@numen/plugin-host'
import type { WelcomeAction } from '@numen/shell'
import { findShortcutKeys, registerCommand, type CommandTarget } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import type { VaultCore } from '../vault'

export function usePluginCommands(core: VaultCore, pluginHost: PluginHost | undefined) {
  const getPluginCommands = () => pluginHost?.getPaletteCommands() ?? []
  /** How a plugin's command is drawn: its keystroke and its icon, where it declares them. */
  const getPluginLooks = (cmd: PluginPaletteCommand) => {
    const keys = cmd.shortcut ? findShortcutKeys(cmd.shortcut, navigator.userAgent) : undefined
    return {
      ...(keys ? { keys } : {}),
      ...(cmd.icon ? { icon: getNamedIcon(cmd.icon) } : {}),
    }
  }
  /** The command a plugin declares to put a note in front, and nothing where none does. */
  const findNoteDestination = () => getPluginCommands().find((cmd) => cmd.isNoteDestination)

  /** The commands the plugins declare, as the palette lists them, each available where its scope is. */
  const getPaletteEntries = () =>
    getPluginCommands().map((cmd) => ({
      id: cmd.id,
      text: cmd.title,
      ...getPluginLooks(cmd),
      group: cmd.scope === 'note' ? ('note' as const) : ('window' as const),
      isAvailable: (target: CommandTarget) =>
        cmd.scope !== 'note' || (target.isReady && target.path !== ''),
    }))

  /**
   * The commands the plugins declare over the window with a keystroke of their
   * own, as the screen of a window with no tab offers them.
   */
  const pluginWays = computed<readonly WelcomeAction[]>(() =>
    getPluginCommands()
      .filter((cmd) => cmd.scope !== 'note' && cmd.shortcut)
      .map((cmd) => ({ id: cmd.id, text: cmd.title, ...getPluginLooks(cmd) })),
  )

  /** The commands the plugins declare over one note, in the words the palette offers them by. */
  const getNoteCommands = (): readonly MenuEntry[] =>
    getPluginCommands()
      .filter((cmd) => cmd.scope === 'note')
      .map((cmd) => ({ id: cmd.id, text: cmd.title, group: 'note' }))

  const getCommandNote = async (path: string, title: string): Promise<CommandNote> => ({
    path,
    title,
    type: path === '' ? 'note' : ((await core.fileKinds([path])).get(path)?.type ?? 'note'),
  })

  /** The note put in front by the command a plugin declares for it; false where none declares one. */
  const showNote = async (path: string): Promise<boolean> => {
    const destination = findNoteDestination()
    if (!destination || !pluginHost) return false
    return pluginHost.executeCommand(destination.id, await getCommandNote(path, ''))
  }

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => pluginHost.getPaletteCommands(),
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registerCommand(cmd.id, async (invocation) => {
            await pluginHost.executeCommand(
              cmd.id,
              await getCommandNote(invocation.path, invocation.title),
            )
          }),
        )
      },
      { immediate: true },
    )
  }

  return {
    getPaletteEntries,
    findNoteDestination,
    getCommandNote,
    showNote,
    getNoteCommands,
    pluginWays,
  }
}
