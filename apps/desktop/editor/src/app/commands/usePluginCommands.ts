/**
 * The commands the enabled plugins declare: how the palette and the welcome screen
 * list them, and the registration that runs them.
 */
import { computed, watch } from 'vue'
import type {
  CommandNote,
  HandlerDeclaration,
  PluginHost,
  PluginPaletteCommand,
} from '@numen/plugin-host'
import type { WelcomeAction } from '@numen/shell'
import {
  findShortcutKeys,
  registerCommand,
  type Command,
  type CommandTarget,
} from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import type { VaultCore } from '../vault'
import {
  createEmptyWhenContext,
  createWhenAvailability,
  hasNoteVariable,
  type WindowWhenContext,
} from './whenContext'
import { createSharedCommands } from './sharedCommands'

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

export function usePluginCommands(
  core: VaultCore,
  pluginHost: PluginHost | undefined,
  context: WindowWhenContext = createEmptyWhenContext(),
  getHandlers: () => readonly HandlerDeclaration[] | undefined = () => [],
  getCommandNote: ReturnType<typeof createCommandNoteReader> = createCommandNoteReader(core),
) {
  const isWhenAvailable = createWhenAvailability(context)
  const getPluginCommands = () => pluginHost?.getPaletteCommands() ?? []
  const { getSharedCommands, getSharedEntries } = createSharedCommands(getHandlers, isWhenAvailable)
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
  const getPaletteEntries = (): readonly Command[] => [
    ...getPluginCommands().map((cmd): Command => ({
      id: cmd.id,
      text: cmd.title,
      ...getPluginLooks(cmd),
      group: hasNoteVariable(context.whenReader.getCondition(cmd.when))
        ? ('note' as const)
        : ('window' as const),
      ...(cmd.when ? { when: cmd.when } : {}),
      ...(pluginHost?.hasCommandItems(cmd.id) ? { input: 'items' as const } : {}),
      isAvailable: (target: CommandTarget) => isWhenAvailable(cmd.when, target),
    })),
    ...getSharedEntries(),
  ]

  /**
   * The commands the plugins declare over the window with a keystroke of their
   * own, as the screen of a window with no tab offers them.
   */
  const pluginWays = computed<readonly WelcomeAction[]>(() =>
    getPluginCommands()
      .filter((cmd) => !hasNoteVariable(context.whenReader.getCondition(cmd.when)) && cmd.shortcut)
      .map((cmd) => ({ id: cmd.id, text: cmd.title, ...getPluginLooks(cmd) })),
  )

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => [...pluginHost.getPaletteCommands(), ...getSharedCommands()],
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registerCommand(cmd.id, async (invocation) => {
            const note = await getCommandNote(invocation)
            if (invocation.values) {
              await pluginHost.executeCommand(cmd.id, note, { values: invocation.values })
            } else if (pluginHost.hasCommandItems(cmd.id) && invocation.name !== '') {
              await pluginHost.executeCommand(cmd.id, note, { itemId: invocation.name })
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
    pluginWays,
  }
}
