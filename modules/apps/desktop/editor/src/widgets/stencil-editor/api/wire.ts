/**
 * Wire adapters and vault communication for flashcard stencil tabs.
 */
import type { NoteBaseline } from '@/entities/note'
import type { ErrorCode } from '@/shared/errors'
import type { PathRename } from '@/shared/paths'
import type { Cards, DeckProblem } from '@/entities/deck'
import type { MessageWriter } from '@/shared/notices/messages'
import { ERRORS } from '@/shared/words'
import { WORDS as words } from '@/entities/deck'
import {
  facesOf,
  stencilBodyOf,
  stencilIn,
  stencilOf,
} from '../stencil'

/** What the vault said about one file the last time it was read or written. */
export interface VaultAnswer {
  readonly problems: readonly DeckProblem[]
  readonly reading: ErrorCode | null
  readonly writing: ErrorCode | null
  readonly at: string
}

export const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, at: '' }

export function createStencilWire(
  cards: Cards,
  says: MessageWriter = () => {},
) {
  const told = new Map<string, VaultAnswer>()
  const titles = new Map<string, string>()

  const read = async (path: string) => {
    const answer = await cards.readStencil(path)
    const stencil = answer.stencil ? stencilOf(answer.stencil) : null
    told.set(path, {
      problems: answer.stencil?.problems ?? [],
      reading: answer.error,
      writing: null,
      at: answer.at,
    })
    if (answer.stencil) titles.set(path, answer.stencil.title)
    if (answer.error !== null) return { body: '', error: answer.error }
    return { body: stencil ? stencilBodyOf(stencil) : '', error: null, at: answer.at }
  }

  const write = async (path: string, body: string, seen: NoteBaseline | null = null) => {
    const stencil = stencilIn(body)
    const answer = await cards.writeStencil(
      path,
      stencil.fields,
      { preamble: stencil.preamble, faces: facesOf(stencil), tail: stencil.tail },
      seen?.at ?? null,
    )
    const said = told.get(path) ?? NOTHING
    told.set(path, {
      problems: said.problems,
      reading: said.reading,
      writing: answer.error,
      at: answer.changed || answer.error !== null ? said.at : answer.at,
    })
    return { body: '', changed: answer.changed, error: answer.error, at: answer.at }
  }

  const renameField = async (
    path: string,
    field: string,
    name: string,
    onChanged: (paths: readonly string[]) => void,
  ): Promise<void> => {
    if (!name || name === field) return
    const answer = await cards.renameField(path, field, name, (told.get(path) ?? NOTHING).at || null)
    if (answer.error !== null) return says(ERRORS[answer.error], 'error')
    if (answer.changed) {
      says(words.notRenamed, 'error')
      return onChanged([path])
    }
    if (answer.cards > 0) says(words.renamed(answer.cards, answer.decks.length))
    if (answer.notWritten.length > 0) {
      says(words.notWritten(answer.notWritten.map((one) => one.path)), 'error')
    }
    onChanged([path])
  }

  const getErrorMessage = (path: string, error: ErrorCode | null): string => {
    if (error === null) return ''
    const said = told.get(path) ?? NOTHING
    if (said.reading !== null) {
      return said.reading === 'notAStencil' ? words.notAStencil : words.refused
    }
    if (said.writing !== null) {
      return said.writing === 'notAStencil' ? words.notAStencil : words.notSaved
    }
    return words.unreachable
  }

  const getProblems = (path: string): readonly DeckProblem[] => {
    return (told.get(path) ?? NOTHING).problems
  }

  const getTitle = (path: string): string | undefined => {
    return titles.get(path)
  }

  const setTitle = (path: string, title: string): void => {
    titles.set(path, title)
  }

  const forget = (path: string, hasOtherTab: boolean): void => {
    if (hasOtherTab) return
    told.delete(path)
    titles.delete(path)
  }

  const movePaths = (renamed: readonly PathRename[]): void => {
    for (const went of renamed) {
      const said = told.get(went.from)
      if (said) told.set(went.to, said)
      told.delete(went.from)
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
    }
  }

  return {
    read,
    write,
    renameField,
    getErrorMessage,
    getProblems,
    getTitle,
    setTitle,
    forget,
    movePaths,
  }
}
