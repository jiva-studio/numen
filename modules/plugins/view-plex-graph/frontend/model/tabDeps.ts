/**
 * What one plex tab asks of the window, answered by the host's workspace and the
 * notes of the vault.
 */
import type { VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import type { PlexTabDeps } from '../types'

/** The host services a plex tab is made from. */
export interface PlexHost {
  readonly workspace: WorkspaceHost
  readonly notes: VaultNotes
}

/** What the plex tab `id` asks of the window. A note it opens goes to its linked tabs first. */
export function createPlexTabDeps({ workspace, notes }: PlexHost, id: string): PlexTabDeps {
  return {
    editor: {
      createInRelation: (from, relation) => notes.createInRelation(from, relation),
      createWithTitle: (title, from, relation) => notes.createWithTitle(title, from, relation),
      join: (from, to, relation, label) => notes.join(from, to, relation, label),
      replaceLinks: (first, second, links) => notes.replaceLinks(first, second, links),
    },
    isReady: workspace.isReady,
    isNesting: workspace.headingDisplay.isNesting,
    headings: workspace.headingDisplay.count,
    openNote: (path, options) => {
      const title = options?.title ?? ''
      if (workspace.syncPath(id, path, title)) return
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
    askAgent: (text) => workspace.ask(text),
    runCommand: (command, path, title) => workspace.runCommand(command, { path, title }),
    openingPath: workspace.openingPath,
    readOpeningPath: () => workspace.readOpeningPath(),
    dragged: workspace.draggedPaths,
    showMessage: (text) => workspace.showError(text),
    createUntitledNote: () => notes.createUntitledNote(),
    creatable: notes.creatableRelations,
    searchNames: (query, limit) => notes.findNotes(query, limit),
  }
}
