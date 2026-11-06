/**
 * The commands the enabled plugins declare: how the palette and the welcome screen
 * list them, and the registration that runs them.
 */
import { computed, watch } from 'vue'
import {
  SHARED_COMMANDS,
  type CommandNote,
  type HandlerDeclaration,
  type MenuEntry,
  type PluginHost,
  type PluginPaletteCommand,
} from '@numen/plugin-host'
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

/** What a run names: the note, the others selected with it, and the tab it is over. */
type NoteTarget = Pick<CommandTarget, 'path' | 'title' | 'isFolder' | 'others'> & {
  readonly tab?: string
}

/** What a plugin command runs over: the target, the others selected with it, its type and properties. */
export const createCommandNoteReader =
  (core: Pick<VaultCore, 'fileKinds'>) =>
  async (target: NoteTarget): Promise<CommandNote> => {
    const { path } = target
    const kind = path === '' ? undefined : (await core.fileKinds([path])).get(path)
    return {
      path,
      title: target.title,
      type: kind?.type ?? 'note',
      ...(kind?.properties ? { properties: kind.properties } : {}),
      isFolder: target.isFolder ?? false,
      others: target.others ?? [],
      tab: target.tab ?? '',
    }
  }

/** Whether the target of the palette names a file, which the handlers of a shared command run over. */
const isOverFile = (target: CommandTarget): boolean =>
  target.isReady && (target.file || target.path) !== ''

export function usePluginCommands(
  core: VaultCore,
  pluginHost: PluginHost | undefined,
  context: WindowWhenContext = EMPTY_WHEN_CONTEXT,
  getHandlers: () => readonly HandlerDeclaration[] | undefined = () => [],
) {
  const isWhenAvailable = createWhenAvailability(context)
  const getPluginCommands = () => pluginHost?.getPaletteCommands() ?? []
  /** The shared commands an enabled plugin handles; their handlers are their items. */
  const getSharedCommands = (): readonly PluginPaletteCommand[] => {
    const handlers = getHandlers() ?? []
    return SHARED_COMMANDS.filter(({ id }) => handlers.some((one) => one.command === id)).map(
      ({ id, title }) => ({ id, title }),
    )
  }
  const getSharedEntries = () =>
    getSharedCommands().map((cmd) => ({
      id: cmd.id,
      text: cmd.title,
      group: 'note' as const,
      input: 'items' as const,
      isAvailable: isOverFile,
    }))
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
    getPluginCommands()
      .map((cmd) => ({
        id: cmd.id,
        text: cmd.title,
        ...getPluginLooks(cmd),
        group: namesNote(cmd.when) ? ('note' as const) : ('window' as const),
        ...(cmd.when ? { when: cmd.when } : {}),
        ...(pluginHost?.hasCommandItems(cmd.id) ? { input: 'items' as const } : {}),
        isAvailable: (target: CommandTarget) => isWhenAvailable(cmd.when, target),
      }))
      .concat(getSharedEntries())

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

  const getCommandNote = createCommandNoteReader(core)

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => [...pluginHost.getPaletteCommands(), ...getSharedCommands()],
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registerCommand(cmd.id, async (invocation) => {
            const note = await getCommandNote(invocation)
            if (pluginHost.hasCommandItems(cmd.id) && invocation.name !== '') {
              await pluginHost.executeCommand(cmd.id, note, invocation.name)
            } else await pluginHost.executeCommand(cmd.id, note)
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
