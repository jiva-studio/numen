import { seatWord } from '@numen/ui'
import type { PlexRelatedSeat } from '@numen/ui'

/** What a plex tab is called, on its own and after the note it stands on. */
export const WORDS = {
  plex: 'Plex',
  /** The neighbourhood is being read, and what stands around is not known yet. */
  reading: 'Reading…',
  /** What the menu offers where the picture stands on nothing. */
  newNote: 'New note',
  /** What the shape under the pointer says while notes are dragged over the picture. */
  dropName: (seat: PlexRelatedSeat) => `as ${seatWord(seat)}`,
  /** The notes that stayed unjoined, because the vault would not write the link. */
  notJoined: 'These were not joined:',
}
