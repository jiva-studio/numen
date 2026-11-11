/**
 * The menu of a tab of the window: the commands it offers run through the commands of the window.
 */
import type { PluginHost } from '@numen/plugin-loader'
import { useWindowTabMenu, type WindowTabs } from '@/entities/tab'
import { createNoteKindReader, type CommandItemLists } from '@/features/plugin-commands'
import type { createWindowTabMover } from '@/features/tab-move'
import { formatError, type Caught } from '@/shared/api/errors'
import { COMMAND } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { createHostConnection } from './createHostConnection'

/** What the menu of a tab is made from. */
export interface WindowTabMenuDeps {
  readonly windowTabs: WindowTabs
  readonly moves: Pick<ReturnType<typeof createWindowTabMover>, 'highlightWindow'>
  readonly pluginHost: Pick<PluginHost, 'slots'>
  readonly windowCommands: ReturnType<typeof createHostConnection>
  readonly itemLists: CommandItemLists
  readonly writeMessage: MessageWriter
}

export function createWindowTabMenu(deps: WindowTabMenuDeps) {
  return useWindowTabMenu({
    moveCommand: COMMAND.moveTabToWindow,
    windowTabs: deps.windowTabs,
    onHighlight: deps.moves.highlightWindow,
    slots: deps.pluginHost.slots,
    commands: deps.windowCommands,
    readNoteKind: createNoteKindReader(deps.itemLists),
    reportError: (error: Caught) => deps.writeMessage(formatError(error), 'error'),
  })
}
