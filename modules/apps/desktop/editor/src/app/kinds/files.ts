/** The files tab kind of a window, with the creators it makes files through. */
import type { ShallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { cards, presets, WORDS as cardWords } from '@/entities/deck'
import { running } from '@/entities/artifact'
import { createFileCreators } from '@/entities/tab'
import {
  ANSWER_WORDS,
  invocationOf,
  openDestination,
  runInvocation,
  type DestinationDeps,
} from '@/features/command-palette'
import {
  createFilesKind as buildFilesKind,
  useFileTree,
  type FilesTabState,
} from '@/pages/file-manager'
import { isNotePath } from '../useTabSync'
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

export function createFilesKind({
  core,
  tabOpeners,
  held,
  runs,
  editing,
  vaults,
  getTarget,
  runCommand,
  commandDeps,
  dragged,
  writeMessage,
  destinations,
  tabSync,
}: FilesKindDeps) {
  const fetchArtifact = async (path: string): Promise<void> => {
    try {
      await running.fetchArtifact(path)
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return
    }
    editing.notes.applyPathChanges([path])
  }

  const made = createFileCreators(
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
    { errors: words.errors },
    writeMessage,
  )

  const files = buildFilesKind(held.handle, () => useFileTree(core), {
    openDestination: (landing, state) => {
      if (landing?.path && isNotePath(landing.path)) {
        const tabId = findFilesTabId(held, state)
        if (tabId && tabSync?.syncPathFromTab(tabId, landing.path, landing.title ?? '')) {
          return
        }
      }
      void openDestination(landing, destinations)
    },
    onSelectPath: (path, state) => {
      const entry = (state as FilesTabState).list.getEntryAt(path)
      if (entry && !entry.isFolder && (entry.kind === 'note' || isNotePath(path))) {
        const tabId = findFilesTabId(held, state)
        if (tabId) {
          tabSync?.syncPathFromTab(tabId, path, entry.name)
        }
      }
    },
    runCommand: (id, paths, name, source) => {
      const path = paths[0] ?? ''
      runCommand(id, {
        ...getTarget(),
        path,
        title: name,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    movePath: (from, to) =>
      runInvocation(
        invocationOf('move', { ...getTarget(), path: from }, to),
        commandDeps(),
        ANSWER_WORDS,
      ),
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    createFolder: (path) =>
      runInvocation(invocationOf('createFolder', getTarget(), path), commandDeps(), ANSWER_WORDS),
    createNote: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    createDeck: (folder, name) => made.createFile('deck', folder, name),
    createStencil: (folder, name) => made.createFile('stencil', folder, name, [cardWords.newField]),
    createPreset: (folder, name) => made.createFile('preset', folder, name),
    importUrl: (folder, url) => made.createUrl(folder, url),
    showError: (text) => writeMessage(text, 'error'),
    canRun: (run) => runs.canRun(run),
    getArtifactStates: (path) => vaults.makes.value.get(path) ?? {},
  })

  return { files, made }
}
