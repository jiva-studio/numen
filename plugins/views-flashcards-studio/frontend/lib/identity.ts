/**
 * The identity a deck tab and a card model tab address a card, a section or a template
 * by, which no file carries.
 */

/** An identity something is drawn under. */
export type IdMaker = () => string

export const generateId: IdMaker = () => crypto.randomUUID()
