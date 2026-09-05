import { seatWord } from '@numen/ui'
import type { PlexRelatedSeat } from '@numen/ui'

/** What a plex tab is called, on its own and after the note it stands on. */
export const WORDS = {
  plex: 'Plex',
  newPlex: 'New plex',
  /** What the menu offers where the picture stands on nothing. */
  newNote: 'New note',
  /** What the shape under the pointer says while notes are carried over the picture. */
  carried: (seat: PlexRelatedSeat) => `as ${seatWord(seat)}`,
  /** The notes that stayed unjoined, because the vault would not write the link. */
  refused: 'These were not joined:',
}
