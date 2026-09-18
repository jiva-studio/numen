/**
 * Wire adapters and vault communication for flashcard stencil tabs.
 */
import { asFailure, asValue } from '@numen/wire'
import type { NoteBaseline } from '@/entities/note'
import type { ErrorCode } from '@/shared/errors'
import type { PathRename } from '@/shared/paths'
import type { Cards, DeckProblem } from '@/entities/deck'
import type { MessageWriter } from '@/shared/notices/messages'
import { ERRORS } from '@/shared/words'
import { getFailureCode, WORDS as words } from '@/entities/deck'
import { facesOf, stencilBodyOf, stencilIn, stencilOf } from '../lib/stencil'

/** What the vault said about one file the last time it was read or written. */
export interface VaultAnswer {
  readonly problems: readonly DeckProblem[]
  readonly reading: ErrorCode | null
  readonly writing: ErrorCode | null
  readonly at: string
}

export const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, at: '' }

export function createStencilWire(cards: Cards, say: MessageWriter = () => {}) {
  const told = new Map<string, VaultAnswer>()
  const titles = new Map<string, string>()

  const read = async (path: string) => {
    const answer = await cards.readStencil(path)
    if (!answer.ok) {
      told.set(path, { problems: [], reading: getFailureCode(answer.error), writing: null, at: '' })
      return asFailure(getFailureCode(answer.error) ?? 'notAStencil')
    }
    const read = answer.value.stencil
    told.set(path, {
      problems: read.problems,
      reading: null,
      writing: null,
      at: answer.value.at,
    })
    titles.set(path, read.title)
    return asValue({ body: stencilBodyOf(stencilOf(read)), at: answer.value.at })
  }

  const write = async (path: string, body: string, baseline: NoteBaseline | null = null) => {
    const stencil = stencilIn(body)
    const answer = await cards.writeStencil(
      path,
      stencil.fields,
      { preamble: stencil.preamble, faces: facesOf(stencil), tail: stencil.tail },
      baseline?.at ?? null,
    )
    const said = told.get(path) ?? NOTHING
    told.set(path, {
      problems: said.problems,
      reading: said.reading,
      writing: answer.ok ? null : getFailureCode(answer.error),
      at: answer.ok ? answer.value.at : said.at,
    })
    if (!answer.ok) return asFailure(answer.error)
    return asValue({ body: '', at: answer.value.at })
  }

  const renameField = async (
    path: string,
    field: string,
    name: string,
    onChanged: (paths: readonly string[]) => void,
  ): Promise<void> => {
    if (!name || name === field) return
    const answer = await cards.renameField(
      path,
      field,
      name,
      (told.get(path) ?? NOTHING).at || null,
    )
    if (!answer.ok) {
      if (answer.error !== 'changed') return say(ERRORS[answer.error], 'error')
      say(words.notRenamed, 'error')
      return onChanged([path])
    }
    const renamed = answer.value
    if (renamed.cards > 0) say(words.renamed(renamed.cards, renamed.decks.length))
    if (renamed.notWritten.length > 0) {
      say(words.notWritten(renamed.notWritten.map((one) => one.path)), 'error')
    }
    onChanged([path])
  }

  const getErrorMessage = (path: string, error: ErrorCode | null): string => {
    if (error === null) return ''
    const said = told.get(path) ?? NOTHING
    if (said.reading !== null) {
      return said.reading === 'notAStencil' ? words.notAStencil : words.notRead
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

  const movePaths = (renames: readonly PathRename[]): void => {
    for (const went of renames) {
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
