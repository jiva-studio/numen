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
  /** The slots of the window the menu of a row is drawn from, where the host offers them. */
  readonly slots?: SlotHost
  /** The commands of the window and of the plugins, run over a row. */
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
    openEntry: ({ path, name }) => {
      if (isNotePath(path) && workspace.syncPath(id, path, getFileStem(name))) return
      workspace.openFile(path, { title: name || path, placement: 'current' })
    },
    onSelectPath: (path, state) => {
      const entry = state.list.getEntryAt(path)
      if (entry && !entry.isFolder && (entry.isNote || isNotePath(path))) {
        workspace.syncPath(id, path, getFileStem(entry.name))
      }
    },
    getMenuEntries: (target) => slots?.getEntries(SLOT.fileTreeContext, target) ?? [],
    listItems: async (command, target) => (await commands?.listItems(command, target)) ?? [],
    runCommand: async (command, target, argument) => {
      await commands?.run(command, target, argument)
    },
    movePath: (from, to) => files.move(from, to),
    setDraggedPaths: (paths) => workspace.setDraggedPaths(paths),
    createFolder: (path) => files.createFolder(path),
    createNote: (folder) => files.createNote(folder),
    canDropData: (types) => drops?.canDrop(types) ?? false,
    dropData: async (folder, data) => (drops ? drops.drop(folder, data) : []),
    showError: (text) => workspace.showError(text),
    getEntryIcon: (entry, isOpen) => workspace.getEntryIcon(entry, isOpen),
  }
}
