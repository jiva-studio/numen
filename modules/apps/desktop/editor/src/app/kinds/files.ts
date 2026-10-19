/** The files tab kind of a window, with the creators it makes files through. */
import type { ShallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { cards, presets, WORDS as cardWords } from '@/entities/deck'
import { running } from '@/entities/artifact'
import { createFileCreators } from '@/entities/tab'
import {
  ANSWER_WORDS,
  createInvocation,
  openDestination,
  runInvocation,
  type DestinationDeps,
} from '@/features/command-palette'
import {
  createFilesKind as buildFilesKind,
  useFileTree,
  type FilesTabDeps,
  type FilesTabState,
} from '@/pages/file-manager'
import { isNotePath } from '@numen/plugin-sdk'
import { WORDS as words } from '@/shared/words'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WindowKindsDeps } from './deps'

export interface FilesKindDeps extends Pick<
  WindowKindsDeps,
  | 'core'
  | 'tabOpeners'
  | 'held'
  | 'runs'
  | 'editing'
  | 'vaults'
  | 'getTarget'
  | 'runCommand'
  | 'commandDeps'
  | 'tabLinks'
  | 'tabSync'
> {
  dragged: ShallowRef<readonly string[]>
  writeMessage: MessageWriter
  destinations: DestinationDeps
}

function findFilesTabId(held: FilesKindDeps['held'], state?: unknown): string | null {
  if (state) {
    const match = held.handle.each<FilesTabState>('files').find((tab) => tab.state === state)
    if (match) return match.id
  }
  const front = held.handle.front()
  if (front?.kind === 'files') return front.id
  return held.handle.last<FilesTabState>('files')?.id ?? null
}

interface FileCreatorsDeps {
  core: FilesKindDeps['core']
  tabOpeners: FilesKindDeps['tabOpeners']
  editing: FilesKindDeps['editing']
  writeMessage: FilesKindDeps['writeMessage']
}

function createFilesCreators(deps: FileCreatorsDeps) {
  const { core, tabOpeners, editing, writeMessage } = deps
  const fetchArtifact = async (path: string): Promise<void> => {
    try {
      await running.fetchArtifact(path)
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return
    }
    editing.notes.applyPathChanges([path])
  }

  return createFileCreators(
    {
      createDeck: (title, folder) => cards.createDeck(title, folder),
      createStencil: (title, folder, fields) => cards.createStencil(title, folder, fields),
      createPreset: (title, folder) => presets.createPreset(title, folder),
      createUrl: async (address, folder) => {
        const made = await core.createUrl(address, folder)
        if (made.ok) void fetchArtifact(made.value.path)
        return made
      },
    },
    tabOpeners,
    {
      words: { errors: words.errors },
      write: writeMessage,
    },
  )
}

interface FilesTabConfigDeps {
  deps: FilesKindDeps
  made: ReturnType<typeof createFilesCreators>
}

function createFilesTabPathHandlers(deps: FilesKindDeps) {
  const { held, destinations, tabSync } = deps
  return {
    openDestination: (
      landing: Parameters<FilesTabDeps['openDestination']>[0],
      state: Parameters<FilesTabDeps['openDestination']>[1],
    ) => {
      if (landing?.path && isNotePath(landing.path)) {
        const tabId = findFilesTabId(held, state)
        if (tabId && tabSync?.syncPathFromTab(tabId, landing.path, landing.title ?? '')) {
          return
        }
      }
      void openDestination(landing, destinations)
    },
    onSelectPath: (path: string, state: unknown) => {
      const entry = (state as FilesTabState).list.getEntryAt(path)
      if (entry && !entry.isFolder && (entry.kind === 'note' || isNotePath(path))) {
        const tabId = findFilesTabId(held, state)
        if (tabId) {
          tabSync?.syncPathFromTab(tabId, path, entry.name)
        }
      }
    },
  }
}

function createFilesTabDeps(config: FilesTabConfigDeps): FilesTabDeps {
  const { deps, made } = config
  const { runs, editing, vaults, getTarget, runCommand, commandDeps, dragged, writeMessage } = deps

  return {
    ...createFilesTabPathHandlers(deps),
    runCommand: (id, paths, details) => {
      const path = paths[0] ?? ''
      runCommand(id, {
        ...getTarget(),
        path,
        title: details.name,
        file: path,
        source: details.source,
        made: vaults.makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    movePath: (from, to) =>
      runInvocation(
        createInvocation('move', { ...getTarget(), path: from }, to),
        commandDeps(),
        ANSWER_WORDS,
      ),
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    createFolder: (path) =>
      runInvocation(
        createInvocation('createFolder', getTarget(), path),
        commandDeps(),
        ANSWER_WORDS,
      ),
    createNote: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    createDeck: (folder, name) => made.createFile('deck', folder, name),
    createStencil: (folder, name) =>
      made.createFile('stencil', folder, { name, fields: [cardWords.newField] }),
    createPreset: (folder, name) => made.createFile('preset', folder, name),
    importUrl: (folder, url) => made.createUrl(folder, url),
    importUrls: (folder, urls) => made.createUrls(folder, urls),
    showError: (text) => writeMessage(text, 'error'),
    canRun: (run) => runs.canRun(run),
    getArtifactStates: (path) => vaults.makes.value.get(path) ?? {},
  }
}

export function createFilesKind(deps: FilesKindDeps) {
  const made = createFilesCreators(deps)
  const filesDeps = createFilesTabDeps({ deps, made })
  const files = buildFilesKind(deps.held.handle, () => useFileTree(deps.core), filesDeps)
  return { files, made }
}
