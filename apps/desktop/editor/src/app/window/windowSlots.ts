/**
 * The slots of the window as the plugin host serves them: the entries the plugins and the application
 * place, how their commands are drawn, and the runner of the window that runs them over their target.
 */
import type { PluginHost, PluginHostDeps } from '@numen/plugin-host'
import { createCommands, type Command } from '@/features/command-palette'
import { WORDS as words } from '@/shared/words'
import type { DeclaredCommands } from '../commands/declaredCommands'
import { createSlotCommandFinder, readSlotEntries } from '../commands/pluginSlots'

export interface WindowSlotsDeps {
  readonly declaredCommands: Pick<DeclaredCommands, 'slots' | 'commands'>
  readonly hostHolder: { readonly current?: PluginHost }
  /** Writes what is wrong with the slot entries of a plugin to its messages. */
  readonly writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') => void
}

export function createWindowSlots(deps: WindowSlotsDeps) {
  let editorCommands: readonly Command[] | undefined
  const findCommand = createSlotCommandFinder({
    getPluginCommands: () => deps.hostHolder.current?.getPaletteCommands() ?? [],
    getDeclaredCommands: () => deps.declaredCommands.commands.value,
    getEditorCommands: () => (editorCommands ??= createCommands(words)),
  })
  const hostDeps: Required<
    Pick<PluginHostDeps, 'getSlotEntries' | 'findSlotCommand' | 'reportSlotIssue'>
  > = {
    getSlotEntries: () => readSlotEntries(deps.declaredCommands.slots.value),
    findSlotCommand: findCommand,
    reportSlotIssue: deps.writeIssue,
  }
  return { hostDeps }
}
