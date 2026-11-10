/**
 * The files of the vault as a plugin's tabs are given them: the folders listed,
 * the files made and moved, and the icons the window draws commands and entries with.
 */
import type { Component } from 'vue'
import type { VaultFiles, WorkspaceHost } from '@numen/plugin-loader'
import { getKnownErrorCode, asFailure } from '@numen/core-client'
import { createFileCreators, type FileOpeners, type useWindowTabs } from '@/entities/tab'
import { listFiles, type CreateResult } from '@/entities/file'
import {
  ANSWER_WORDS,
  createInvocation,
  runInvocation,
  type CommandDeps,
} from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { NoteCreator } from '@/entities/note'
import { COMMAND } from '@/shared/commandIds'
import { WORDS as words } from '@/shared/words'
import { findIcon } from '@/shared/icons'
import type { MessageWriter } from '@/shared/notices/messages'
import { getEntryIcon } from '../tabs/entryIcon'

/** A handler that creates a file in a vault folder under the given title. */
export type FileCreatorFn = (title: string, folder: string) => Promise<CreateResult | null>

/** Dynamic registry for domain-specific file creators. */
export interface FileCreatorRegistry {
  registerCreator(kind: string, creator: FileCreatorFn): () => void
  getCreator(kind: string): FileCreatorFn | undefined
}

/** The window operations needed to create a file. */
export interface FileMakersDeps {
  readonly tabOpeners: FileOpeners
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'createFileOfKind'>
  readonly writeMessage: MessageWriter
  readonly creators?: Record<string, FileCreatorFn>
}

/** The files the window creates across registered kinds, each opened in a tab. */
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
  readonly making: NoteCreator
  readonly getTarget: () => CommandTarget
  readonly commandDeps: () => CommandDeps
}

export function createVaultFiles(deps: VaultFilesDeps): VaultFiles {
  const { making, getTarget, commandDeps } = deps
  const run = async (id: string, target: CommandTarget, name: string): Promise<void> => {
    await runInvocation(createInvocation(id, target, name), commandDeps(), ANSWER_WORDS)
  }
  return {
    list: (folder) => listFiles(folder),
    move: (from, to) => run(COMMAND.moveFile, { ...getTarget(), path: from }, to),
    createFolder: (path) => run(COMMAND.createFolder, getTarget(), path),
    createNote: (folder) => making.createUntitled(folder),
  }
}

/** What the window finds the icons of commands and files with. */
export interface IconHostDeps {
  readonly getPluginCommandIcon?: (id: string) => unknown
  readonly findFileIcon?: ((path: string) => Component | undefined) | undefined
}

/** The icons the window draws commands and the entries of the vault with. */
export function createIconHost(
  deps: IconHostDeps,
): Pick<WorkspaceHost, 'getCommandIcon' | 'getEntryIcon'> {
  return {
    getCommandIcon: (id) =>
      (deps.getPluginCommandIcon?.(id) as Component | undefined) ?? findIcon(id) ?? null,
    getEntryIcon: (entry, isOpen) => getEntryIcon(entry, isOpen, deps.findFileIcon),
  }
}
