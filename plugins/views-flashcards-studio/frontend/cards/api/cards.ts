/**
 * The decks and the card models a vault holds, as the window requests them and as
 * they are returned.
 */
import { asFailure, asValue } from '@numen/plugin-sdk'
import { formatFingerprint, parseFingerprint } from '../../api/fingerprint'
import { findErrorCode, isStaleAnswer } from '../wire/errors'
import { cardsService } from './clients'
import type { Cards, DeckService, CardModelService } from '../types'
import {
  deserializeDeck,
  deserializeCardModel,
  deserializeCardModelSummary,
  serializeCardToWire,
} from './serialize'

export * from '../types'
export * from './serialize'

/** Deck persistence and lifecycle service. */
export const deckService: DeckService = {
  createDeck: async (title, folder) => {
    const answer = await cardsService.createDeck({ title, path: folder })
    const error = findErrorCode(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  readDeck: async (path) => {
    const answer = await cardsService.readDeck({ path })
    const linked = answer.bound
    const code = isStaleAnswer(answer) ? 'changed' : findErrorCode(answer)
    if (code || !answer.deck) return asFailure({ code: code ?? 'notADeck', bound: linked })
    return asValue({ deck: deserializeDeck(answer.deck), at: formatFingerprint(answer.at) ?? '' })
  },
  writeDeck: async (path, deck, observed) => {
    const answer = await cardsService.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(serializeCardToWire),
      sections: deck.sections.map((section) => ({
        name: section.name,
        preamble: section.preamble,
      })),
      tail: deck.tail,
      ...(observed === null ? {} : { baseline: parseFingerprint(observed) }),
    })
    const code = isStaleAnswer(answer) ? 'changed' : findErrorCode(answer)
    if (code) return asFailure({ code, bound: answer.bound })
    return asValue({ at: formatFingerprint(answer.at) ?? '' })
  },
}

/** Card model metadata and schema service. */
export const cardModelService: CardModelService = {
  cardModels: async (limit) => {
    const answer = await cardsService.listCardModels({ limit: limit ?? 0 })
    return { cardModels: answer.cardModels.map(deserializeCardModelSummary), held: answer.total }
  },
  createCardModel: async (title, folder, fields) => {
    const answer = await cardsService.createCardModel({ title, path: folder, fields: [...fields] })
    const error = findErrorCode(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  renameField: async (path, { from, to }, observed) => {
    const answer = await cardsService.renameCardModelField({
      path,
      from,
      to,
      ...(observed === null ? {} : { baseline: parseFingerprint(observed) }),
    })
    const code = isStaleAnswer(answer) ? 'changed' : findErrorCode(answer)
    if (code) return asFailure(code)
    return asValue({
      decks: answer.decks,
      cards: answer.cards,
      notWritten: answer.notWritten.map((one) => ({
        path: one.path,
        text: one.problem?.text ?? '',
      })),
      at: formatFingerprint(answer.at) ?? '',
    })
  },
  readCardModel: async (path) => {
    const answer = await cardsService.readCardModel({ path })
    const code = isStaleAnswer(answer) ? 'changed' : findErrorCode(answer)
    if (code || !answer.cardModel) return asFailure(code ?? 'notACardModel')
    return asValue({
      cardModel: deserializeCardModel(answer.cardModel),
      at: formatFingerprint(answer.at) ?? '',
    })
  },
  writeCardModel: async (path, cardModel, observed) => {
    const answer = await cardsService.writeCardModel({
      path,
      fields: [...cardModel.fields],
      preamble: cardModel.preamble,
      templates: cardModel.templates.map((template) => ({
        name: template.name,
        preamble: template.preamble,
        front: template.front,
        back: template.back,
      })),
      tail: cardModel.tail,
      ...(observed === null ? {} : { baseline: parseFingerprint(observed) }),
    })
    const code = isStaleAnswer(answer) ? 'changed' : findErrorCode(answer)
    if (code) return asFailure(code)
    return asValue({ at: formatFingerprint(answer.at) ?? '' })
  },
}

/** The card models and the decks of that vault, in the shape the window requests them. */
export const cards: Cards = {
  ...deckService,
  ...cardModelService,
}
