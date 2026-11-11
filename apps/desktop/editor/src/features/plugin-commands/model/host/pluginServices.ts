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
  WorkspaceFiles,
  WorkspaceHost,
  WorkspaceIcons,
} from '@numen/plugin-loader'
import type { FileOpeners } from '@/entities/tab'
import { getNeighborhood, listHeadings, resolveLinks, type NoteCreator } from '@/entities/note'
import { getFileKinds } from '@/entities/file'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { Task } from '@/entities/window'
import type { SettingsStore } from '@/entities/settings'
import type { PluginNotes } from './pluginNotes'
import type { NoteBuffer } from '@numen/plugin-loader'

/** The window and the notes a plugin is given. */
export interface PluginServices {
  readonly workspace: WorkspaceHost
  readonly notes: VaultNotes
  readonly relations: VaultRelations
  readonly files: VaultFiles
  readonly agent: AgentHost
  readonly buffers: BufferHost
  readonly noteBuffer: NoteBuffer
  readonly commands: CommandsHost
  readonly vaults: VaultsHost
  readonly window: WindowHost
  readonly taskList: Readonly<Ref<readonly Task[]>>
  readonly backgroundTasks: BackgroundTaskHost
}

/** What the files a plugin's tabs open are opened through. */
export interface WorkspaceFilesDeps {
  readonly tabOpeners: Pick<FileOpeners, 'openFile' | 'openFileAt'>
  readonly tabSync?: { syncPathFromTab(tabId: string, path: string, title?: string): boolean }
  readonly writeMessage: MessageWriter
}

/** The files a plugin's tabs open, and the linked tabs a path change moves. */
export function createWorkspaceFiles(deps: WorkspaceFilesDeps): WorkspaceFiles {
  const { tabOpeners, writeMessage } = deps
  return {
    open: (path, options) => {
      tabOpeners
        .openFile(path, options?.title ?? '', {
          ...(options?.placement ? { where: options.placement } : {}),
          ...(options?.line === undefined ? {} : { line: options.line }),
          ...(options?.selector ? { selector: options.selector } : {}),
        })
        .catch((error: Caught) => writeMessage(formatError(error), 'error'))
    },
    openAt: (path, spans) => {
      tabOpeners
        .openFileAt(path, spans)
        .catch((error: Caught) => writeMessage(formatError(error), 'error'))
    },
    syncPath: (tabId, path, title) => deps.tabSync?.syncPathFromTab(tabId, path, title) ?? false,
  }
}

/** What the messages a plugin's tabs write are written through. */
export interface WorkspaceMessagesDeps {
  readonly writeMessage: MessageWriter
  readonly getMessageWriter: (name: string) => MessageWriter
}

/** The messages a plugin's tabs write: an error, or a message under a name of their own. */
export function createWorkspaceMessages(deps: WorkspaceMessagesDeps): WorkspaceHost['messages'] {
  return {
    showError: (text) => deps.writeMessage(text, 'error'),
    getWriter: (name) => deps.getMessageWriter(name),
  }
}

/** What the window hands a plugin's tabs. */
export interface WorkspaceHostDeps {
  readonly window: {
    readonly health: { readonly failure: Readonly<Ref<string>> }
    readonly opening: { readonly path: Readonly<Ref<string>>; read(): Promise<string> }
  }
  readonly dragged: ShallowRef<readonly string[]>
  readonly files: WorkspaceFiles
  readonly icons: WorkspaceIcons
  readonly messages: WorkspaceHost['messages']
  readonly ask: (text: string) => void
  readonly getNoteInFocus: () => NoteRef
  readonly today: () => string
}

export function createWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, dragged } = deps
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
    files: deps.files,
    messages: deps.messages,
    icons: deps.icons,
    getNoteInFocus: () => deps.getNoteInFocus(),
    ask: (text) => deps.ask(text),
  }
}

/** What the notes of the vault are read and written through. */
export interface VaultNotesDeps {
  readonly search: {
    readonly names: (
      query: string,
      limit: number,
    ) => Promise<readonly { path: string; title: string }[]>
  }
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
        SettingsStore['read'],
        'getSetting' | 'getModelsAt'
      >
      readonly write: Pick<SettingsStore['write'], 'writeSetting'>
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
