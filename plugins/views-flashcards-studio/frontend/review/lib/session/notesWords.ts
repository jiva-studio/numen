/** The text of the panel in which the deck's notes are shown. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  reading: 'Source notes',
  nothing: 'No linked notes.',
  unreached: 'Linked notes unreadable.',
  /** Shown on a note that links to the deck and is not linked from it. */
  pointsHere: 'links to this deck',
  /** Shown for a link name that no note resolves. */
  dangling: 'no note with this name',
  /** Shown where several notes match one name and the nearest was read. */
  ambiguous: 'several notes have this name; showing the nearest',
  /** How many at the end came named and not read. */
  named: (many: number) => `${many} more linked notes not shown`,
}
