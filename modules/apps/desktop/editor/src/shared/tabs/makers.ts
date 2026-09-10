/**
 * What making a file from nothing is: a deck, a stencil, a preset or a note
 * pointing at an address, made in a folder under the name it is given, and put
 * in front of the person in a tab of its own.
 */
import { troubleWords } from '@numen/wire'
import type { MakeResult, RefusalReason } from '../note'
import type { MessageWriter } from '../notices/messages'
import type { FileOpeners } from './openers'

/** Which of the four a file is made as. */
export type MakeKind = 'deck' | 'stencil' | 'preset' | 'url'

/** What the window asks the vault to make from nothing. */
export interface VaultMaker {
  /** A deck of no cards, filed in that folder under a name made from the title. */
  makeDeck(title: string, folder: string): Promise<MakeResult>
  /**
   * A stencil declaring those fields and showing no face, the same way. The
   * first field names the cards it cuts.
   */
  makeStencil(title: string, folder: string, fields: readonly string[]): Promise<MakeResult>
  /** A preset naming none of its settings, the same way. */
  makePreset(title: string, folder: string): Promise<MakeResult>
  /**
   * A note pointing at an address, named by the address. What is at it is
   * fetched afterwards, and says what the note is called from then on.
   */
  makeURL(address: string, folder: string): Promise<MakeResult>
}

/** What making one of the four says in the window's voice. */
export interface MakeWords {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<RefusalReason, string>
}

/** What making each of the four asks of the vault, each entry naming its own. */
const asks: Record<
  MakeKind,
  (vault: VaultMaker, folder: string, name: string, fields: readonly string[]) => Promise<MakeResult>
> = {
  deck: (vault, folder, name) => vault.makeDeck(name, folder),
  stencil: (vault, folder, name, fields) => vault.makeStencil(name, folder, fields),
  preset: (vault, folder, name) => vault.makePreset(name, folder),
  url: (vault, folder, name) => vault.makeURL(name, folder),
}

/**
 * A deck, a stencil, a preset or a note pointing at an address made in a
 * folder under the name it is given. The vault names the file after it and
 * answers where it stands. A vault that answers nothing at all is said here,
 * because the roads that ask for one carry no word of their own.
 */
export function fileMakers(vault: VaultMaker, puts: FileOpeners, words: MakeWords, said: MessageWriter) {
  const makes = async (
    what: MakeKind,
    folder: string,
    name: string,
    fields: readonly string[] = [],
  ): Promise<string> => {
    try {
      const answer = await asks[what](vault, folder, name, fields)
      if (answer.refusal) {
        said(words.refused[answer.refusal], 'refusal')
        return ''
      }
      return answer.path
    } catch (error) {
      said(troubleWords(error), 'refusal')
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
    const path = await makes(what, folder, name, fields)
    if (!path) return ''
    puts.made(path, '', what)
    return path
  }

  return {
    makes,
    decks: (folder: string, name: string) => opens('deck', folder, name),
    stencils: (folder: string, name: string, fields: readonly string[]) =>
      opens('stencil', folder, name, fields),
    presets: (folder: string, name: string) => opens('preset', folder, name),
    imports: (folder: string, address: string) => opens('url', folder, address),
  }
}
