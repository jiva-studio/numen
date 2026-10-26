import { getRelationName } from './plex'
import type { PlexRelation } from './plex'

/** What a plex tab is called, on its own and after the note it relies on. */
export const WORDS = {
  plex: 'Plex',
  /** The neighbourhood is being read, and what stands around is not known yet. */
  loading: 'Loading…',
  /** What the menu offers where the picture relies on nothing. */
  newNote: 'New note',
  /** What the shape under the pointer says while notes are dragged over the picture. */
  dropName: (relation: PlexRelation) => `as ${getRelationName(relation)}`,
  /** The notes that stayed unjoined, because the vault would not write the link. */
  notJoined: 'These were not joined:',
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
  addReverseDirection: 'Reverse',
  /** Button to cycle direction. */
  changeDirection: 'Change direction',
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
  /** The commands the plugin declares: a plex opened, and a note put in front in one. */
  newPlex: 'New plex',
  travel: 'Show in plex',
  /** Menu command labels */
  read: 'Open the note',
  copy: 'Copy path',
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
  remove: 'Remove note',
}
