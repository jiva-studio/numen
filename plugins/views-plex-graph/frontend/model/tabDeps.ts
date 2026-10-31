/**
 * What one plex tab requests of the window, served by the host's workspace and the
 * notes of the vault.
 */
import type { VaultNotes, VaultRelations, WorkspaceHost } from '@numen/plugin-sdk'
import type { PlexTabDeps } from '../types'
import type { Nesting } from './nesting'

/** The host services a plex tab is made from, and the plugin's nesting settings. */
export interface PlexHost {
  readonly workspace: WorkspaceHost
  readonly notes: VaultNotes
  readonly relations: VaultRelations
  readonly nesting: Nesting
}

/** What the plex tab `id` requests of the window. A note it opens goes to its linked tabs first. */
export function createPlexTabDeps(
  { workspace, notes, relations, nesting }: PlexHost,
  id: string,
): PlexTabDeps {
  return {
    editor: {
      createRelated: (request) => relations.createRelated(request),
      join: (from, to, side) => relations.join(from, to, side),
      getConnection: (first, second, type) => relations.getConnection(first, second, type),
      saveConnection: (connection) => relations.saveConnection(connection),
      removeConnection: (first, second, type) => relations.removeConnection(first, second, type),
    },
    isReady: workspace.isReady,
    isNesting: nesting.isNesting,
    headings: nesting.headings,
    openNote: (path, options) => {
      const title = options?.title ?? ''
      if (options?.line === undefined && workspace.syncPath(id, path, title)) return
      workspace.openFile(path, {
        title,
        ...(options?.showing ? { placement: options.showing } : {}),
        ...(options?.line === undefined ? {} : { line: options.line }),
      })
    },
    onSelectPath: (path, title) => {
      workspace.syncPath(id, path, title ?? '')
    },
    readHeadings: (paths) => notes.getHeadings(paths),
    runCommand: (command, path, title) => workspace.runCommand(command, { path, title }),
    getNoteCommands: () => workspace.getNoteCommands(),
    getCommandIcon: (command) => workspace.getCommandIcon(command),
    getNoteIcon: (type) =>
      workspace.getEntryIcon({ path: '', name: '', isFolder: false, isNote: true, type }, false),
    openingPath: workspace.openingPath,
    readOpeningPath: () => workspace.readOpeningPath(),
    dragged: workspace.draggedPaths,
    showMessage: (text) => workspace.showError(text),
    createUntitledNote: () => notes.createUntitledNote(),
    creatable: relations.roles,
    searchNames: (query, limit) => notes.findNotes(query, limit),
  }
}
