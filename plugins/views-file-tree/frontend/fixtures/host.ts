/**
 * A fake window hosting a files tab, and the files of a vault behind it, for
 * tests. Every request the tab makes to either is recorded in call order.
 */
import { defineComponent, h, ref } from 'vue'
import {
  NOTE_TYPE,
  type MenuEntry,
  type OpenFileOptions,
  type PathCommandTarget,
  type PathMenuRow,
  type VaultEntry,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { FileTreeHost } from '../model/tabDeps'

const command = (id: string, group: string, text = id): MenuEntry => ({ id, text, group })

const COPY = command('app:note.copyPath', 'file')
const REMOVE = command('app:note.trash', 'app:note.trash')
const LINKS: MenuEntry = {
  id: 'links',
  text: 'Links',
  group: 'links',
  children: [
    command('app:note.createChild', 'links'),
    command('app:note.createParent', 'links'),
    command('app:note.createJump', 'links'),
    command('app:note.rename', 'links'),
  ],
}
const NOTE = [
  command('app:note.open', 'open'),
  command('travel', 'open'),
  COPY,
  LINKS,
  command('ask', 'agent'),
  command('app:note.trash', 'app:note.trash', 'Move note to trash'),
]
/** The commands the plugins add over a file, by the extension of its name. */
const RUNS: Readonly<Record<string, readonly MenuEntry[]>> = {
  mp3: [command('extract', 'run'), command('refine', 'run')],
  pdf: [command('scan', 'run'), command('refine', 'run')],
  url: [command('downloadText', 'run'), command('downloadCopy', 'run')],
}

/** The window's commands over a row, shaped as the window offers them. */
export const getPathCommands = ({ entry, isSeveral }: PathMenuRow): readonly MenuEntry[] => {
  if (isSeveral) return [REMOVE]
  if (entry.isFolder) return [COPY, REMOVE]
  if (entry.isNote) return NOTE
  return [COPY, ...(RUNS[entry.path.split('.').pop() ?? ''] ?? []), REMOVE]
}

/** The icon a command or an entry is drawn with, one made-up component for each. */
const Icon = defineComponent({ name: 'DrawnIcon', setup: () => () => h('svg') })

/** What a files tab requests from the window to render its menu and rows. */
export const renderDeps = {
  getPathCommands,
  getCommandIcon: (id: string) => (id === 'unknown' ? null : Icon),
  getEntryIcon: () => Icon,
}

export interface HostOptions {
  readonly linked?: Readonly<Record<string, boolean>>
  readonly folders?: Readonly<Record<string, readonly VaultEntry[]>>
}

/** A fake window and vault that serve a files tab, and the requests the tab made to them. */
export function createHost({ linked = {}, folders = {} }: HostOptions = {}) {
  const recorded = {
    synced: [] as [string, string, string | undefined][],
    opened: [] as [string, OpenFileOptions | undefined][],
    commands: [] as [string, PathCommandTarget][],
    dragged: [] as (readonly string[])[],
    errors: [] as string[],
    made: [] as string[],
  }

  const workspace: WorkspaceHost = {
    isReady: ref(true),
    openingPath: ref(''),
    readOpeningPath: async () => '',
    draggedPaths: ref([]),
    setDraggedPaths: (paths) => {
      recorded.dragged.push(paths)
    },
    openFile: (path, options) => {
      recorded.opened.push([path, options])
    },
    openFileAt: () => undefined,
    getNoteInFocus: () => ({ path: '', title: '' }),
    syncPath: (tabId, path, title) => {
      recorded.synced.push([tabId, path, title])
      return linked[tabId] ?? false
    },
    runCommand: () => undefined,
    getNoteCommands: () => [],
    ask: () => undefined,
    showError: (text) => {
      recorded.errors.push(text)
    },
    getMessageWriter: () => () => undefined,
    getToday: () => '2026-01-01',
    runPathCommand: (id, target) => {
      recorded.commands.push([id, target])
    },
    ...renderDeps,
  }

  const record = (what: string) => async (folder: string, name?: string) => {
    recorded.made.push([what, folder, name].filter((one) => one !== undefined).join(' '))
    return `${folder}${name ?? what}`
  }

  const files: FileTreeHost['files'] = {
    list: async (folder) => folders[folder] ?? [],
    move: async (from, to) => {
      recorded.made.push(`move ${from} ${to}`)
    },
    createFolder: async (path) => {
      recorded.made.push(`folder ${path}`)
    },
    createNote: record(NOTE_TYPE),
  }

  const host: FileTreeHost = { workspace, files }
  return { host, recorded }
}
