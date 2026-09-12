/** The files tab kind of a window, with the creators it makes files through. */
import type { ShallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { cards, presets, WORDS as cardWords } from '@/entities/deck'
import { running } from '@/shared/artifacts'
import { createFileCreators } from '@/entities/tab'
import { does, invocationOf, lands, type DestinationDeps } from '@/features/command-palette'
import { filesKind, useFileTree } from '@/pages/file-manager'
import { WORDS as words } from '@/shared/words'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WindowKindsDeps } from './deps'

export interface FilesKindDeps
  extends Pick<
    WindowKindsDeps,
    'core' | 'puts' | 'held' | 'runs' | 'editing' | 'vaults' | 'where' | 'carries' | 'doing'
  > {
  dragged: ShallowRef<readonly string[]>
  told: MessageWriter
  places: DestinationDeps
}

export function createFilesKind({
  core,
  puts,
  held,
  runs,
  editing,
  vaults,
  where,
  carries,
  doing,
  dragged,
  told,
  places,
}: FilesKindDeps) {
  const fetches = async (path: string): Promise<void> => {
    try {
      await running.fetchArtifact(path)
    } catch (error) {
      told(formatErrorMessage(error), 'error')
      return
    }
    editing.notes.changed([path])
  }

  const made = createFileCreators(
    {
      createDeck: (title, folder) => cards.createDeck(title, folder),
      createStencil: (title, folder, fields) => cards.createStencil(title, folder, fields),
      createPreset: (title, folder) => presets.makes(title, folder),
      createUrl: async (address, folder) => {
        const made = await core.createUrl(address, folder)
        if (made.path) void fetches(made.path)
        return made
      },
    },
    puts,
    { errors: words.errors },
    told,
  )

  const files = filesKind(held.handle, () => useFileTree(core), {
    openDestination: (landing) => void lands(landing, places),
    runCommand: (id, paths, name, source) => {
      const path = paths[0] ?? ''
      carries(id, {
        ...where(),
        path,
        title: name,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    movePath: (from, to) => does(invocationOf('move', { ...where(), path: from }, to), doing(), words),
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    createFolder: (path) => does(invocationOf('makeFolder', where(), path), doing(), words),
    createNote: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    createDeck: (folder, name) => made.makes('deck', folder, name),
    createStencil: (folder, name) => made.makes('stencil', folder, name, [cardWords.newField]),
    createPreset: (folder, name) => made.makes('preset', folder, name),
    importAddress: (folder, address) => made.imports(folder, address),
    showError: (text) => told(text, 'error'),
    canRun: (run) => runs.canRun(run),
  })

  return { files, made }
}
