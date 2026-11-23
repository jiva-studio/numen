/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, type Ref, type ShallowRef } from 'vue'
import type {
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
import type { NoteApi, NoteCreator } from '@/entities/note'
import type { FileApi } from '@/entities/file'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { Task } from '@/entities/window'
import type { PluginNotes } from './pluginNotes'
import type { NoteBuffer } from '@numen/plugin-loader'

/** The window and the notes a plugin is given. */
export interface PluginServices {
  readonly workspace: WorkspaceHost
  readonly notes: VaultNotes
  readonly relations: VaultRelations
  readonly files: VaultFiles
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
  readonly getNoteInFocus: () => NoteRef
  readonly getToday: () => string
}

export function createWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, dragged } = deps
  return {
    window: {
      isReady: computed(() => !window.health.failure.value),
      openingPath: window.opening.path,
      readOpeningPath: () => window.opening.read(),
      getToday: () => deps.getToday(),
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
  }
}

/** What the notes of the vault are read and written through. */
export interface VaultNotesDeps {
  readonly notes: Pick<NoteApi, 'listRelatedNotes' | 'listHeadings'>
  readonly getFileKinds: FileApi['getFileKinds']
  readonly search: {
    readonly searchNames: (
      query: string,
      limit: number,
    ) => Promise<readonly { path: string; title: string }[]>
  }
  readonly making: NoteCreator
  readonly pluginNotes: PluginNotes
}

/** How many notes a search for a note to link to answers with, where the plugin names no number. */
const FOUND_LIMIT = 20

/** The notes of the vault as a plugin reads and writes them, with an action to forget cached related notes when the vault changes. */
export interface CachedVaultNotes extends VaultNotes {
  readonly clearRelatedNotes: () => void
}

export function createVaultNotes(deps: VaultNotesDeps): CachedVaultNotes {
  const { notes, search, making, pluginNotes } = deps
  type DepthOneGraph = Awaited<ReturnType<VaultNotes['listRelatedNotes']>>[number]
  const relatedAtDepthOne = new Map<string, DepthOneGraph>()
  const inFlightAtDepthOne = new Map<string, Promise<DepthOneGraph | undefined>>()
  return {
    clearRelatedNotes: () => {
      relatedAtDepthOne.clear()
      inFlightAtDepthOne.clear()
    },
    listRelatedNotes: async (paths, depth = 1) => {
      if (depth !== 1) return notes.listRelatedNotes(paths, depth)
      const missing: string[] = []
      const seen = new Set<string>()
      for (const path of paths) {
        if (!relatedAtDepthOne.has(path) && !inFlightAtDepthOne.has(path) && !seen.has(path)) {
          seen.add(path)
          missing.push(path)
        }
      }
      if (missing.length > 0) {
        const batch = notes.listRelatedNotes(missing, 1)
        missing.forEach((path, index) => {
          const pending = batch.then(
            (loaded) => {
              inFlightAtDepthOne.delete(path)
              const graph = loaded[index]
              if (graph) relatedAtDepthOne.set(path, graph)
              return graph
            },
            (error) => {
              inFlightAtDepthOne.delete(path)
              throw error
            },
          )
          inFlightAtDepthOne.set(path, pending)
        })
      }
      const resolved = await Promise.all(
        paths.map(async (path) => relatedAtDepthOne.get(path) ?? inFlightAtDepthOne.get(path)),
      )
      return resolved.map(
        (graph) =>
          graph ?? {
            focus: { path: '', title: '' },
            focusType: 'note',
            related: [],
            connections: [],
          },
      )
    },
    getHeadings: (paths) => notes.listHeadings(paths),
    findNotes: async (query, limit = FOUND_LIMIT) =>
      (await search.searchNames(query, limit)).map(({ path, title }) => ({ path, title })),
    createUntitledNote: () => making.createUntitledNote(''),
    read: (path) => pluginNotes.readNote(path),
    write: pluginNotes.write,
    readProperties: async (path) => (await deps.getFileKinds([path])).get(path)?.properties ?? {},
  }
}
