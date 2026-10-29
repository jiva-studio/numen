/**
 * A file of a kind a tab creates, created in a folder under the given name and
 * opened in a tab of its own.
 */
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { CreateResult } from '@/shared/model/fileTypes'
import type { FileOpeners } from './openers'

/** The tab kind a file is created as. */
export type CreateKind = string

/**
 * What the window asks the vault to create from nothing. A creator that returns
 * nothing has already reported the failure to the user.
 */
export interface VaultCreator {
  createFile(kind: string, title: string, folder: string): Promise<CreateResult | null>
}

/** What creating a file reports to the window. */
export interface CreateWords {
  /** The vault's reported error, in user-readable words. */
  readonly errors: Record<ErrorCode, string>
}

export type CreateTarget = string

export interface FileCreatorsOptions {
  words: CreateWords
  write: MessageWriter
}

/**
 * A file of a kind, created in a folder under the given name. The vault names
 * the file after it and reports its current position. A vault that returns
 * nothing is reported here.
 */
export function createFileCreators(
  vault: VaultCreator,
  tabOpeners: FileOpeners,
  options: FileCreatorsOptions,
) {
  const { words, write } = options

  const createFile = async (
    what: CreateKind,
    folder: string,
    name: CreateTarget,
  ): Promise<string> => {
    try {
      const answer = await vault.createFile(what, name, folder)
      if (answer === null) return ''
      if (!answer.ok) {
        write(words.errors[answer.error], 'error')
        return ''
      }
      return answer.value.path
    } catch (error) {
      write(formatErrorMessage(error), 'error')
      return ''
    }
  }

  /** The same, opened in a tab of its own. */
  const createAndOpenFile = async (
    what: CreateKind,
    folder: string,
    name: CreateTarget,
  ): Promise<string> => {
    const path = await createFile(what, folder, name)
    if (!path) return ''
    tabOpeners.openNewFile(path, '', what)
    return path
  }

  return {
    createFile,
    createAndOpenFile,
  }
}
