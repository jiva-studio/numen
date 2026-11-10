/**
 * Interface text of the window.
 *
 * Text describing a tab's content belongs to the tab kind that owns it, next to
 * the code that renders it. This file holds the window's own text: tabs,
 * palette, commands, status corner, and quit.
 */
// Stryker disable StringLiteral: interface copy
import { commandKeyChord } from '@numen/workbench'
import { ERRORS } from '../api/errors'

export { ERRORS }

export const WORDS = {
  /** The row of a submenu or a pick list of items while they are read. */
  loadingItems: 'Loading…',
  /** Status corner text shown while something about the vault is wrong. */
  unwatched: 'Vault changes not watched',
  unread: 'Vault unreadable',
  going: 'Saving paused: the files of these notes changed on disk.',
  later: 'Later',
  /** The palette. */
  find: 'Search the vault',
  searchVault: 'Search vault',
  openSettings: 'Open settings',
  names: 'Names',
  text: 'Text',
  travel: 'Show the note',
  read: 'Open note',
  readDocument: 'Open the document here',
  /** The commands, and the groups they are drawn in. */
  beside: 'Open note beside',
  reveal: 'Show note',
  child: 'Create child note',
  parent: 'Create parent note',
  jump: 'Create jump note',
  title: 'Rename note',
  remove: 'Move note to trash',
  copy: 'Copy note path',
  /** What a file is made as when the plugin that offers it is off or did not start. */
  unavailable: 'Unavailable: the plugin is disabled or stopped.',
  newNote: 'Create note',
  newFolder: 'Create folder',
  /** The note that points at a web address. */
  close: 'Close tab',
  settings: 'Settings',
  findKeys: commandKeyChord(navigator.userAgent),
  /** The commands, under the second of the two keystrokes the window keeps for itself. */
  commands: 'Show commands',
  goto: 'Go to note',
  openVault: 'Open vault',
  newVault: 'Create vault',
  renameVault: 'Rename vault',
  removeVault: 'Remove vault from list',
  eraseVault: 'Erase vault',
  emptyTrash: 'Empty trash',
  /** Steps of a command: the chip next to the field, and the field. */
  command: 'Command',
  naming: 'Name',
  typeName: 'Name',
  /** The vaults the installation holds, and why one of them cannot be chosen. */
  vaults: 'Vaults',
  gone: 'Missing',
  folder: 'Choose vault folder',
  /** The row a list of values opens on, which is the value in force. */
  current: 'Current',
  on: 'On',
  off: 'Off',
  asking: 'Confirm',
  several: (files: number) => `${files} files`,
  answer: 'Choose',
  retainedNotice: 'No changes',
  keepsVault: 'Keep vault',
  keepsTrash: 'Keep trash',
  emptiesTrash: 'Empty the trash of',
  removeVaultAction: 'Remove',
  stays: 'The folder is not deleted',
  exactly: 'This cannot be undone',
  typeVaultBack: 'Type the vault name to confirm',
  erases: 'Erase',
  binned: 'The folder moves to the trash',
  /** The note a search did not find, offered as one to make. */
  creates: 'Create note',
  errors: ERRORS,
  dangling: 'Broken links in:',
  undeleted: (count: number) =>
    `Not deleted: ${count} ${count === 1 ? 'item' : 'items'} in the trash`,
  /** The vault opens with no note at all. */
  nowhere: 'No start note',
  unanswered: 'Not saved: resolve the file conflict in its tab first',
  stale: 'Not saved: the note changed on disk',
  /** A file landed where something of its name is filed, and kept its place. */
  alreadyExists: 'Not moved: a file with this name exists',
  /** What the action panel of the palette is called. */
  actions: 'Actions',
  /** The action on a plugin task that failed for want of configuration. */
  configure: 'Configure in settings',
}
