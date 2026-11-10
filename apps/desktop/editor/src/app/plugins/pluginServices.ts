/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, type Ref, type ShallowRef } from 'vue'
import type {
  AgentHost,
  AgentPort,
  BackgroundTaskHost,
  BufferHost,
  CommandsHost,
  NoteRef,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultsHost,
  WindowHost,
  WorkspaceHost,
} from '@numen/plugin-loader'
import type { SearchDeps } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { FileOpeners } from '@/entities/tab'
import { getNeighbourhood, listHeadings, resolveLinks, type NoteCreator } from '@/entities/note'
import { getFileKinds } from '@/entities/file'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Task } from '@/entities/window'
import type { useSettings } from '../window/useSettings'
import { createIconHost, type IconHostDeps } from './fileServices'
import type { PluginNotes } from './pluginNotes'
import type { NoteBuffer } from '@numen/plugin-loader'

/** The window and the notes a plugin is given, once the window has made them. */
export interface PluginServices {
  workspace?: WorkspaceHost
  notes?: VaultNotes
  relations?: VaultRelations
  files?: VaultFiles
  agent?: AgentHost
  buffers?: BufferHost
  noteBuffer?: NoteBuffer
  commands?: CommandsHost
  vaults?: VaultsHost
  window?: WindowHost
  taskList?: Readonly<Ref<readonly Task[]>>
  backgroundTasks?: BackgroundTaskHost
}

/** What the window hands a plugin's tabs. */
export interface WorkspaceHostDeps extends IconHostDeps {
  readonly getTarget: () => CommandTarget
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly window: {
    readonly health: { readonly failure: Readonly<Ref<string>> }
    readonly opening: { readonly path: Readonly<Ref<string>>; read(): Promise<string> }
  }
  readonly dragged: ShallowRef<readonly string[]>
  readonly tabOpeners: Pick<FileOpeners, 'openFile' | 'openFileAt' | 'findFileIcon'>
  readonly tabSync?: { syncPathFromTab(tabId: string, path: string, title?: string): boolean }
  readonly ask: (text: string) => void
  readonly getNoteInFocus: () => NoteRef
  readonly writeMessage: MessageWriter
  readonly getMessageWriter: (name: string) => MessageWriter
  readonly today: () => string
}

export function getWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, dragged, tabOpeners, writeMessage } = deps
  return {
    isReady: computed(() => !window.health.failure.value),
    openingPath: window.opening.path,
    readOpeningPath: () => window.opening.read(),
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
        .catch((error: unknown) => writeMessage(formatError(error), 'error'))
    },
    openFileAt: (path, spans) => {
      tabOpeners
        .openFileAt(path, spans)
        .catch((error: unknown) => writeMessage(formatError(error), 'error'))
    },
    getNoteInFocus: () => deps.getNoteInFocus(),
    syncPath: (tabId, path, title) => deps.tabSync?.syncPathFromTab(tabId, path, title) ?? false,
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
  readonly search: Pick<SearchDeps, 'names'>
  readonly making: NoteCreator
  readonly pluginNotes: PluginNotes
}

/** How many notes a search for a note to link to answers with, where the plugin names no number. */
const FOUND_LIMIT = 20

export function createVaultNotes({ search, making, pluginNotes }: VaultNotesDeps): VaultNotes {
  return {
    getNeighbourhood: (path) => getNeighbourhood(path),
    getHeadings: (paths) => listHeadings(paths),
    findNotes: async (query, limit = FOUND_LIMIT) =>
      (await search.names(query, limit)).map(({ path, title }) => ({ path, title })),
    createUntitledNote: () => making.createUntitled(''),
    read: (path) => pluginNotes.readNote(path),
    write: (path, body) => pluginNotes.writeNote(path, body),
    readProperties: async (path) => (await getFileKinds([path])).get(path)?.properties ?? {},
  }
}

/** What the agent a plugin's tabs ask is made from. */
export interface AgentHostDeps {
  readonly port: AgentPort
  readonly window: { readonly vault: { readonly unreachable: Readonly<Ref<string>> } }
  readonly settings: {
    readonly rest: {
      readonly read: Pick<ReturnType<typeof useSettings>['rest']['read'], 'getSetting' | 'getModelsAt'>
      readonly write: Pick<ReturnType<typeof useSettings>['rest']['write'], 'writeSetting'>
    }
  }
}

export function createAgentHost({ port, window, settings }: AgentHostDeps): AgentHost {
  return {
    ask: (query) => port.ask(query),
    finish: (conversation) => port.finish(conversation),
    resolveLinks: (text, vault) => resolveLinks('', text, vault),
    getUnreachable: () => window.vault.unreachable.value,
    getSetting: (path) => settings.rest.read.getSetting(path),
    writeSetting: (path, value) => settings.rest.write.writeSetting(path, value),
    getModelsAt: (path) => settings.rest.read.getModelsAt(path),
  }
}
