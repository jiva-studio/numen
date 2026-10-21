/**
 * The window a plex is drawn in, and the notes of a vault behind it, made up for
 * a test. Everything the plex asks of either is written down in the order asked.
 */
import { ref } from 'vue'
import type { NoteRef, OpenFileOptions, VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import type { PlexHost } from '../model/tabDeps'

export interface HostOptions {
  /** The note the vault opens with. */
  readonly opening?: string
  /** The tabs linked to each plex tab, by the identity of the plex tab. */
  readonly linked?: Readonly<Record<string, boolean>>
}

/** A window and a vault that answer a plex, and what the plex asked of them. */
export function createHost({ opening = 'Opening.md', linked = {} }: HostOptions = {}) {
  const openingPath = ref(opening)
  const isNesting = ref(true)
  const asked = { opening: [] as string[], headings: [] as (readonly string[])[] }
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
      asked.opening.push(openingPath.value)
      return openingPath.value
    },
    headingDisplay: { isNesting, count: ref(6) },
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
      focusType: 'note',
      related: [],
    }),
    getHeadings: async (paths) => {
      asked.headings.push(paths)
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
    creatableRelations: ['parent', 'child', 'jump'],
  }

  const host: PlexHost = { workspace, notes }
  return { host, openingPath, isNesting, asked, recorded }
}
