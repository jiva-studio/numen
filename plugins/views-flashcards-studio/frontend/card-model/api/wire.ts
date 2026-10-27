/**
 * Wire adapters and vault communication for flashcard card model tabs.
 */
import { asFailure, asValue } from '@numen/wire'
import { type ErrorCode, ERRORS } from '../../cards/wire/errors'
import type { PathRename, MessageWriter } from '@numen/plugin-sdk'
import type { NoteBaseline } from '../../buffer'
import { type Cards, type DeckProblem, getFailureCode, WORDS as words } from '../../cards'
import {
  createVaultTemplates,
  serializeCardModel,
  parseCardModel,
  createBufferCardModel,
} from '../lib/card-model'

/** What the vault said about one file the last time it was read or written. */
export interface VaultAnswer {
  readonly problems: readonly DeckProblem[]
  readonly reading: ErrorCode | null
  readonly writing: ErrorCode | null
  readonly at: string
}

export const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, at: '' }

function createCardModelState() {
  const fileAnswers = new Map<string, VaultAnswer>()
  const titles = new Map<string, string>()

  const getErrorMessage = (path: string, error: ErrorCode | null): string => {
    if (error === null) return ''
    const currentAnswer = fileAnswers.get(path) ?? NOTHING
    if (currentAnswer.reading !== null) {
      return currentAnswer.reading === 'notACardModel' ? words.notACardModel : words.notRead
    }
    if (currentAnswer.writing !== null) {
      return currentAnswer.writing === 'notACardModel' ? words.notACardModel : words.notSaved
    }
    return words.unreachable
  }

  const getProblems = (path: string): readonly DeckProblem[] =>
    (fileAnswers.get(path) ?? NOTHING).problems

  const getTitle = (path: string): string | undefined => titles.get(path)
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
    fileAnswers,
    titles,
    getErrorMessage,
    getProblems,
    getTitle,
    setTitle,
    forget,
    movePaths,
  }
}

interface RenameOptions {
  currentAt: string
  say: MessageWriter
  onChanged: (paths: readonly string[]) => void
}

async function handleRenameField(
  cards: Cards,
  target: { path: string; from: string; to: string },
  options: RenameOptions,
): Promise<void> {
  const { from, to: name } = target
  if (!name || name === from) return
  const answer = await cards.renameField(target.path, { from, to: name }, options.currentAt || null)
  if (!answer.ok) {
    if (answer.error !== 'changed') return options.say(ERRORS[answer.error], 'error')
    options.say(words.notRenamed, 'error')
    return options.onChanged([target.path])
  }
  const renameSummary = answer.value
  if (renameSummary.cards > 0)
    options.say(words.renamed(renameSummary.cards, renameSummary.decks.length))
  if (renameSummary.notWritten.length > 0) {
    options.say(words.notWritten(renameSummary.notWritten.map((one) => one.path)), 'error')
  }
  options.onChanged([target.path])
}

function createCardModelReader(
  cards: Cards,
  fileAnswers: Map<string, VaultAnswer>,
  titles: Map<string, string>,
) {
  return async (path: string) => {
    const answer = await cards.readCardModel(path)
    if (!answer.ok) {
      fileAnswers.set(path, {
        problems: [],
        reading: getFailureCode(answer.error),
        writing: null,
        at: '',
      })
      return asFailure(getFailureCode(answer.error) ?? 'notACardModel')
    }
    const cardModelData = answer.value.cardModel
    fileAnswers.set(path, {
      problems: cardModelData.problems,
      reading: null,
      writing: null,
      at: answer.value.at,
    })
    titles.set(path, cardModelData.title)
    return asValue({
      body: serializeCardModel(createBufferCardModel(cardModelData)),
      at: answer.value.at,
    })
  }
}

export function createCardModelWire(cards: Cards, say: MessageWriter = () => {}) {
  const state = createCardModelState()
  const { fileAnswers, titles } = state

  const readItem = createCardModelReader(cards, fileAnswers, titles)

  const write = async (path: string, body: string, baseline: NoteBaseline | null = null) => {
    const cardModel = parseCardModel(body)
    const answer = await cards.writeCardModel(
      path,
      {
        fields: cardModel.fields,
        preamble: cardModel.preamble,
        templates: createVaultTemplates(cardModel),
        tail: cardModel.tail,
      },
      baseline?.stat ?? null,
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

  const renameField = (
    path: string,
    field: { from: string; to: string },
    onChanged: (paths: readonly string[]) => void,
  ) =>
    handleRenameField(
      cards,
      { path, from: field.from, to: field.to },
      { currentAt: (fileAnswers.get(path) ?? NOTHING).at, say, onChanged },
    )

  return {
    ...state,
    read: readItem,
    write,
    renameField,
  }
}
