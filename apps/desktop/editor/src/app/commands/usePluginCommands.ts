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
import {
  EMPTY_WHEN_CONTEXT,
  createWhenAvailability,
  namesNote,
  type WindowWhenContext,
} from './whenContext'

export function usePluginCommands(
  core: VaultCore,
  pluginHost: PluginHost | undefined,
  context: WindowWhenContext = EMPTY_WHEN_CONTEXT,
) {
  const isWhenAvailable = createWhenAvailability(context)
  const getPluginCommands = () => pluginHost?.getPaletteCommands() ?? []
  /** How a plugin's command is drawn: its keystroke and its icon, where it declares them. */
  const getPluginLooks = (cmd: PluginPaletteCommand) => {
    const keys = cmd.shortcut ? findShortcutKeys(cmd.shortcut, navigator.userAgent) : undefined
    return {
      ...(keys ? { keys } : {}),
      ...(cmd.icon ? { icon: getNamedIcon(cmd.icon) } : {}),
    }
  }
  /**
   * The commands the plugins declare, as the palette lists them, each available where its condition
   * holds, and over the note in front where the condition reads target.isNote.
   */
  const getPaletteEntries = () =>
    getPluginCommands().map((cmd) => ({
      id: cmd.id,
      text: cmd.title,
      ...getPluginLooks(cmd),
      group: namesNote(cmd.when) ? ('note' as const) : ('window' as const),
      ...(cmd.when ? { when: cmd.when } : {}),
      isAvailable: (target: CommandTarget) => isWhenAvailable(cmd.when, target),
    }))

  /**
   * The commands the plugins declare over the window with a keystroke of their
   * own, as the screen of a window with no tab offers them.
   */
  const pluginWays = computed<readonly WelcomeAction[]>(() =>
    getPluginCommands()
      .filter((cmd) => !namesNote(cmd.when) && cmd.shortcut)
      .map((cmd) => ({ id: cmd.id, text: cmd.title, ...getPluginLooks(cmd) })),
  )

  /** The commands the plugins declare over one note, in the words the palette offers them by. */
  const getNoteCommands = (): readonly MenuEntry[] =>
    getPluginCommands()
      .filter((cmd) => namesNote(cmd.when))
      .map((cmd) => ({ id: cmd.id, text: cmd.title, group: 'note' }))

  const getCommandNote = async (path: string, title: string): Promise<CommandNote> => ({
    path,
    title,
    type: path === '' ? 'note' : ((await core.fileKinds([path])).get(path)?.type ?? 'note'),
  })

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
    getCommandNote,
    getNoteCommands,
    pluginWays,
  }
}
