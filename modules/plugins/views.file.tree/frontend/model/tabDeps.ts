/**
 * What one files tab asks of the window, answered by the host's workspace and
 * the files of the vault.
 */
import { isNotePath, type VaultFiles, type WorkspaceHost } from '@numen/plugin-sdk'
import type { FilesTabDeps } from '../types'

/** The host services a files tab is made from. */
export interface FileTreeHost {
  readonly workspace: WorkspaceHost
  readonly files: VaultFiles & {
    createDeck?(folder: string, name: string): Promise<string>
    createStencil?(folder: string, name: string): Promise<string>
    createPreset?(folder: string, name: string): Promise<string>
  }
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
      if (entry && !entry.isFolder && (entry.kind === 'note' || isNotePath(path))) {
        workspace.syncPath(id, path, entry.name)
      }
    },
    runCommand: (command, paths, { name, source }) =>
      workspace.runPathCommand(command, { paths, name, source }),
    movePath: (from, to) => files.move(from, to),
    setDraggedPaths: (paths) => workspace.setDraggedPaths(paths),
    createFolder: (path) => files.createFolder(path),
    createNote: (folder) => files.createNote(folder),
    createDeck: (folder, name) =>
      files.createDeck ? files.createDeck(folder, name) : Promise.resolve(''),
    createStencil: (folder, name) =>
      files.createStencil ? files.createStencil(folder, name) : Promise.resolve(''),
    createPreset: (folder, name) =>
      files.createPreset ? files.createPreset(folder, name) : Promise.resolve(''),
    importUrl: (folder, url) => files.importUrl(folder, url),
    importUrls: (folder, urls) => files.importUrls(folder, urls),
    showError: (text) => workspace.showError(text),
    getPathCommands: (row) => workspace.getPathCommands(row),
    getCommandIcon: (command) => workspace.getCommandIcon(command),
    getEntryIcon: (entry, isOpen) => workspace.getEntryIcon(entry, isOpen),
  }
}
