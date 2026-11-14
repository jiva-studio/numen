/**
 * The window a plex is drawn in, and the notes of a vault behind it, as test
 * fixtures. Every request the plex makes of either is recorded in order.
 */
import { ref } from 'vue'
import {
  asValue,
  NOTE_TYPE,
  type CommandTarget,
  type CommandsHost,
  type SlotHost,
  type OpenFileOptions,
  type VaultNotes,
  type VaultRelations,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { PlexHost } from '../model/tabDeps'
import { getDefaultLook } from '../model/look'
import { getDefaultNames } from '../model/names'

export interface HostOptions {
  readonly opening?: string
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
    commands: [] as [string, CommandTarget | undefined][],
    entries: [] as [string, CommandTarget | undefined][],
    errors: [] as string[],
    made: [] as [string, string][],
    joined: [] as [string, string, string][],
  }

  const workspace: WorkspaceHost = {
    window: {
      isReady: ref(true),
      openingPath,
      readOpeningPath: async () => {
        request.opening.push(openingPath.value)
        return openingPath.value
      },
      getToday: () => '2026-01-01',
    },
    dragged: { paths: ref([]), setPaths: () => undefined },
    files: {
      open: (path, options) => {
        recorded.opened.push([path, options])
      },
      openAt: () => undefined,
      syncPath: (tabId, path, title) => {
        recorded.synced.push([tabId, path, title])
        return linked[tabId] ?? false
      },
    },
    messages: {
      showError: (text) => {
        recorded.errors.push(text)
      },
      getWriter: () => () => undefined,
    },
    icons: { getCommand: () => null, getEntry: () => ({}) },
    getNoteInFocus: () => ({ path: '', title: '' }),
  }

  const notes: VaultNotes = {
    listRelatedNotes: async (path) => ({
      focus: { path, title: path.replace(/\.md$/, '') },
      focusType: NOTE_TYPE,
      related: [],
      connections: [],
    }),
    getHeadings: async (paths) => {
      request.headings.push(paths)
      return new Map()
    },
    findNotes: async () => [],
    createUntitledNote: async () => 'Untitled.md',
    read: async () => ({ body: '' }),
    write: async () => undefined,
    readProperties: async () => ({}),
  }

  const relations: VaultRelations = {
    roles: ['parent', 'child', 'jump'],
    createRelated: async ({ from, role, title }) => {
      recorded.made.push([from, role])
      return asValue(title ? `${title}.md` : 'Made.md')
    },
    join: async (from, to, side) => {
      recorded.joined.push([from, to, side.role])
      return asValue(undefined)
    },
    getConnection: async (first, second, type = '') =>
      asValue({ first, second, role: null, type, firstSide: null, secondSide: null }),
    saveConnection: async () => asValue(undefined),
    removeSide: async () => asValue(undefined),
    removeConnection: async () => asValue(undefined),
    listBacklinks: async () => [],
    resolveLinks: async () => [],
  }

  const commands: CommandsHost = {
    getIds: () => [],
    watch: () => ({ dispose: () => undefined }),
    listItems: async () => [],
    run: async (id, target) => {
      recorded.commands.push([id, target])
      return ''
    },
  }

  const slots: SlotHost = {
    getEntries: (slot, target) => {
      recorded.entries.push([slot, target])
      return []
    },
    watchEntries: () => ({ dispose: () => undefined }),
  }

  const host: PlexHost = {
    workspace,
    commands,
    slots,
    notes,
    relations,
    nesting: { isNesting, headings: ref(6) },
    names: getDefaultNames(),
    look: getDefaultLook(),
  }
  return { host, openingPath, isNesting, requested: request, requestLog: request, recorded }
}
