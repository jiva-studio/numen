/**
 * What one files tab asks of the window, answered by the host's workspace and
 * the files of the vault.
 */
import { isNotePath, type VaultFiles, type WorkspaceHost } from '@numen/plugin-sdk'
import type { FilesTabDeps } from '../types'

/** The host services a files tab is made from. */
export interface FileTreeHost {
  readonly workspace: WorkspaceHost
  readonly files: VaultFiles
}

/**
 * What the files tab `id` asks of the window. A note it opens or selects goes to
 * the tabs linked to it first.
 */
export function createFilesTabDeps({ workspace, files }: FileTreeHost, id: string): FilesTabDeps {
  return {
    openEntry: ({ path, name }) => {
      if (isNotePath(path) && workspace.syncPath(id, path, name)) return
      workspace.openFile(path, { title: name || path, placement: 'here' })
    },
    onSelectPath: (path, state) => {
      const entry = state.list.getEntryAt(path)
      if (entry && !entry.isFolder && (entry.isNote || isNotePath(path))) {
        workspace.syncPath(id, path, entry.name)
      }
    },
    runCommand: (command, paths, { name, isNote, isFolder }) =>
      workspace.runPathCommand(command, { paths, name, isNote, isFolder }),
    movePath: (from, to) => files.move(from, to),
    setDraggedPaths: (paths) => workspace.setDraggedPaths(paths),
    createFolder: (path) => files.createFolder(path),
    createNote: (folder) => files.createNote(folder),
    showError: (text) => workspace.showError(text),
    getPathCommands: (row) => workspace.getPathCommands(row),
    getCommandIcon: (command) => workspace.getCommandIcon(command),
    getEntryIcon: (entry, isOpen) => workspace.getEntryIcon(entry, isOpen),
  }
}
