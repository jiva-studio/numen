/**
 * What making a file from nothing is: a deck, a stencil, a preset or a note
 * pointing at an address, made in a folder under the name it is given, and put
 * in front of the person in a tab of its own.
 */
import { formatErrorMessage } from '@numen/wire'
import type { CreateResult, ErrorCode } from '../note'
import type { MessageWriter } from '../notices/messages'
import type { FileOpeners } from './openers'

/** Which of the four a file is created as. */
export type CreateKind = 'deck' | 'stencil' | 'preset' | 'url'
export type MakeKind = CreateKind

/** What the window asks the vault to create from nothing. */
export interface VaultCreator {
  createDeck?(title: string, folder: string): Promise<CreateResult>
  createStencil?(title: string, folder: string, fields: readonly string[]): Promise<CreateResult>
  createPreset?(title: string, folder: string): Promise<CreateResult>
  createUrl?(address: string, folder: string): Promise<CreateResult>
  createURL?(address: string, folder: string): Promise<CreateResult>
}

/** What creating one of the four says in the window's voice. */
export interface CreateWords {
  /** What the vault reported as error, in words a person reads. */
  readonly errors: Record<ErrorCode, string>
}

/** What creating each of the four asks of the vault, each entry naming its own. */
const asks: Record<
  CreateKind,
  (vault: VaultCreator, folder: string, name: string, fields: readonly string[]) => Promise<CreateResult>
> = {
  deck: (vault, folder, name) => vault.createDeck!.call(vault, name, folder),
  stencil: (vault, folder, name, fields) =>
    vault.createStencil!.call(vault, name, folder, fields),
  preset: (vault, folder, name) =>
    vault.createPreset!.call(vault, name, folder),
  url: (vault, folder, name) =>
    (vault.createUrl ?? vault.createURL)!.call(vault, name, folder),
}

/**
 * A deck, a stencil, a preset or a note pointing at an address made in a
 * folder under the name it is given. The vault names the file after it and
 * answers where it stands. A vault that answers nothing at all is said here,
 * because the roads that ask for one carry no word of their own.
 */
export function fileCreators(vault: VaultCreator, puts: FileOpeners, words: CreateWords, said: MessageWriter) {
  const createFile = async (
    what: CreateKind,
    folder: string,
    name: string,
    fields: readonly string[] = [],
  ): Promise<string> => {
    try {
      const answer = await asks[what](vault, folder, name, fields)
      const error = answer.error
      if (error) {
        said(words.errors[error], 'error')
        return ''
      }
      return answer.path
    } catch (error) {
      said(formatErrorMessage(error), 'error')
      return ''
    }
  }

  /** The same, put in front of the person in a tab of its own. */
  const opens = async (
    what: MakeKind,
    folder: string,
    name: string,
    fields: readonly string[] = [],
  ): Promise<string> => {
    const path = await createFile(what, folder, name, fields)
    if (!path) return ''
    puts.made(path, '', what)
    return path
  }

  return {
    createFile,
    makes: createFile,
    decks: (folder: string, name: string) => opens('deck', folder, name),
    stencils: (folder: string, name: string, fields: readonly string[]) =>
      opens('stencil', folder, name, fields),
    presets: (folder: string, name: string) => opens('preset', folder, name),
    imports: (folder: string, address: string) => opens('url', folder, address),
  }
}

export const fileMakers = fileCreators
