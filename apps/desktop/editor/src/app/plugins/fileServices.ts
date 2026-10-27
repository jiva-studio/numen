/**
 * The files of the vault as a plugin's tabs are given them: the folders listed,
 * the files made and moved, and the window's commands over a row.
 */
import type { Component } from 'vue'
import {
  asFailure,
  getKnownErrorCode,
  type ContextMenuRegistration,
  type ContextMenuTarget,
  type VaultFiles,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createFileCreators, type FileOpeners, type useWindowTabs } from '@/entities/tab'
import type { CreateResult } from '@/entities/file'
import {
  ANSWER_WORDS,
  createInvocation,
  createPathCommands,
  runInvocation,
  type CommandDeps,
  type CommandTarget,
} from '@/features/command-palette'
import type { NoteCreator } from '@/entities/note'
import { WORDS as words } from '@/shared/words'
import { findIcon } from '@/shared/icons'
import type { MessageWriter } from '@/shared/notices/messages'
import type { FilePort } from '@/app/ports/files'
import { getEntryIcon } from '../tabs/entryIcon'

/** A handler that creates a file in a vault folder under the given title. */
export type FileCreatorFn = (title: string, folder: string) => Promise<CreateResult | null>

/** Dynamic registry for domain-specific file creators. */
export interface FileCreatorRegistry {
  registerCreator(kind: string, creator: FileCreatorFn): () => void
  getCreator(kind: string): FileCreatorFn | undefined
}

/** What making a file asks of the window. */
export interface FileMakersDeps {
  readonly tabOpeners: FileOpeners
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'createFileOfKind'>
  readonly writeMessage: MessageWriter
  readonly creators?: Record<string, FileCreatorFn>
}

/** The files the window makes across registered kinds, each put in front of the person. */
export function createFileMakers({
  tabOpeners,
  windowTabs,
  writeMessage,
  creators,
}: FileMakersDeps) {
  const registeredCreators = new Map<string, FileCreatorFn>()

  const registerCreator = (kind: string, creator: FileCreatorFn): (() => void) => {
    registeredCreators.set(kind, creator)
    return () => {
      if (registeredCreators.get(kind) === creator) {
        registeredCreators.delete(kind)
      }
    }
  }

  if (creators) {
    for (const [kind, creator] of Object.entries(creators)) {
      registerCreator(kind, creator)
    }
  }

  const createGenericOfKind = async (
    kind: string,
    title: string,
    folder: string,
  ): Promise<CreateResult | null> => {
    const registered = registeredCreators.get(kind)
    if (registered) return registered(title, folder)
    const createdItem = await windowTabs.createFileOfKind(kind, folder, title)
    if (createdItem === undefined) {
      writeMessage(words.unavailable, 'error')
      return null
    }
    return createdItem.ok ? createdItem : asFailure(getKnownErrorCode(createdItem.error))
  }

  const makers = createFileCreators(
    {
      createFile: (kind, title, folder) => createGenericOfKind(kind, title, folder),
    },
    tabOpeners,
    { words: { errors: words.errors }, write: writeMessage },
  )

  return {
    ...makers,
    registerCreator,
    createFileOfKind: createGenericOfKind,
  }
}

/** What the files of the vault are listed, made and moved through. */
export interface VaultFilesDeps {
  readonly core: Pick<FilePort, 'list'>
  readonly making: NoteCreator
  readonly getTarget: () => CommandTarget
  readonly commandDeps: () => CommandDeps
}

export function createVaultFiles(deps: VaultFilesDeps): VaultFiles {
  const { core, making, getTarget, commandDeps } = deps
  const run = (id: string, target: CommandTarget, name: string) =>
    runInvocation(createInvocation(id, target, name), commandDeps(), ANSWER_WORDS)
  return {
    list: (folder) => core.list(folder),
    move: (from, to) => run('move', { ...getTarget(), path: from }, to),
    createFolder: (path) => run('createFolder', getTarget(), path),
    createNote: async (folder) => (await making.createUntitled(folder, []))?.path ?? '',
  }
}

/** What the window's commands over a row are offered and run with. */
export interface PathCommandDeps {
  readonly getTarget: () => CommandTarget
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly getContextMenuItems?: (target: ContextMenuTarget) => readonly ContextMenuRegistration[]
  readonly runContextMenu?: (id: string, target: ContextMenuTarget) => Promise<boolean>
  /** The icon a plugin draws its menu item with, and nothing for an item it draws none for. */
  readonly getContextMenuIcon?: (id: string) => unknown
  /** The icon a plugin draws its command with, and nothing for one it draws none for. */
  readonly getPluginCommandIcon?: (id: string) => unknown
  /** The icon of the reader that opens a file that is no note. */
  readonly findFileIcon?: ((path: string) => Component | undefined) | undefined
}

/** The window's commands over a row of the vault, and the icons it draws them and its entries with. */
export function createPathCommandHost(
  deps: PathCommandDeps,
): Pick<WorkspaceHost, 'getPathCommands' | 'runPathCommand' | 'getCommandIcon' | 'getEntryIcon'> {
  const {
    getTarget,
    runCommand,
    getContextMenuItems,
    runContextMenu,
    getContextMenuIcon,
    getPluginCommandIcon,
  } = deps
  return {
    getPathCommands: ({ entry, isSeveral }) => {
      const target: ContextMenuTarget = {
        path: entry.path,
        name: entry.name,
        isFolder: entry.isFolder,
        isNote: entry.isNote,
        isSeveral,
      }
      const contributions = getContextMenuItems ? getContextMenuItems(target) : []
      return createPathCommands(
        { isNote: entry.isNote, isFolder: entry.isFolder },
        isSeveral,
        contributions,
      )
    },
    runPathCommand: (id, { paths, name, isNote, isFolder }) => {
      const path = paths[0] ?? ''
      const target: ContextMenuTarget = {
        path,
        name,
        isFolder,
        isNote,
        isSeveral: paths.length > 1,
      }
      if (runContextMenu) {
        void runContextMenu(id, target).then((handled) => {
          if (!handled) {
            runCommand(id, {
              ...getTarget(),
              path,
              title: name,
              file: path,
              others: paths.slice(1),
            })
          }
        })
        return
      }
      runCommand(id, {
        ...getTarget(),
        path,
        title: name,
        file: path,
        others: paths.slice(1),
      })
    },
    getCommandIcon: (id) =>
      findIcon(id) ??
      ((getContextMenuIcon?.(id) ?? getPluginCommandIcon?.(id)) as Component | undefined) ??
      null,
    getEntryIcon: (entry, isOpen) => getEntryIcon(entry, isOpen, deps.findFileIcon),
  }
}
