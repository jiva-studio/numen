/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, type Ref, type ShallowRef } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import type {
  AgentHost,
  AgentPort,
  BackgroundTaskHost,
  BufferHost,
  CommandsHost,
  MenuEntry,
  NoteRef,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultsHost,
  WindowHost,
  WorkspaceHost,
} from '@numen/plugin-host'
import type { CommandTarget, SearchDeps } from '@/features/command-palette'
import type { FileOpeners } from '@/entities/tab'
import type { NoteCreator } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Task } from '@/shared/notices/task'
import type { NotePort } from '@/app/ports/notes'
import type { useSettings } from '../window/useSettings'
import { createIconHost, type IconHostDeps } from './fileServices'
import type { PluginNotes } from './pluginNotes'
import type { NoteBuffer } from '@numen/plugin-host'

/** The window and the notes a plugin is given, once the window has made them. */
export interface PluginServices {
  workspace?: WorkspaceHost
  notes?: VaultNotes
  relations?: VaultRelations
  files?: VaultFiles
  agent?: AgentHost
  buffers?: BufferHost
  /** The window's store of open notes, which the tabs of the note kind read and write through. */
  noteBuffer?: NoteBuffer
  /** The commands of the window, which every plugin runs its commands through. */
  commands?: CommandsHost
  /** The vaults the window can show, which every plugin reads. */
  vaults?: VaultsHost
  /** The window the plugins run in. */
  window?: WindowHost
  /** The window's task list, which each plugin's task host reads. */
  taskList?: Readonly<Ref<readonly Task[]>>
  /** The background work of the window, which every plugin reads. */
  backgroundTasks?: BackgroundTaskHost
}

/** What the window hands a plugin's tabs. */
export interface WorkspaceHostDeps extends IconHostDeps {
  readonly getTarget: () => CommandTarget
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly window: {
    readonly failure: Readonly<Ref<string>>
    readonly opening: Readonly<Ref<string>>
    readInitialNote(): Promise<string>
  }
  readonly dragged: ShallowRef<readonly string[]>
  readonly tabOpeners: Pick<FileOpeners, 'openFile' | 'openFileAt' | 'findFileIcon'>
  readonly tabSync?: { syncPathFromTab(tabId: string, path: string, title?: string): boolean }
  readonly ask: (text: string) => void
  /** The note focused in the tool tab the user was last in. */
  readonly getNoteInFocus: () => NoteRef
  /** The commands the plugins declare over one note. */
  readonly getNoteCommands: () => readonly MenuEntry[]
  readonly writeMessage: MessageWriter
  /** A writer of messages under a name of its own. */
  readonly getMessageWriter: (name: string) => MessageWriter
  /** Today's date as a string. */
  readonly today: () => string
}

export function createWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, dragged, tabOpeners, writeMessage } = deps
  return {
    isReady: computed(() => !window.failure.value),
    openingPath: window.opening,
    readOpeningPath: () => window.readInitialNote(),
    draggedPaths: dragged,
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    openFile: (path, options) => {
      tabOpeners
        .openFile(path, options?.title ?? '', {
          ...(options?.placement ? { where: options.placement } : {}),
          ...(options?.line === undefined ? {} : { line: options.line }),
          ...(options?.selector ? { selector: options.selector } : {}),
        })
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    },
    openFileAt: (path, spans) => {
      tabOpeners
        .openFileAt(path, spans)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    },
    getNoteInFocus: () => deps.getNoteInFocus(),
    syncPath: (tabId, path, title) => deps.tabSync?.syncPathFromTab(tabId, path, title) ?? false,
    getNoteCommands: () => deps.getNoteCommands(),
    ask: (text) => deps.ask(text),
    showError: (text) => writeMessage(text, 'error'),
    getMessageWriter: (name) => deps.getMessageWriter(name),
    getToday: () => deps.today(),
    ...createIconHost({
      ...(deps.getPluginCommandIcon ? { getPluginCommandIcon: deps.getPluginCommandIcon } : {}),
      findFileIcon: tabOpeners.findFileIcon,
    }),
  }
}

/** What the notes of the vault are read and written through. */
export interface VaultNotesDeps {
  readonly core: Pick<NotePort, 'neighbourhood' | 'headings'>
  readonly search: Pick<SearchDeps, 'names'>
  readonly making: NoteCreator
  readonly pluginNotes: PluginNotes
}

/** How many notes a search for a note to link to answers with, where the plugin names no number. */
const FOUND_LIMIT = 20

export function createVaultNotes({
  core,
  search,
  making,
  pluginNotes,
}: VaultNotesDeps): VaultNotes {
  return {
    getNeighbourhood: (path) => core.neighbourhood(path),
    getHeadings: (paths) => core.headings(paths),
    findNotes: async (query, limit = FOUND_LIMIT) =>
      (await search.names(query, limit)).map(({ path, title }) => ({ path, title })),
    createUntitledNote: () => making.createUntitled(''),
    read: (path) => pluginNotes.readNote(path),
    write: (path, body) => pluginNotes.writeNote(path, body),
  }
}

/** What the relationships between notes are read and written through. */
export interface VaultRelationsDeps {
  readonly core: Pick<NotePort, 'relations'>
}

/** The relationships a plugin reads and writes; the core derives the roles and enforces the hierarchy. */
export function createVaultRelations({ core }: VaultRelationsDeps): VaultRelations {
  return {
    get roles() {
      return core.relations.roles
    },
    getConnection: (first, second, type) => core.relations.getConnection(first, second, type),
    saveConnection: (connection) => core.relations.saveConnection(connection),
    removeSide: (from, to, type) => core.relations.removeSide(from, to, type),
    removeConnection: (first, second, type) => core.relations.removeConnection(first, second, type),
    join: (from, to, side) => core.relations.join(from, to, side),
    createRelated: (request) => core.relations.createRelated(request),
    listBacklinks: (path) => core.relations.listBacklinks(path),
  }
}

/** What the agent a plugin's tabs ask is made from. */
export interface AgentHostDeps {
  readonly port: AgentPort
  readonly core: Pick<NotePort, 'resolveLinks'>
  readonly window: { readonly unreachable: Readonly<Ref<string>> }
  readonly settings: {
    readonly rest: Pick<
      ReturnType<typeof useSettings>['rest'],
      'getSetting' | 'writeSetting' | 'getModelsAt'
    >
  }
}

export function createAgentHost({ port, core, window, settings }: AgentHostDeps): AgentHost {
  return {
    ask: (query) => port.ask(query),
    finish: (conversation) => port.finish(conversation),
    resolveLinks: (text, vault) => core.resolveLinks('', text, vault),
    getUnreachable: () => window.unreachable.value,
    getSetting: (path) => settings.rest.getSetting(path),
    writeSetting: (path, value) => settings.rest.writeSetting(path, value),
    getModelsAt: (path) => settings.rest.getModelsAt(path),
  }
}
