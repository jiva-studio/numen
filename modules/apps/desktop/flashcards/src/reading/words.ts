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

/** Why a note has no text here, in words a person reads. */
export const REFUSED = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notAPreset: 'that note is not a preset',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this reads',
  unreadable: 'the frontmatter of that note cannot be read',
} as const

export type Refused = keyof typeof REFUSED
