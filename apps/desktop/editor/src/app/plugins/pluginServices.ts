/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, type Ref, type ShallowRef } from 'vue'
import {
  formatErrorMessage,
  type AgentHost,
  type AgentPort,
  type BufferHost,
  type MenuEntry,
  type NoteRef,
  type VaultFiles,
  type VaultNotes,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import type { SearchDeps } from '@/features/command-palette'
import type { FileOpeners } from '@/entities/tab'
import { CREATABLE, type NoteCreator } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Task } from '@/shared/notices/task'
import type { NotePort } from '@/app/ports/notes'
import type { useSettings } from '../window/useSettings'
import { createPathCommandHost, type PathCommandDeps } from './fileServices'
import type { PluginNotes } from './pluginNotes'

/** The window and the notes a plugin is given, once the window has made them. */
export interface PluginServices {
  workspace?: WorkspaceHost
  notes?: VaultNotes
  files?: VaultFiles
  agent?: AgentHost
  buffers?: BufferHost
  /** The window's task list, which each plugin's task host reads. */
  taskList?: Readonly<Ref<readonly Task[]>>
}

/** What the window hands a plugin's tabs. */
export interface WorkspaceHostDeps extends PathCommandDeps {
  readonly window: {
    readonly failure: Readonly<Ref<string>>
    readonly opening: Readonly<Ref<string>>
    readInitialNote(): Promise<string>
  }
  readonly dragged: ShallowRef<readonly string[]>
  readonly tabOpeners: Pick<FileOpeners, 'openFile' | 'openFileAt' | 'findFileIcon'>
  readonly tabSync?: { syncPathFromTab(tabId: string, path: string, title?: string): boolean }
  readonly ask: (text: string) => void
  /** The note the tool tab the person was last in relies on. */
  readonly getNoteInFocus: () => NoteRef
  /** The commands the plugins declare over one note. */
  readonly getNoteCommands: () => readonly MenuEntry[]
  readonly writeMessage: MessageWriter
  /** A writer of messages under a name of its own. */
  readonly getMessageWriter: (name: string) => MessageWriter
  /** The day it is for the person now. */
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
    runCommand: (id, { path, title }) => deps.runCommand(id, { ...deps.getTarget(), path, title }),
    getNoteCommands: () => deps.getNoteCommands(),
    ask: (text) => deps.ask(text),
    showError: (text) => writeMessage(text, 'error'),
    getMessageWriter: (name) => deps.getMessageWriter(name),
    getToday: () => deps.today(),
    ...createPathCommandHost({ ...deps, findFileIcon: tabOpeners.findFileIcon }),
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
    createUntitledNote: async () => (await making.createUntitled('', []))?.path ?? '',
    createInRelation: (from, relation) => making.createInRelation(from, relation),
    createWithTitle: (title, from, relation) => making.createWithTitle(title, from, relation),
    join: (...link) => making.join(...link),
    replaceLinks: (first, second, links) => making.replaceLinks(first, second, links),
    read: (path) => pluginNotes.readNote(path),
    write: (path, body) => pluginNotes.writeNote(path, body),
    creatableRelations: CREATABLE,
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
