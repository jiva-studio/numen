/**
 * What one plex tab requests of the window, served by the host's workspace and the
 * notes of the vault.
 */
import {
  type CommandsHost,
  type SlotHost,
  type VaultNotes,
  type VaultRelations,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { PlexLook, PlexTabDeps } from '../types'
import { PLEX_SLOT } from './slotNames'

const WIDGET_SLOTS = {
  above: PLEX_SLOT.nodeAbove,
  below: PLEX_SLOT.nodeBelow,
  over: PLEX_SLOT.node,
  boardTop: PLEX_SLOT.boardTop,
  boardBottom: PLEX_SLOT.boardBottom,
} as const

/** The host services a plex tab is made from, and the plugin's settings. */
export interface PlexHost {
  readonly workspace: WorkspaceHost
  readonly commands?: CommandsHost
  readonly slots?: SlotHost
  readonly notes: VaultNotes
  readonly relations: VaultRelations
  readonly look: PlexLook
}

/** What the plex tab `id` requests of the window. A note it opens goes to its linked tabs first. */
export function createPlexTabDeps(
  { workspace, commands, slots, notes, relations, look }: PlexHost,
  id: string,
): PlexTabDeps {
  return {
    tabId: id,
    look,
    editor: {
      createRelated: (request) => relations.createRelated(request),
      join: (from, to, side) => relations.join(from, to, side),
      getConnection: (first, second, type) => relations.getConnection(first, second, type),
      saveConnection: (connection) => relations.saveConnection(connection),
      deleteConnection: (first, second, type) => relations.deleteConnection(first, second, type),
    },
    creatable: relations.roles,
    window: {
      isReady: workspace.window.isReady,
      openingPath: workspace.window.openingPath,
      dragged: workspace.dragged.paths,
      openNote: (path, options) => {
        const title = options?.title ?? ''
        if (options?.line === undefined && workspace.files.syncPath(id, path, title)) return
        workspace.files.open(path, {
          title,
          ...(options?.showing ? { placement: options.showing } : {}),
          ...(options?.line === undefined ? {} : { line: options.line }),
        })
      },
      onSelectPath: (path, title) => {
        workspace.files.syncPath(id, path, title ?? '')
      },
      getNoteIcon: (tabKind) =>
        workspace.icons.getEntry(
          { path: '', name: '', isFolder: false, isNote: true, type: '', tabKind },
          false,
        ),
      showMessage: (text) => workspace.messages.showError(text),
      readOpeningPath: () => workspace.window.readOpeningPath(),
    },
    menu: {
      getMenuEntries: (target) => slots?.getEntries(PLEX_SLOT.context, target) ?? [],
      listItems: (command, target) => commands?.listItems(command, target) ?? Promise.resolve([]),
      runCommand: (command, target, itemId, values) =>
        commands?.run(command, target, itemId, values) ?? Promise.resolve(''),
    },
    watchWidgets: (side, handler) =>
      slots?.watchEntries(WIDGET_SLOTS[side], handler) ?? { dispose: () => undefined },
    searchNames: (query, limit) => notes.findNotes(query, limit),
  }
}
