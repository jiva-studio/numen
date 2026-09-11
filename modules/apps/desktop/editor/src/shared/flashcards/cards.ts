/**
 * The decks and the stencils a vault holds, as the window asks for them and as
 * they come back.
 */
import { createClient } from '@connectrpc/connect'
import { CardsService } from '@numen/protocol'
import { transport } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '../answers'
import type { Cards } from './types'
import { carding, decked, offered, stencilled } from './serialize'

export * from './types'
export * from './serialize'

const cardsService = createClient(CardsService, transport)

/** The stencils and the decks of that vault, in the shape the window asks about them. */
export const cards: Cards = {
  stencils: async (limit) => {
    const answer = await cardsService.listStencils({ limit: limit ?? 0 })
    return { stencils: answer.stencils.map(offered), held: answer.total }
  },
  createDeck: async (title, folder) => {
    const answer = await cardsService.createDeck({ title, path: folder })
    const error = errorIn(answer)
    return { path: answer.path, error }
  },
  createStencil: async (title, folder, fields) => {
    const answer = await cardsService.createStencil({ title, path: folder, fields: [...fields] })
    const error = errorIn(answer)
    return { path: answer.path, error }
  },
  renameField: async (path, from, to, seen) => {
    const answer = await cardsService.renameStencilField({
      path,
      from,
      to,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      decks: answer.decks,
      cards: answer.cards,
      notWritten: answer.notWritten.map((one) => ({
        path: one.path,
        text: one.problem?.text ?? '',
      })),
      error: errorIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  readDeck: async (path) => {
    const answer = await cardsService.readDeck({ path })
    return {
      deck: answer.deck ? decked(answer.deck) : null,
      error: errorIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  writeDeck: async (path, deck, seen) => {
    const answer = await cardsService.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(carding),
      sections: deck.sections.map((section) => ({ name: section.name, preamble: section.preamble })),
      tail: deck.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      error: errorIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  readStencil: async (path) => {
    const answer = await cardsService.readStencil({ path })
    return {
      stencil: answer.stencil ? stencilled(answer.stencil) : null,
      error: errorIn(answer),
      at: stamp(answer.at) ?? '',
    }
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
    return {
      error: errorIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
}
