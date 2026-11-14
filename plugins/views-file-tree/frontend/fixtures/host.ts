/**
 * A fake window hosting a files tab, the files of a vault behind it, and the entries of the slot
 * the tab draws its menu from, for tests. Every request the tab makes is recorded in call order.
 */
import { defineComponent, h, ref, type Component } from 'vue'
import {
  NOTE_TYPE,
  SLOT,
  type OpenFileOptions,
  type SlotCommand,
  type SlotItem,
  type SlotSeparator,
  type SlotSubmenu,
  type SlotView,
  type VaultEntry,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createMockCommandsHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import type { FileTreeHost } from '../model/tabDeps'

const TREE = 'md.numen.views.file.tree'
const CHAT = 'org.example.chat'

/** The icon a command or an entry is drawn with, one made-up component for each. */
const Icon = defineComponent({ name: 'DrawnIcon', setup: () => () => h('svg') })

const Sketch = () => h('em', 'sketch')

const command = (pluginId: string, id: string, title: string, icon?: Component): SlotCommand => ({
  kind: 'command',
  key: `${pluginId}/command/${id}`,
  pluginId,
  command: pluginId === 'app' ? `app:${id}` : `${pluginId}:${id}`,
  title,
  ...(icon ? { icon } : {}),
  shortcut: '',
  hasItems: false,
})

const separator = (index: number): SlotSeparator => ({
  kind: 'separator',
  key: `separator/${index}`,
})

export const OPEN = command('app', 'file.open', 'Open note', Icon)
const RENAME_ENTRY = command(TREE, 'tree.renameFile', 'Rename')
export const COPY_PATH = command('app', 'file.copyPath', 'Copy path')
export const TRASH = command('app', 'file.remove', 'Move to trash')
const NEW_NOTE_ENTRY = command('app', 'note.create', 'Create note')
const NEW_FOLDER_ENTRY = command('app', 'folder.create', 'Create folder')
export const ASK = command(CHAT, 'note.ask', 'Ask about the note')

export const NEW_MENU: SlotSubmenu = {
  kind: 'submenu',
  key: `${TREE}/submenu/fileTree.new`,
  pluginId: TREE,
  submenu: 'fileTree.new',
  title: 'New',
  entries: [NEW_NOTE_ENTRY, NEW_FOLDER_ENTRY],
}

const SKETCH: SlotView = {
  kind: 'view',
  key: `${CHAT}/view/sketch`,
  pluginId: CHAT,
  view: 'sketch',
  component: Sketch,
}

/** The entries of the menu on a row, as the slot registry resolves them. */
export const ROW_MENU: readonly SlotItem[] = [
  OPEN,
  separator(1),
  NEW_MENU,
  separator(2),
  RENAME_ENTRY,
  COPY_PATH,
  separator(3),
  ASK,
  SKETCH,
  separator(4),
  TRASH,
]

/** What a files tab requests from the window to render its rows. */
export const renderDeps = {
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
    dragged: [] as (readonly string[])[],
    errors: [] as string[],
    made: [] as string[],
  }

  const workspace: WorkspaceHost = {
    window: {
      isReady: ref(true),
      openingPath: ref(''),
      readOpeningPath: async () => '',
      getToday: () => '2026-01-01',
    },
    dragged: {
      paths: ref([]),
      setPaths: (paths) => {
        recorded.dragged.push(paths)
      },
    },
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
    icons: { getCommand: () => null, getEntry: renderDeps.getEntryIcon },
    getNoteInFocus: () => ({ path: '', title: '' }),
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
      recorded.made.push(`createFolder ${path}`)
    },
    createNote: record(NOTE_TYPE),
  }

  const slots = createMockSlotHost()
  slots.setEntries(SLOT.fileTreeContext, ROW_MENU)

  const commands = createMockCommandsHost()

  const host: FileTreeHost = { workspace, files, slots, commands }
  return { host, recorded, slots, commands }
}
