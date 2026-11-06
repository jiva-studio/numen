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

/** What a run names: the note, the others selected with it, and the tab it is over. */
type NoteTarget = Pick<CommandTarget, 'path' | 'title' | 'isFolder' | 'others'> & {
  readonly tab?: string
}

/** What a plugin command runs over: the target, the others selected with it, and its type. */
export const createCommandNoteReader =
  (core: Pick<VaultCore, 'fileKinds'>) =>
  async (target: NoteTarget): Promise<CommandNote> => {
    const { path } = target
    return {
      path,
      title: target.title,
      type: path === '' ? 'note' : ((await core.fileKinds([path])).get(path)?.type ?? 'note'),
      isFolder: target.isFolder ?? false,
      others: target.others ?? [],
      tab: target.tab ?? '',
    }
  }

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

  const getCommandNote = createCommandNoteReader(core)

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => pluginHost.getPaletteCommands(),
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registerCommand(cmd.id, async (invocation) => {
            await pluginHost.executeCommand(cmd.id, await getCommandNote(invocation))
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
