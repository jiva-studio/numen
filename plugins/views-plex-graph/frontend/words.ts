import type { PlexRelation } from './types'

/** What a plex tab is called, on its own and after the note it relies on. */
export const WORDS = {
  plex: 'Plex',
  /** The neighborhood is being read, and its contents are not known yet. */
  loading: 'Loading…',
  /** The row of a command of the menu whose items are being read. */
  loadingItems: 'Loading…',
  /** The label of the drop preview shown while notes are dragged over the plex. */
  dropName: (relation: PlexRelation) => `as ${relation}`,
  /** The notes that stayed unjoined, because the vault would not write the link. */
  notJoined: 'Links not created:',
  /** What the field in the quick-link popover is called. */
  quickLinkLabel: 'Note to link',
  /** The input placeholder in the quick-link popover. */
  quickLinkPlaceholder: 'Search or type note title…',
  /** The action to create a note with the typed title. */
  createNote: (title: string) => `Create "${title}"`,
  /** When no existing notes match the search query. */
  noMatches: 'No matching notes',
  /** What the link description field in the inspector is called. */
  linkDescription: 'Link description',
  /** The input placeholder for link description in inspector. */
  linkDescriptionPlaceholder: 'Link description…',
  /** Button to add reverse direction. */
  addReverseDirection: 'Add reverse direction',
  /** What the arrow of a record is, by the titles of the notes it joins: where it points, or that it has none. */
  direction: {
    none: (_first: string, _second: string) => 'No arrow',
    forward: (_first: string, second: string) => `Arrow to ${second}`,
    backward: (first: string, _second: string) => `Arrow to ${first}`,
  },
  /** What the direction button of a record says: the arrow it shows and the one a click makes. */
  directionButton: (now: string, next: string) => `${now}. Click for: ${next}`,
  /** Button to remove a link. */
  removeLink: 'Remove link',
  /** Button to add link when none exist. */
  addLink: 'Add link',
  /** Notice shown when all links between nodes are deleted. */
  linkWillBeRemoved: 'The link between these notes will be removed.',
  /** Button to close inspector and save changes. */
  apply: 'Done',
  /** Button to cancel and close inspector without saving. */
  cancel: 'Cancel',
}
