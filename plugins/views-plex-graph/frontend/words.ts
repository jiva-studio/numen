import { getRelationName } from './plex'
import type { PlexRelation } from './plex'

/** What a plex tab is called, on its own and after the note it relies on. */
export const WORDS = {
  plex: 'Plex',
  /** The neighbourhood is being read, and its contents are not known yet. */
  loading: 'Loading…',
  /** The menu entry shown when the plex has no content. */
  newNote: 'New note',
  /** The label of the drop preview shown while notes are dragged over the plex. */
  dropName: (relation: PlexRelation) => `as ${getRelationName(relation)}`,
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
  /** What the arrow switch of a record in the inspector is called. */
  toggleArrow: 'Arrow',
  /** What the kind of link field in the inspector is called. */
  linkKind: 'Kind',
  /** The choice shown while a connection has no records and so no kind. */
  noLinkKind: 'Choose a kind',
  /** One kind of link, read as what the second note is to the first. */
  linkKindOption: (second: string, role: PlexRelation, first: string) =>
    `${second} is the ${getRelationName(role)} of ${first}`,
  /** Button to remove both records of a connection. */
  removeConnection: 'Remove link',
  /** Button to close inspector and save changes. */
  apply: 'Done',
  /** Button to cancel and close inspector without saving. */
  cancel: 'Cancel',
  /** The commands the plugin declares: a plex opened, and a note put in front in one. */
  newPlex: 'Open plex tab',
  travel: 'Show in plex',
  /** Menu command labels */
  read: 'Open note',
  copy: 'Copy path',
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
  remove: 'Remove note',
}
