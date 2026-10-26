/** What the panel the deck's notes are read in says. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  reading: 'Source notes',
  nothing: 'No linked notes.',
  unreached: 'Linked notes unreadable.',
  /** Said over a note that points at the deck and not being pointed at. */
  pointsHere: 'links to this deck',
  /** Said where a name is written that no note answers to. */
  dangling: 'no note with this name',
  /** Said where several notes answer to one name and the nearest was read. */
  ambiguous: 'several notes have this name; showing the nearest',
  /** How many at the end came named and not read. */
  named: (many: number) => `${many} more linked notes not shown`,
}
