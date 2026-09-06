/** What the panel the deck's notes are read in says. */
export const WORDS = {
  reading: "What these cards were written from",
  nothing: 'This deck is not joined to anything.',
  unreached: 'What this deck is joined to could not be read.',
  /** Said over a note that points at the deck rather than being pointed at. */
  pointsHere: 'points at this deck',
  /** Said where a name is written that no note answers to. */
  dangling: 'no note of that name is in the vault',
  /** Said where several notes answer to one name and the nearest was read. */
  ambiguous: 'several notes answer to that name; this is the nearest',
  /** How many at the end came named and not read. */
  named: (many: number) => `${many} more are joined to this deck and are not read here`,
}
