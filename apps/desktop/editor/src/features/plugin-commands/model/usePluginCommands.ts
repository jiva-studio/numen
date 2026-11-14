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
} from '@numen/plugin-loader'
import type { WelcomeAction } from '@numen/workbench'
import { win } from '@numen/ui'
import { type Command } from '@/entities/command'
import { findShortcutKeys, type CommandRegistry, type CommandTarget } from '@/shared/model/command'
import type { FileKind } from '@/entities/file'
import { getNamedIcon } from '@/shared/ui/icons'
import {
  createEmptyWhenContext,
  createWhenAvailability,
  hasNoteVariable,
  type WindowWhenContext,
} from '../lib/whenContext'
import { createSharedCommands } from '../lib/sharedCommands'

/** What a run names: the note, the others selected with it, and the tab it is over. */
type NoteTarget = Pick<CommandTarget, 'path' | 'title' | 'isFolder' | 'others'> & {
  readonly tab?: string
}

/** What a plugin command runs over: the target, the others selected with it, its type and properties. */
export const createCommandNoteReader =
  (core: { fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>> }) =>
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

/** What `usePluginCommands` reads beyond the host: the context of the conditions and the shared handlers. */
export interface PluginCommandsExtras {
  readonly context?: WindowWhenContext
  readonly getHandlers?: () => readonly HandlerDeclaration[] | undefined
}

export function usePluginCommands(
  registry: Pick<CommandRegistry, 'register'>,
  getCommandNote: ReturnType<typeof createCommandNoteReader>,
  pluginHost: PluginHost | undefined,
  extras: PluginCommandsExtras = {},
) {
  const context = extras.context ?? createEmptyWhenContext()
  const getHandlers = extras.getHandlers ?? (() => [])
  const isWhenAvailable = createWhenAvailability(context)
  const getPluginCommands = () => pluginHost?.commands.getPaletteCommands() ?? []
  const { getSharedCommands, getSharedEntries } = createSharedCommands(getHandlers, isWhenAvailable)
  /** How a plugin's command is drawn: its keystroke and its icon, where it declares them. */
  const getPluginLooks = (cmd: PluginPaletteCommand) => {
    const key = pluginHost?.keybindings.getKeyFor(cmd.id)
    const keys = key ? findShortcutKeys(key, win.navigator.userAgent) : undefined
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
      ...(pluginHost?.commands.hasCommandItems(cmd.id) ? { input: 'items' as const } : {}),
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
      .filter(
        (cmd) =>
          !hasNoteVariable(context.whenReader.getCondition(cmd.when)) &&
          pluginHost?.keybindings.getKeyFor(cmd.id),
      )
      .map((cmd) => ({ id: cmd.id, text: cmd.title, ...getPluginLooks(cmd) })),
  )

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => [...pluginHost.commands.getPaletteCommands(), ...getSharedCommands()],
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registry.register(cmd.id, async (invocation) => {
            const note = await getCommandNote(invocation.target)
            if (invocation.values) {
              await pluginHost.commands.executeCommand(cmd.id, note, { values: invocation.values })
            } else if (pluginHost.commands.hasCommandItems(cmd.id) && invocation.name !== '') {
              await pluginHost.commands.executeCommand(cmd.id, note, { itemId: invocation.name })
            } else await pluginHost.commands.executeCommand(cmd.id, note)
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
