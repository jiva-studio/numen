/**
 * The window a plex is drawn in, and the notes of a vault behind it, as test
 * fixtures. Every request the plex makes of either is recorded in order.
 */
import { ref } from 'vue'
import {
  NOTE_TYPE,
  type NoteRef,
  type OpenFileOptions,
  type VaultNotes,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { PlexHost } from '../model/tabDeps'

export interface HostOptions {
  /** The note the vault opens with. */
  readonly opening?: string
  /** The tabs linked to each plex tab, by the identity of the plex tab. */
  readonly linked?: Readonly<Record<string, boolean>>
}

/** A window and a vault that serve a plex, and the requests the plex made of them. */
export function createHost({ opening = 'Opening.md', linked = {} }: HostOptions = {}) {
  const openingPath = ref(opening)
  const isNesting = ref(true)
  const request = { opening: [] as string[], headings: [] as (readonly string[])[] }
  const recorded = {
    synced: [] as [string, string, string | undefined][],
    opened: [] as [string, OpenFileOptions | undefined][],
    commands: [] as [string, NoteRef][],
    questions: [] as string[],
    errors: [] as string[],
    made: [] as [string, string][],
    joined: [] as [string, string, string][],
  }

  const workspace: WorkspaceHost = {
    isReady: ref(true),
    openingPath,
    readOpeningPath: async () => {
      request.opening.push(openingPath.value)
      return openingPath.value
    },
    draggedPaths: ref([]),
    setDraggedPaths: () => {},
    openFile: (path, options) => void recorded.opened.push([path, options]),
    openFileAt: () => {},
    getNoteInFocus: () => ({ path: '', title: '' }),
    syncPath: (tabId, path, title) => {
      recorded.synced.push([tabId, path, title])
      return linked[tabId] ?? false
    },
    runCommand: (id, target) => void recorded.commands.push([id, target]),
    getNoteCommands: () => [{ id: 'plugin:share', text: 'Share the note', group: NOTE_TYPE }],
    ask: (text) => void recorded.questions.push(text),
    showError: (text) => void recorded.errors.push(text),
    getMessageWriter: () => () => {},
    getToday: () => '2026-01-01',
    getPathCommands: () => [],
    runPathCommand: () => {},
    getCommandIcon: () => null,
    getEntryIcon: () => ({}),
  }

  const notes: VaultNotes = {
    getNeighbourhood: async (path) => ({
      focus: { path, title: path.replace(/\.md$/, '') },
      focusType: NOTE_TYPE,
      related: [],
    }),
    getHeadings: async (paths) => {
      request.headings.push(paths)
      return new Map()
    },
    findNotes: async () => [],
    createUntitledNote: async () => 'Untitled.md',
    createInRelation: async (from, relation) => {
      recorded.made.push([from, relation])
      return { path: 'Made.md', title: 'Made' }
    },
    createWithTitle: async (title, from, relation) => {
      recorded.made.push([from, relation])
      return { path: `${title}.md`, title }
    },
    join: async (from, to, relation) => {
      recorded.joined.push([from, to, relation])
      return true
    },
    replaceLinks: async () => true,
    read: async () => ({ body: '' }),
    write: async () => {},
    creatableRelations: ['parent', 'child', 'jump'],
  }

  const host: PlexHost = { workspace, notes, nesting: { isNesting, headings: ref(6) } }
  return { host, openingPath, isNesting, requested: request, requestLog: request, recorded }
}
