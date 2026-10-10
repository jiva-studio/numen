import { seatWord } from '@numen/ui'
import type { PlexRelatedSeat } from '@numen/ui'

/** What a plex tab is called, on its own and after the note it stands on. */
export const WORDS = {
  plex: 'Plex',
  /** The neighbourhood is being read, and what stands around is not known yet. */
  loading: 'Loading…',
  /** What the menu offers where the picture stands on nothing. */
  newNote: 'New note',
  /** What the shape under the pointer says while notes are dragged over the picture. */
  dropName: (seat: PlexRelatedSeat) => `as ${seatWord(seat)}`,
  /** The notes that stayed unjoined, because the vault would not write the link. */
  notJoined: 'These were not joined:',
  /** The input placeholder in the quick-link popover. */
  quickLinkPlaceholder: 'Search or type note title…',
  /** The action to create a note with the typed title. */
  createNote: (title: string) => `Create "${title}"`,
  /** When no existing notes match the search query. */
  noMatches: 'No matching notes',
  /** The input placeholder for link description in inspector. */
  linkDescriptionPlaceholder: 'Link description…',
  /** Button to add reverse direction. */
  addReverseDirection: 'Reverse',
  /** Button to cycle direction. */
  changeDirection: 'Change direction',
  /** Button to remove a link. */
  removeLink: 'Remove link',
  /** Button to close inspector. */
  done: 'Done',
}
