/**
 * What the vault last said about each deck file the window holds: what is
 * wrong with it, what a read or a write was refused for, and what it is called.
 *
 * It is filed under the path the file stands at, so a file that moved carries
 * it along and a file no tab stands at any longer lets it go.
 */
import type { ErrorCode } from '@/shared/errors'
import type { DeckProblem } from '@/entities/deck'
import { fileOf } from '@/shared/paths'
import { WORDS as words } from '@/entities/deck'

/** What the vault said about one file the last time it was read or written. */
export interface VaultAnswer {
  /**
   * What is wrong with the file, in the order the cards were read in. Which
   * card each stands on is decided against the deck the grid is drawing, so a
   * card the window is holding through a re-read keeps its mark.
   */
  readonly problems: readonly DeckProblem[]
  /** What the last read of the file encountered as error. */
  readonly reading: ErrorCode | null
  /** What the last write of it encountered as error. */
  readonly writing: ErrorCode | null
  /** The size a deck is read up to, where that is what caused the error. */
  readonly bound: number
}

const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, bound: 0 }

/** The answers one window has, about the decks that window holds. */
export function createDeckAnswers() {
  /** What the vault reports about each file, under the path it is filed at. */
  const fileAnswers = new Map<string, VaultAnswer>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()

  /** What was reported about a file, and default empty answer where nothing was. */
  const getAnswer = (path: string): VaultAnswer => fileAnswers.get(path) ?? NOTHING

  /** What a read of a file came back with, under the title it came back as. */
  const recordRead = (
    path: string,
    answer: {
      readonly problems: readonly DeckProblem[]
      readonly error?: ErrorCode | null
      readonly bound: number
      /** The title the file carries, and nothing where the read reached none. */
      readonly title: string | null
    },
  ): void => {
    fileAnswers.set(path, {
      problems: answer.problems,
      reading: answer.error ?? null,
      writing: null,
      bound: answer.bound,
    })
    if (answer.title !== null) titles.set(path, answer.title)
  }

  /** What a write of a file came back with. What the last read found stands. */
  const recordWrite = (
    path: string,
    answer: {
      readonly error?: ErrorCode | null
      readonly bound: number
    },
  ): void => {
    const currentAnswer = getAnswer(path)
    fileAnswers.set(path, {
      problems: currentAnswer.problems,
      reading: currentAnswer.reading,
      writing: answer.error ?? null,
      bound: answer.bound,
    })
  }

  /** What is wrong with a file, as the marks against its cards are made from. */
  const problemsAt = (path: string): readonly DeckProblem[] => getAnswer(path).problems

  const whyOf = (errorCode: ErrorCode | null, bound: number): string | null => {
    if (errorCode === 'deckTooLarge') return words.tooLarge(bound)
    if (errorCode === 'notADeck') return words.notADeck
    return null
  }

  /**
   * What an errored tab failed for, in words a person reads.
   */
  const getErrorMessage = (path: string, hasError: boolean): string => {
    if (!hasError) return ''
    const currentAnswer = getAnswer(path)
    if (currentAnswer.reading !== null)
      return whyOf(currentAnswer.reading, currentAnswer.bound) ?? words.notRead
    if (currentAnswer.writing !== null)
      return whyOf(currentAnswer.writing, currentAnswer.bound) ?? words.notSaved
    return words.unreachable
  }

  /** What a deck tab is called: the title the file carries, or the file itself. */
  const getTitle = (path: string): string => titles.get(path) || fileOf(path)

  /** A file the window was told the title of before any read of it answered. */
  const setTitle = (path: string, title: string): void => {
    titles.set(path, title)
  }

  /** What was recorded about a file no tab of this window holds any longer. */
  const forgetFile = (path: string): void => {
    fileAnswers.delete(path)
    titles.delete(path)
  }

  /** The same, carried to where the file was filed instead. */
  const moveFile = (from: string, to: string): void => {
    const existing = fileAnswers.get(from)
    if (existing) fileAnswers.set(to, existing)
    fileAnswers.delete(from)
    const title = titles.get(from)
    if (title !== undefined) titles.set(to, title)
    titles.delete(from)
  }

  return {
    recordRead,
    recordWrite,
    problemsAt,
    getErrorMessage,
    getTitle,
    setTitle,
    forgetFile,
    moveFile,
  }
}
