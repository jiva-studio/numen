/**
 * The clients the decks, the stencils and the presets are asked through.
 */
import { createClient } from '@connectrpc/connect'
import { CardsService } from '../../gen/numen/flashcards/v1/cards_pb.js'
import { PresetsService } from '../../gen/numen/flashcards/v1/presets_pb.js'
import { transport } from '@numen/wire'

/** The decks, the cards and the stencils a vault holds. */
export const cardsService = createClient(CardsService, transport)

/** The presets a review is run under. */
export const presetsService = createClient(PresetsService, transport)
