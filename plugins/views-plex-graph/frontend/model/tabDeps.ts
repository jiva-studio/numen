/**
 * What one plex tab asks of the window, answered by the host's workspace and the
 * notes of the vault.
 */
import type { VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import type { PlexTabDeps } from '../types'
import type { Nesting } from './nesting'

/** The host services a plex tab is made from, and the plugin's nesting settings. */
export interface PlexHost {
  readonly workspace: WorkspaceHost
  readonly notes: VaultNotes
  readonly nesting: Nesting
}

/** What the plex tab `id` asks of the window. A note it opens goes to its linked tabs first. */
export function createPlexTabDeps(
  { workspace, notes, nesting }: PlexHost,
  id: string,
): PlexTabDeps {
  return {
    editor: {
      createInRelation: (from, relation) => notes.createInRelation(from, relation),
      createWithTitle: (title, from, relation) => notes.createWithTitle(title, from, relation),
      join: (from, to, relation, label) => notes.join(from, to, relation, label),
      replaceLinks: (first, second, links) => notes.replaceLinks(first, second, links),
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
    creatable: notes.creatableRelations,
    searchNames: (query, limit) => notes.findNotes(query, limit),
  }
}
