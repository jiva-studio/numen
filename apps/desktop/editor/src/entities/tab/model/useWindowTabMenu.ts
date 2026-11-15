/**
 * The menu of a tab of the window, whose commands run through the commands of the window.
 */
import type { CommandsHost, CommandTarget as RunTarget, SlotHost } from '@numen/plugin-loader'
import type { Caught } from '@/shared/api/errors'
import type { WindowTabs } from './windowTabs'
import { useTabMenu, type NoteKind } from './useTabMenu'

/** The tabs of a window, and how the commands of the menu of a tab are read and run. */
interface WindowTabMenuDeps {
  readonly moveCommand: string
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly slots?: SlotHost | undefined
  readonly commands?: CommandsHost | undefined
  onHighlight?(this: void, windowId: string, isOn: boolean): void
  /** The type and the properties of the note of the path. */
  readNoteKind?(this: void, path: string): Promise<NoteKind>
  /** Shows the error of a command of the menu that failed, or of items that could not be read. */
  reportError?(this: void, error: Caught): void
}

const getTabFileTarget = (windowTabs: WindowTabMenuDeps['windowTabs'], id: string): RunTarget => {
  const tab = windowTabs.tabs.getTab(id)
  const on = tab ? tab.kind.address?.getTarget?.(tab.state) : undefined
  return {
    path: on?.file || on?.path || '',
    title: on?.title ?? '',
    isFolder: false,
    others: [],
    tab: id,
  }
}

/** The menu of a tab of the window, whose commands run through the commands of the window. */
export function useWindowTabMenu(deps: WindowTabMenuDeps) {
  const { windowTabs, slots, commands } = deps
  return useTabMenu({
    moveCommand: deps.moveCommand,
    slots,
    runCommand: (command, target, itemId, values) =>
      commands
        ? commands.run(command, target, itemId, values).then(() => undefined)
        : Promise.resolve(),
    listItems: (command, target) => commands?.listItems(command, target) ?? Promise.resolve([]),
    getTabTarget: (id) => getTabFileTarget(windowTabs, id),
    ...(deps.onHighlight ? { onHighlight: deps.onHighlight } : {}),
    ...(deps.readNoteKind ? { readNoteKind: deps.readNoteKind } : {}),
    ...(deps.reportError ? { reportError: deps.reportError } : {}),
  })
}
