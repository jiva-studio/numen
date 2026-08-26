import { seatWord } from '@numen/ui'
import type { PlexRelatedSeat } from '@numen/ui'

/** What a plex tab is called, on its own and after the note it stands on. */
export const WORDS = {
  plex: 'Plex',
  newPlex: 'New plex',
  /** What the shape under the pointer says while a note is carried over the picture. */
  carried: (seat: PlexRelatedSeat) => `as ${seatWord(seat)}`,
}
