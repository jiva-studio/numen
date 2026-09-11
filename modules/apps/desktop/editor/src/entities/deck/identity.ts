/**
 * The identity a deck tab and a stencil tab address a card, a section or a face
 * by, which no file carries.
 */

/** An identity something is drawn under. */
export type IdMaker = () => string

export const generateId: IdMaker = () => crypto.randomUUID()
