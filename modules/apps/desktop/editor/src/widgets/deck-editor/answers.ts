/**
 * What the vault last said about each deck file the window holds: what is
 * wrong with it, what a read or a write was refused for, and what it is called.
 *
 * It is filed under the path the file stands at, so a file that moved carries
 * it along and a file no tab stands at any longer lets it go.
 */
import type { ErrorCode } from '../../shared/core'
import type { DeckProblem } from '../../entities/deck/cards'
import { fileOf } from '../../shared/paths'
import { WORDS as words } from '../../entities/deck/words'

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
export function answers() {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, VaultAnswer>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()

  /** What was said about a file, and nothing said where nothing was. */
  const at = (path: string): VaultAnswer => told.get(path) ?? NOTHING

  /** What a read of a file came back with, under the title it came back as. */
  const reads = (
    path: string,
    answer: {
      readonly problems: readonly DeckProblem[]
      readonly error?: ErrorCode | null
      readonly bound: number
      /** The title the file carries, and nothing where the read reached none. */
      readonly title: string | null
    },
  ): void => {
    told.set(path, {
      problems: answer.problems,
      reading: answer.error ?? null,
      writing: null,
      bound: answer.bound,
    })
    if (answer.title !== null) titles.set(path, answer.title)
  }

  /** What a write of a file came back with. What the last read found stands. */
  const writes = (
    path: string,
    answer: {
      readonly error?: ErrorCode | null
      readonly bound: number
    },
  ): void => {
    const said = at(path)
    told.set(path, {
      problems: said.problems,
      reading: said.reading,
      writing: answer.error ?? null,
      bound: answer.bound,
    })
  }

  /** What is wrong with a file, as the marks against its cards are made from. */
  const problemsAt = (path: string): readonly DeckProblem[] => at(path).problems

  const whyOf = (errorCode: ErrorCode | null, bound: number): string | null => {
    if (errorCode === 'deckTooLarge') return words.tooLarge(bound)
    if (errorCode === 'notADeck') return words.notADeck
    return null
  }

  /**
   * What an errored tab failed for, in words a person reads. A vault that
   * answered nothing at all left the tab failed and said no word of its own.
   */
  const getErrorMessage = (path: string, hasError: boolean): string => {
    if (!hasError) return ''
    const said = at(path)
    if (said.reading !== null) return whyOf(said.reading, said.bound) ?? words.refused
    if (said.writing !== null) return whyOf(said.writing, said.bound) ?? words.notSaved
    return words.unreachable
  }

  /** What a deck tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => titles.get(path) || fileOf(path)

  /** A file the window was told the title of before any read of it answered. */
  const names = (path: string, title: string): void => {
    titles.set(path, title)
  }

  /** What was said about a file no tab of this window stands at any longer. */
  const forgets = (path: string): void => {
    told.delete(path)
    titles.delete(path)
  }

  /** The same, carried to where the file was filed instead. */
  const moved = (from: string, to: string): void => {
    const said = told.get(from)
    if (said) told.set(to, said)
    told.delete(from)
    const title = titles.get(from)
    if (title !== undefined) titles.set(to, title)
    titles.delete(from)
  }

  return { reads, writes, problemsAt, getErrorMessage, called, names, forgets, moved }
}
