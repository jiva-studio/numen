/**
 * A file of a kind a tab makes, made in a folder under the name it is given and
 * put in front of the person in a tab of its own.
 */
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { CreateResult } from '@/entities/file/@x/tab'
import type { FileOpeners } from './openers'

/** The kind of tab a file is created as. */
export type CreateKind = string

/**
 * What the window asks the vault to create from nothing. A creator that answers
 * nothing has already told the person why none was made.
 */
export interface VaultCreator {
  createFile(kind: string, title: string, folder: string): Promise<CreateResult | null>
}

/** What creating a file says in the window's voice. */
export interface CreateWords {
  /** What the vault reported as error, in words a person reads. */
  readonly errors: Record<ErrorCode, string>
}

export type CreateTarget = string

export interface FileCreatorsOptions {
  words: CreateWords
  write: MessageWriter
}

/**
 * A file of a kind, made in a folder under the name it is given. The vault names
 * the file after it and answers at its current position. A vault that answers
 * nothing at all is said here.
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

  /** The same, put in front of the person in a tab of its own. */
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
