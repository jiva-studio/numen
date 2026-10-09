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
  const fileAnswers = new Map<string, VaultAnswer>()
  const titles = new Map<string, string>()

  const read = async (path: string) => {
    const answer = await cards.readStencil(path)
    if (!answer.ok) {
      fileAnswers.set(path, {
        problems: [],
        reading: getFailureCode(answer.error),
        writing: null,
        at: '',
      })
      return asFailure(getFailureCode(answer.error) ?? 'notAStencil')
    }
    const stencilData = answer.value.stencil
    fileAnswers.set(path, {
      problems: stencilData.problems,
      reading: null,
      writing: null,
      at: answer.value.at,
    })
    titles.set(path, stencilData.title)
    return asValue({ body: stencilBodyOf(stencilOf(stencilData)), at: answer.value.at })
  }

  const write = async (path: string, body: string, baseline: NoteBaseline | null = null) => {
    const stencil = stencilIn(body)
    const answer = await cards.writeStencil(
      path,
      stencil.fields,
      { preamble: stencil.preamble, faces: facesOf(stencil), tail: stencil.tail },
      baseline?.fingerprint ?? null,
    )
    const currentAnswer = fileAnswers.get(path) ?? NOTHING
    fileAnswers.set(path, {
      problems: currentAnswer.problems,
      reading: currentAnswer.reading,
      writing: answer.ok ? null : getFailureCode(answer.error),
      at: answer.ok ? answer.value.at : currentAnswer.at,
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
      (fileAnswers.get(path) ?? NOTHING).at || null,
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
    const currentAnswer = fileAnswers.get(path) ?? NOTHING
    if (currentAnswer.reading !== null) {
      return currentAnswer.reading === 'notAStencil' ? words.notAStencil : words.notRead
    }
    if (currentAnswer.writing !== null) {
      return currentAnswer.writing === 'notAStencil' ? words.notAStencil : words.notSaved
    }
    return words.unreachable
  }

  const getProblems = (path: string): readonly DeckProblem[] => {
    return (fileAnswers.get(path) ?? NOTHING).problems
  }

  const getTitle = (path: string): string | undefined => {
    return titles.get(path)
  }

  const setTitle = (path: string, title: string): void => {
    titles.set(path, title)
  }

  const forget = (path: string, hasOtherTab: boolean): void => {
    if (hasOtherTab) return
    fileAnswers.delete(path)
    titles.delete(path)
  }

  const movePaths = (renames: readonly PathRename[]): void => {
    for (const rename of renames) {
      const existing = fileAnswers.get(rename.from)
      if (existing) fileAnswers.set(rename.to, existing)
      fileAnswers.delete(rename.from)
      const title = titles.get(rename.from)
      if (title !== undefined) titles.set(rename.to, title)
      titles.delete(rename.from)
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
