/**
 * What the vault last said about each deck file the window holds: what is
 * wrong with it, what a read or a write was refused for, and what it is called.
 *
 * It is filed under the path the file stands at, so a file that moved carries
 * it along and a file no tab stands at any longer lets it go.
 */
import type { ErrorCode } from '@numen/wire'
import { type DeckProblem, WORDS as words } from '../../cards'
import { getFileName } from '@numen/plugin-sdk'

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

const findFailureReason = (errorCode: ErrorCode | null, bound: number): string | null => {
  if (errorCode === 'deckTooLarge') return words.tooLarge(bound)
  if (errorCode === 'notADeck') return words.notADeck
  return null
}

const formatErrorMessage = (answer: VaultAnswer, hasError: boolean): string => {
  if (!hasError) return ''
  if (answer.reading !== null)
    return findFailureReason(answer.reading, answer.bound) ?? words.notRead
  if (answer.writing !== null)
    return findFailureReason(answer.writing, answer.bound) ?? words.notSaved
  return words.unreachable
}

function moveEntry<T>(map: Map<string, T>, from: string, to: string): void {
  const value = map.get(from)
  if (value !== undefined) map.set(to, value)
  map.delete(from)
}

export interface ReadAnswer {
  readonly problems: readonly DeckProblem[]
  readonly error?: ErrorCode | null
  readonly bound: number
  readonly title: string | null
}

export interface WriteAnswer {
  readonly error?: ErrorCode | null
  readonly bound: number
}

/** The answers one window has, about the decks that window holds. */
export function createDeckAnswers() {
  const fileAnswers = new Map<string, VaultAnswer>()
  const titles = new Map<string, string>()

  const getAnswer = (path: string): VaultAnswer => fileAnswers.get(path) ?? NOTHING

  const recordRead = (path: string, answer: ReadAnswer): void => {
    fileAnswers.set(path, {
      problems: answer.problems,
      reading: answer.error ?? null,
      writing: null,
      bound: answer.bound,
    })
    if (answer.title !== null) titles.set(path, answer.title)
  }

  const recordWrite = (path: string, answer: WriteAnswer): void => {
    const currentAnswer = getAnswer(path)
    fileAnswers.set(path, {
      problems: currentAnswer.problems,
      reading: currentAnswer.reading,
      writing: answer.error ?? null,
      bound: answer.bound,
    })
  }

  const getProblems = (path: string): readonly DeckProblem[] => getAnswer(path).problems

  const getErrorMessage = (path: string, hasError: boolean): string =>
    formatErrorMessage(getAnswer(path), hasError)

  const getTitle = (path: string): string => titles.get(path) || getFileName(path)
  const setTitle = (path: string, title: string): void => {
    titles.set(path, title)
  }

  const forgetFile = (path: string): void => {
    fileAnswers.delete(path)
    titles.delete(path)
  }

  const moveFile = (from: string, to: string): void => {
    moveEntry(fileAnswers, from, to)
    moveEntry(titles, from, to)
  }

  return {
    recordRead,
    recordWrite,
    getProblems,
    getErrorMessage,
    getTitle,
    setTitle,
    forgetFile,
    moveFile,
  }
}
