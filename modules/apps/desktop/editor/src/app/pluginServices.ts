/**
 * The window and the notes of the vault, as a plugin's tabs are given them.
 */
import { computed, type Ref, type ShallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { VaultFiles, VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import type { SearchDeps } from '@/features/command-palette'
import type { FileOpeners } from '@/entities/tab'
import { CREATABLE, type NoteCreator } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { NotePort } from '@/app/ports/notes'
import { createPathCommandHost, type PathCommandDeps } from './fileServices'

/** The window and the notes a plugin is given, once the window has made them. */
export interface PluginServices {
  workspace?: WorkspaceHost
  notes?: VaultNotes
  files?: VaultFiles
}

/** What the window hands a plugin's tabs. */
export interface WorkspaceHostDeps extends PathCommandDeps {
  readonly window: {
    readonly failure: Readonly<Ref<string>>
    readonly opening: Readonly<Ref<string>>
    readInitialNote(): Promise<string>
  }
  readonly nesting: {
    readonly isNesting: Readonly<Ref<boolean>>
    readonly headings: Readonly<Ref<number>>
  }
  readonly dragged: ShallowRef<readonly string[]>
  readonly tabOpeners: Pick<FileOpeners, 'openFile'>
  readonly tabSync?: { syncPathFromTab(tabId: string, path: string, title?: string): boolean }
  readonly ask: (text: string) => void
  readonly writeMessage: MessageWriter
}

export function createWorkspaceHost(deps: WorkspaceHostDeps): WorkspaceHost {
  const { window, nesting, dragged, tabOpeners, writeMessage } = deps
  return {
    isReady: computed(() => !window.failure.value),
    openingPath: window.opening,
    readOpeningPath: () => window.readInitialNote(),
    headingDisplay: { isNesting: nesting.isNesting, count: nesting.headings },
    draggedPaths: dragged,
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    openFile: (path, options) => {
      tabOpeners
        .openFile(path, options?.title ?? '', {
          ...(options?.placement ? { where: options.placement } : {}),
          ...(options?.line === undefined ? {} : { line: options.line }),
        })
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    },
    syncPath: (tabId, path, title) => deps.tabSync?.syncPathFromTab(tabId, path, title) ?? false,
    runCommand: (id, { path, title }) => deps.runCommand(id, { ...deps.getTarget(), path, title }),
    ask: (text) => deps.ask(text),
    showError: (text) => writeMessage(text, 'error'),
    ...createPathCommandHost(deps),
  }
}

/** What the notes of the vault are read and written through. */
export interface VaultNotesDeps {
  readonly core: Pick<NotePort, 'neighbourhood' | 'headings'>
  readonly search: Pick<SearchDeps, 'names'>
  readonly making: NoteCreator
}

/** How many notes a search for a note to link to answers with, where the plugin names no number. */
const FOUND_LIMIT = 20

export function createVaultNotes({ core, search, making }: VaultNotesDeps): VaultNotes {
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
    creatableRelations: CREATABLE,
  }
}
