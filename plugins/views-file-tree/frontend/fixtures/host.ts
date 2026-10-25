/**
 * The window a files tab is drawn in, and the files of a vault behind it, made
 * up for a test. Everything the tab asks of either is written down in the order
 * asked.
 */
import { defineComponent, h, ref } from 'vue'
import type {
  MenuEntry,
  OpenFileOptions,
  PathCommandTarget,
  PathMenuRow,
  VaultEntry,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { FileTreeHost } from '../model/tabDeps'

/** One command of the window, in the group it is drawn in. */
const command = (id: string, group: string, text = id): MenuEntry => ({ id, text, group })

const COPY = command('copy', 'file')
const REMOVE = command('remove', 'remove')
const NOTE = [
  command('read', 'open'),
  command('travel', 'open'),
  COPY,
  command('child', 'plex'),
  command('parent', 'plex'),
  command('jump', 'plex'),
  command('title', 'plex'),
  command('ask', 'agent'),
  command('remove', 'remove', 'Remove note'),
]
const RUNS: Readonly<Record<string, readonly MenuEntry[]>> = {
  recording: [command('transcribe', 'run'), command('proofread', 'run')],
  book: [command('recognise', 'run'), command('proofread', 'run')],
  url: [command('downloadText', 'run'), command('downloadCopy', 'run')],
}

/** The window's commands over a row, shaped as the window offers them. */
export const getPathCommands = ({ entry, isSeveral }: PathMenuRow): readonly MenuEntry[] => {
  if (isSeveral) return [REMOVE]
  if (entry.isFolder) return [COPY, REMOVE]
  if (entry.kind === 'note') return NOTE
  return [COPY, ...(RUNS[entry.kind] ?? []), REMOVE]
}

/** The icon a command or an entry is drawn with, one made-up component for each. */
const Icon = defineComponent({ name: 'DrawnIcon', setup: () => () => h('svg') })

/** What a files tab asks of the window to draw its menu and its rows. */
export const renderDeps = {
  getPathCommands,
  getCommandIcon: (id: string) => (id === 'unknown' ? null : Icon),
  getEntryIcon: () => Icon,
}

export interface HostOptions {
  /** The tabs linked to each files tab, by the identity of the files tab. */
  readonly linked?: Readonly<Record<string, boolean>>
  /** What each folder holds. */
  readonly folders?: Readonly<Record<string, readonly VaultEntry[]>>
}

/** A window and a vault that answer a files tab, and what the tab asked of them. */
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
    headingDisplay: { isNesting: ref(false), count: ref(0) },
    draggedPaths: ref([]),
    setDraggedPaths: (paths) => void recorded.dragged.push(paths),
    openFile: (path, options) => void recorded.opened.push([path, options]),
    openFileAt: () => {},
    getNoteInFocus: () => ({ path: '', title: '' }),
    syncPath: (tabId, path, title) => {
      recorded.synced.push([tabId, path, title])
      return linked[tabId] ?? false
    },
    runCommand: () => {},
    ask: () => {},
    showError: (text) => void recorded.errors.push(text),
    getMessageWriter: () => () => {},
    getToday: () => '2026-01-01',
    runPathCommand: (id, target) => void recorded.commands.push([id, target]),
    ...renderDeps,
  }

  const record = (what: string) => async (folder: string, name?: string) => {
    recorded.made.push([what, folder, name].filter((one) => one !== undefined).join(' '))
    return `${folder}${name ?? what}`
  }

  const files: FileTreeHost['files'] = {
    list: async (folder) => folders[folder] ?? [],
    move: async (from, to) => void recorded.made.push(`move ${from} ${to}`),
    createFolder: async (path) => void recorded.made.push(`folder ${path}`),
    createNote: record('note'),
    importUrl: record('url'),
    importUrls: async (folder, urls) => {
      recorded.made.push(`urls ${folder} ${urls.join(' ')}`)
      return urls
    },
  }

  const host: FileTreeHost = { workspace, files }
  return { host, recorded }
}
