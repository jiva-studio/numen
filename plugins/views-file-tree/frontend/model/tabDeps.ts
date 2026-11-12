/**
 * What one files tab requests from the window, served by the host's workspace
 * and the files of the vault.
 */
import {
  getFileStem,
  isNotePath,
  SLOT,
  type DropHost,
  type CommandsHost,
  type SlotHost,
  type VaultFiles,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { FilesTabDeps } from '../types'

/** The host services a files tab is made from. */
export interface FileTreeHost {
  readonly workspace: WorkspaceHost
  readonly files: VaultFiles
  readonly drops?: DropHost
  readonly slots?: SlotHost
  readonly commands?: CommandsHost
}

/**
 * What the files tab `id` requests from the window. A note it opens or selects
 * goes to the tabs linked to it first, titled by its file name without the extension.
 */
export function createFilesTabDeps(
  { workspace, files, drops, slots, commands }: FileTreeHost,
  id: string,
): FilesTabDeps {
  return {
    window: {
      openEntry: ({ path, name, isNote }) => {
        if ((isNote || isNotePath(path)) && workspace.files.syncPath(id, path, getFileStem(name)))
          return
        workspace.files.open(path, { title: name || path, placement: 'current' })
      },
      onSelectPath: (path, state) => {
        const entry = state.list.queries.getEntryAt(path)
        if (entry && !entry.isFolder && (entry.isNote || isNotePath(path))) {
          workspace.files.syncPath(id, path, getFileStem(entry.name))
        }
      },
      setDraggedPaths: (paths) => workspace.dragged.setPaths(paths),
      showError: (text) => workspace.messages.showError(text),
      getEntryIcon: (entry, isOpen) => workspace.icons.getEntry(entry, isOpen),
    },
    menu: {
      getMenuEntries: (target) => slots?.getEntries(SLOT.fileTreeContext, target) ?? [],
      listItems: async (command, target) => (await commands?.listItems(command, target)) ?? [],
      runCommand: (command, target, itemId, values) =>
        commands?.run(command, target, itemId, values) ?? Promise.resolve(''),
    },
    transfer: {
      movePath: (from, to) => files.move(from, to),
      canDropData: (types) => drops?.canDrop(types) ?? false,
      dropData: async (folder, data) => (drops ? drops.drop(folder, data) : []),
    },
  }
}
