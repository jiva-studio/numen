/**
 * pointing at an address, made in a folder under the name it is given, and put
 * in front of the person in a tab of its own.
 */
import { formatErrorMessage } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { CreateResult } from '@/entities/file/@x/tab'
import type { FileOpeners } from './openers'

/** What a file is created as: a note pointing at an address, or a kind that makes files. */
export type CreateKind = 'url' | (string & {})

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
 * A file of a kind, or a note pointing at an address, made in a
 * folder under the name it is given. The vault names the file after it and
 * answers at its current position. A vault that answers nothing at all is said here,
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
    createUrl: (folder: string, address: string) => createAndOpenFile('url', folder, address),
    createUrls: async (
      folder: string,
      addresses: readonly string[],
    ): Promise<readonly string[]> => {
      if (addresses.length === 0) return []
      if (addresses.length === 1 && addresses[0]) {
        const path = await createAndOpenFile('url', folder, addresses[0])
        return path ? [path] : []
      }
      const paths: string[] = []
      for (const address of addresses) {
        const path = await createFile('url', folder, address)
        if (path) paths.push(path)
      }
      return paths
    },
  }
}
