/**
 * The decks and the stencils a vault holds, as the window asks for them and as
 * they come back.
 */
import { asFailure, asValue } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '@/shared/answers'
import { cardsService } from '@/shared/clients'
import type { Cards, DeckService, StencilService } from '../types'
import {
  deserializeDeck,
  deserializeStencil,
  deserializeStencilSummary,
  serializeCardToWire,
} from './serialize'

export * from '../types'
export * from './serialize'

/** Deck persistence and lifecycle service. */
export const deckService: DeckService = {
  createDeck: async (title, folder) => {
    const answer = await cardsService.createDeck({ title, path: folder })
    const error = errorIn(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  readDeck: async (path) => {
    const answer = await cardsService.readDeck({ path })
    const bound = Number(answer.bound)
    const code = staleIn(answer) ? 'changed' : errorIn(answer)
    if (code || !answer.deck) return asFailure({ code: code ?? 'notADeck', bound })
    return asValue({ deck: deserializeDeck(answer.deck), at: stamp(answer.at) ?? '' })
  },
  writeDeck: async (path, deck, seen) => {
    const answer = await cardsService.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(serializeCardToWire),
      sections: deck.sections.map((section) => ({
        name: section.name,
        preamble: section.preamble,
      })),
      tail: deck.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    const code = staleIn(answer) ? 'changed' : errorIn(answer)
    if (code) return asFailure({ code, bound: Number(answer.bound) })
    return asValue({ at: stamp(answer.at) ?? '' })
  },
}

/** Stencil metadata and schema service. */
export const stencilService: StencilService = {
  stencils: async (limit) => {
    const answer = await cardsService.listStencils({ limit: limit ?? 0 })
    return { stencils: answer.stencils.map(deserializeStencilSummary), held: answer.total }
  },
  createStencil: async (title, folder, fields) => {
    const answer = await cardsService.createStencil({ title, path: folder, fields: [...fields] })
    const error = errorIn(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  renameField: async (path, from, to, seen) => {
    const answer = await cardsService.renameStencilField({
      path,
      from,
      to,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    const code = staleIn(answer) ? 'changed' : errorIn(answer)
    if (code) return asFailure(code)
    return asValue({
      decks: answer.decks,
      cards: answer.cards,
      notWritten: answer.notWritten.map((one) => ({
        path: one.path,
        text: one.problem?.text ?? '',
      })),
      at: stamp(answer.at) ?? '',
    })
  },
  readStencil: async (path) => {
    const answer = await cardsService.readStencil({ path })
    const code = staleIn(answer) ? 'changed' : errorIn(answer)
    if (code || !answer.stencil) return asFailure(code ?? 'notAStencil')
    return asValue({ stencil: deserializeStencil(answer.stencil), at: stamp(answer.at) ?? '' })
  },
  writeStencil: async (path, fields, stencil, seen) => {
    const answer = await cardsService.writeStencil({
      path,
      fields: [...fields],
      preamble: stencil.preamble,
      faces: stencil.faces.map((face) => ({
        name: face.name,
        preamble: face.preamble,
        front: face.front,
        back: face.back,
      })),
      tail: stencil.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    const code = staleIn(answer) ? 'changed' : errorIn(answer)
    if (code) return asFailure(code)
    return asValue({ at: stamp(answer.at) ?? '' })
  },
}

/** The stencils and the decks of that vault, in the shape the window asks about them. */
export const cards: Cards = {
  ...deckService,
  ...stencilService,
}
