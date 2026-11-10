/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, shallowRef, type Ref, type ShallowRef } from 'vue'
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
  WorkspaceFiles,
  WorkspaceHost,
} from '@numen/plugin-loader'
import type { SearchDeps } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { FileOpeners } from '@/entities/tab'
import { getNeighborhood, listHeadings, resolveLinks, type NoteCreator } from '@/entities/note'
import { getFileKinds } from '@/entities/file'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Task } from '@/entities/window'
import type { useSettings } from '@/pages/settings'
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

/**
 * The services a plugin is given, stored from the start of the window: the plugin host reads them when
 * a plugin asks, and the parts of the window that make them provide them once they exist.
 */
export function createPluginServices() {
  const stored = shallowRef<PluginServices>({})
  return {
    get: (): PluginServices => stored.value,
    provide: (services: PluginServices): void => {
      stored.value = { ...stored.value, ...services }
    },
  }
}

export type PluginServicesStore = ReturnType<typeof createPluginServices>

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

function createWorkspaceFiles(deps: WorkspaceHostDeps): WorkspaceFiles {
  const { tabOpeners, writeMessage } = deps
  return {
    open: (path, options) => {
      tabOpeners
        .openFile(path, options?.title ?? '', {
          ...(options?.placement ? { where: options.placement } : {}),
          ...(options?.line === undefined ? {} : { line: options.line }),
          ...(options?.selector ? { selector: options.selector } : {}),
        })
        .catch((error: unknown) => writeMessage(formatError(error), 'error'))
    },
    openAt: (path, spans) => {
      tabOpeners
        .openFileAt(path, spans)
        .catch((error: unknown) => writeMessage(formatError(error), 'error'))
    },
    syncPath: (tabId, path, title) => deps.tabSync?.syncPathFromTab(tabId, path, title) ?? false,
  }
}

export function createWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, dragged, tabOpeners, writeMessage } = deps
  return {
    window: {
      isReady: computed(() => !window.health.failure.value),
      openingPath: window.opening.path,
      readOpeningPath: () => window.opening.read(),
      getToday: () => deps.today(),
    },
    dragged: {
      paths: dragged,
      setPaths: (paths) => {
        dragged.value = paths
      },
    },
    files: createWorkspaceFiles(deps),
    messages: {
      showError: (text) => writeMessage(text, 'error'),
      getWriter: (name) => deps.getMessageWriter(name),
    },
    icons: createIconHost({
      ...(deps.getPluginCommandIcon ? { getPluginCommandIcon: deps.getPluginCommandIcon } : {}),
      findFileIcon: tabOpeners.findFileIcon,
    }),
    getNoteInFocus: () => deps.getNoteInFocus(),
    ask: (text) => deps.ask(text),
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
    getNeighborhood: (path) => getNeighborhood(path),
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
      readonly read: Pick<
        ReturnType<typeof useSettings>['rest']['read'],
        'getSetting' | 'getModelsAt'
      >
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
