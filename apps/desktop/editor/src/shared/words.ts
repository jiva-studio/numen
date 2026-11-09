/**
 * Interface text of the window.
 *
 * Text describing a tab's content belongs to the tab kind that owns it, next to
 * the code that renders it. This file holds the window's own text: tabs,
 * palette, commands, status corner, and quit.
 */
// Stryker disable StringLiteral: interface copy
import { keyChord } from '@numen/ui'
import { commandKeyChord } from '@numen/shell'
import { ERRORS } from '@numen/core-client'

export { ERRORS }

const toLowerFirst = (text: string): string => text.charAt(0).toLowerCase() + text.slice(1)

export const WORDS = {
  /** The name the title bar of every window starts with. */
  appName: 'numen',
  /** The row of a submenu or a pick list of items while they are read. */
  loadingItems: 'Loading…',
  /** Status corner text shown while something about the vault is wrong. */
  unwatched: 'Vault changes not watched',
  unread: 'Vault unreadable',
  going: 'Saving paused: the files of these notes changed on disk.',
  later: 'Later',
  /** The palette, and the three groups it draws. */
  find: 'Search the vault',
  searchVault: 'Search vault',
  openSettings: 'Open settings',
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  travel: 'Show the note',
  read: 'Open note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document here',
  noneFound: 'No results found',
  noQuery: 'Search failed',
  nothingFound: (query: string) => `Nothing found for “${query}”`,
  notEmbedded: 'Semantic index not built yet',
  /** The commands, and the groups they are drawn in. */
  noteGroupTitle: 'Note',
  overFile: 'File',
  overWindow: 'General',
  overSettings: 'Settings',
  overVault: 'Vault',
  beside: 'Open note beside',
  reveal: 'Show note',
  revealInPlex: 'Show in plex',
  revealInFiles: 'Show in file tree',
  child: 'Create child note',
  parent: 'Create parent note',
  jump: 'Create jump note',
  title: 'Rename note',
  remove: 'Move note to trash',
  copy: 'Copy note path',
  /** Status corner text shown after a file menu command has run. */
  commandRan: (title: string) => `${title}: done`,
  /** What a file is made as when the plugin that offers it is off or did not start. */
  unavailable: 'Unavailable: the plugin is disabled or stopped.',
  newNote: 'Create note',
  newFolder: 'Create folder',
  /** The name a folder is made under where none was typed. */
  untitledFolder: 'New folder',
  /** The note that points at a web address. */
  close: 'Close tab',
  byDefault: 'default',
  present: 'downloaded',
  notFetched: 'not downloaded',
  settings: 'Settings',
  findKeys: commandKeyChord(navigator.userAgent),
  /** The commands, under the second of the two keystrokes the window keeps for itself. */
  commands: 'Show commands',
  commandsKeys: keyChord('p', navigator.userAgent),
  goto: 'Go to note',
  openVault: 'Open vault',
  newVault: 'Create vault',
  /** The list the title opens to switch the vault: its name, its search field and its empty result. */
  newVaultDetail: 'Choose a folder',
  renameVault: 'Rename vault',
  removeVault: 'Remove vault from list',
  eraseVault: 'Erase vault',
  emptyTrash: 'Empty trash',
  /** Why nothing can be done to a note: the vault never opened, or none is in front. */
  noVault: 'No vault open',
  noNote: 'No active note',
  /** Steps of a command: the chip next to the field, and the field. */
  command: 'Command',
  typeCommand: 'Type a command',
  naming: 'Name',
  typeName: 'Name',
  callIt: 'Name it',
  givesIt: 'Set',
  leavesOut: 'Clear',
  typeNote: 'Search notes',
  /** One of a list the window holds: the field, and what Enter does. */
  typeItems: 'Search',
  chooses: 'Select',
  /** The vaults the installation holds, and why one of them cannot be chosen. */
  vaults: 'Vaults',
  typeVault: 'Search vaults',
  gone: 'Missing',
  folder: 'Choose vault folder',
  /** The list of vaults did not answer. */
  unlistedVaults: 'Vault list unavailable',
  /** The row a list of values opens on, which is the value in force. */
  current: 'Current',
  /** The themes could not be listed, and a theme's file could not be read. */
  unlisted: 'Themes unavailable',
  themeUnreadable: 'Theme unreadable',
  on: 'On',
  off: 'Off',
  unturned: 'Setting not saved:',
  secretFailed: 'The secret was not changed:',
  /** Plugin failures, shown as failed tasks: "<Name> stopped: <reason>", and errors raised by plugin code. */
  pluginStopped: (name: string, reason: string) => `${name} stopped: ${toLowerFirst(reason)}`,
  pluginRetryFailed: (name: string, message: string) => `Could not retry ${name}: ${message}`,
  pluginError: (name: string, message: string) => `${name}: ${message}`,
  pluginLoadFailed: 'failed to start',
  /** The settings the vault answered with could not be read. */
  unreadSettings: 'Settings unreadable. Showing the last loaded values.',
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
  typeBack: 'Type the note name to confirm',
  typeVaultBack: 'Type the vault name to confirm',
  erases: 'Erase',
  binned: 'The folder moves to the trash',
  /** The note a search did not find, offered as one to make. */
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
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
  findAction: 'Search actions',
  noAction: 'No actions found',
  wordsOnly: 'Text search only: no embedding model set',
  /** The action on a plugin task that failed for want of configuration. */
  configure: 'Configure in settings',
}
