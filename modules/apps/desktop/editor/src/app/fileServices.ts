/**
 * The files of the vault as a plugin's tabs are given them: the folders listed,
 * the files made and moved, and the window's commands over a row.
 */
import { asFailure, formatErrorMessage, getKnownErrorCode } from '@numen/wire'
import type { VaultFiles, WorkspaceHost } from '@numen/plugin-sdk'
import { running, type ArtifactStates } from '@/entities/artifact'
import { createFileCreators, type FileOpeners, type useWindowTabs } from '@/entities/tab'
import type { CreateResult } from '@/entities/file'
import {
  ANSWER_WORDS,
  createInvocation,
  createPathCommands,
  runInvocation,
  type CommandDeps,
  type CommandTarget,
  type RunSupport,
} from '@/features/command-palette'
import type { NoteCreator } from '@/entities/note'
import { WORDS as words } from '@/shared/words'
import { findIcon } from '@/shared/icons'
import type { MessageWriter } from '@/shared/notices/messages'
import type { FilePort } from '@/app/ports/files'
import { getEntryIcon } from './entryIcon'

/** What making a file asks of the window. */
export interface FileMakersDeps {
  readonly core: Pick<FilePort, 'createUrl'>
  readonly tabOpeners: FileOpeners
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'createFileOfKind'>
  readonly notes: { applyPathChanges(paths: readonly string[]): void }
  readonly writeMessage: MessageWriter
}

/** The decks, stencils, presets and urls the window makes, each put in front of the person. */
export function createFileMakers({
  core,
  tabOpeners,
  windowTabs,
  notes,
  writeMessage,
}: FileMakersDeps) {
  const fetchArtifact = (path: string): void => {
    running.fetchArtifact(path).then(
      () => notes.applyPathChanges([path]),
      (error: unknown) => writeMessage(formatErrorMessage(error), 'error'),
    )
  }

  const createOfKind = async (
    kind: string,
    title: string,
    folder: string,
  ): Promise<CreateResult | null> => {
    const made = await windowTabs.createFileOfKind(kind, folder, title)
    if (made === undefined) {
      writeMessage(words.notOffered, 'error')
      return null
    }
    return made.ok ? made : asFailure(getKnownErrorCode(made.error))
  }

  return createFileCreators(
    {
      createDeck: (title, folder) => createOfKind('deck', title, folder),
      createStencil: (title, folder) => createOfKind('stencil', title, folder),
      createPreset: (title, folder) => createOfKind('preset', title, folder),
      createUrl: async (address, folder) => {
        const made = await core.createUrl(address, folder)
        if (made.ok) fetchArtifact(made.value.path)
        return made
      },
    },
    tabOpeners,
    { words: { errors: words.errors }, write: writeMessage },
  )
}

/** What the files of the vault are listed, made and moved through. */
export interface VaultFilesDeps {
  readonly core: Pick<FilePort, 'list'>
  readonly made: ReturnType<typeof createFileMakers>
  readonly making: NoteCreator
  readonly getTarget: () => CommandTarget
  readonly commandDeps: () => CommandDeps
}

export function createVaultFiles(deps: VaultFilesDeps): VaultFiles {
  const { core, made, making, getTarget, commandDeps } = deps
  const run = (id: string, target: CommandTarget, name: string) =>
    runInvocation(createInvocation(id, target, name), commandDeps(), ANSWER_WORDS)
  return {
    list: (folder) => core.list(folder),
    move: (from, to) => run('move', { ...getTarget(), path: from }, to),
    createFolder: (path) => run('createFolder', getTarget(), path),
    createNote: async (folder) => (await making.createUntitled(folder, []))?.path ?? '',
    createDeck: (folder, name) => made.createFile('deck', folder, name),
    createStencil: (folder, name) => made.createFile('stencil', folder, name),
    createPreset: (folder, name) => made.createFile('preset', folder, name),
    importUrl: (folder, url) => made.createUrl(folder, url),
    importUrls: (folder, urls) => made.createUrls(folder, urls),
  }
}

/** What the window's commands over a row are offered and run with. */
export interface PathCommandDeps {
  readonly runs: Pick<RunSupport, 'canRun'>
  readonly vaults: { readonly makes: { readonly value: ReadonlyMap<string, ArtifactStates> } }
  readonly getTarget: () => CommandTarget
  readonly runCommand: (id: string, target: CommandTarget) => void
}

/** The window's commands over a row of the vault, and the icons it draws them and its entries with. */
export function createPathCommandHost(
  deps: PathCommandDeps,
): Pick<WorkspaceHost, 'getPathCommands' | 'runPathCommand' | 'getCommandIcon' | 'getEntryIcon'> {
  const { runs, vaults, getTarget, runCommand } = deps
  return {
    getPathCommands: ({ entry, isSeveral }) =>
      createPathCommands(
        {
          source: entry.kind,
          isFolder: entry.isFolder,
          format: entry.path.endsWith('.epub') ? 'epub' : 'pdf',
          made: vaults.makes.value.get(entry.path),
        },
        isSeveral,
        (id) => runs.canRun(id),
      ),
    runPathCommand: (id, { paths, name, source }) => {
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
    getCommandIcon: (id) => findIcon(id),
    getEntryIcon: (entry, isOpen) => getEntryIcon(entry, isOpen),
  }
}
