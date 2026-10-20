/**
 * What making a file from nothing is: a deck, a stencil, a preset or a note
 * pointing at an address, made in a folder under the name it is given, and put
 * in front of the person in a tab of its own.
 */
import { formatErrorMessage } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { CreateResult } from '@/entities/file/@x/tab'
import type { FileOpeners } from './openers'

/** Which of the four a file is created as. */
export type CreateKind = 'deck' | 'stencil' | 'preset' | 'url'

/**
 * What the window asks the vault to create from nothing. A creator that answers
 * nothing has already told the person why none was made.
 */
export interface VaultCreator {
  createDeck?(title: string, folder: string): Promise<CreateResult | null>
  createStencil?(title: string, folder: string): Promise<CreateResult | null>
  createPreset?(title: string, folder: string): Promise<CreateResult | null>
  createUrl?(address: string, folder: string): Promise<CreateResult | null>
}

/** What creating one of the four says in the window's voice. */
export interface CreateWords {
  /** What the vault reported as error, in words a person reads. */
  readonly errors: Record<ErrorCode, string>
}

export type CreateTarget = string

/** What creating each of the four asks of the vault, each entry naming its own. */
const createByKind: Record<
  CreateKind,
  (vault: VaultCreator, folder: string, name: string) => Promise<CreateResult | null>
> = {
  deck: (vault, folder, name) => vault.createDeck!.call(vault, name, folder),
  stencil: (vault, folder, name) => vault.createStencil!.call(vault, name, folder),
  preset: (vault, folder, name) => vault.createPreset!.call(vault, name, folder),
  url: (vault, folder, name) => vault.createUrl!.call(vault, name, folder),
}

export interface FileCreatorsOptions {
  words: CreateWords
  write: MessageWriter
}

/**
 * A deck, a stencil, a preset or a note pointing at an address made in a
 * folder under the name it is given. The vault names the file after it and
 * answers where it stands. A vault that answers nothing at all is said here,
 * because the roads that ask for one carry no word of their own.
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
      const answer = await createByKind[what](vault, folder, name)
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
    createDeck: (folder: string, name: string) => createAndOpenFile('deck', folder, name),
    createStencil: (folder: string, name: string) => createAndOpenFile('stencil', folder, name),
    createPreset: (folder: string, name: string) => createAndOpenFile('preset', folder, name),
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
