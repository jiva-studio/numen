/**
 * What the window says in its own voice.
 *
 * A sentence about what a tab holds is said by the kind that holds it, beside
 * the code that draws it. What is left here is the window itself: its tabs,
 * the palette, the commands, the corner, and the quit.
 */
// Stryker disable StringLiteral: interface copy
import { keyChord } from '@numen/ui'
import { commandKeyChord } from '@numen/shell'
import { ERRORS } from '@numen/plugin-sdk'

export { ERRORS }

/** The string with its first letter in lower case, for a reason placed after a colon. */
const toLowerFirst = (text: string): string => text.charAt(0).toLowerCase() + text.slice(1)

export const WORDS = {
  /** What the corner says while something about the vault is wrong. */
  unwatched: 'Vault changes not watched',
  unread: 'Vault unreadable',
  indexing: 'Indexing the vault…',
  nothingRead: 'No notes loaded',
  going: 'Saving paused: the files of these notes changed on disk.',
  later: 'Later',
  /** The palette, and the three groups it draws. */
  find: 'Search the vault',
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  travel: 'Show the note',
  read: 'Open the note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document here',
  noneFound: 'No results found',
  notAsked: 'Search failed',
  notEmbedded: 'Semantic index not built yet',
  /** The commands, and the groups they are drawn in. */
  noteGroupTitle: 'Note',
  overFile: 'File',
  overWindow: 'General',
  overVault: 'Vault',
  beside: 'Open beside',
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
  remove: 'Remove note',
  destroy: 'Destroy note',
  copy: 'Copy path',
  /** The menu the text versions of the file in front make, each entry by the plugin that does it. */
  runWith: (title: string, plugin: string) => `${title} with ${plugin}`,
  deleteVersionOf: (plugin: string) => `Delete text from ${plugin}`,
  /** What the corner says once a plugin was asked to make or delete a text. */
  textAsked: (plugin: string) => `${plugin}: working`,
  /** What a file is made as when the plugin that offers it is off or did not start. */
  unavailable: 'Unavailable: the plugin is disabled or stopped.',
  newNote: 'New note',
  /** The note that points at a web address. */
  close: 'Close tab',
  appearance: 'Change theme',
  mode: 'Light or dark',
  interfaceScale: 'Interface size',
  textScale: 'Reading font size',
  syncing: 'Sync title and filename',
  performanceProfile: 'Set performance profile',
  performance: 'Performance profile',
  eco: 'Eco',
  ecoDetail: 'Minimal resources, runs in the background',
  balanced: 'Balanced',
  balancedDetail: 'Recommended for everyday use',
  maximum: 'Maximum',
  maximumDetail: 'All hardware for indexing and models',
  byDefault: 'default',
  present: 'downloaded',
  notFetched: 'not downloaded',
  settings: 'Settings',
  findKeys: commandKeyChord(navigator.userAgent),
  /** The commands, under the second of the two keystrokes the window keeps for itself. */
  commands: 'Show commands',
  commandsKeys: keyChord('p', navigator.userAgent),
  first: 'Go to start note',
  goto: 'Go to a note',
  openVault: 'Open vault',
  newVault: 'New vault',
  newVaultDetail: 'Choose a folder',
  renameVault: 'Rename vault',
  removeVault: 'Remove from list',
  eraseVault: 'Erase vault',
  /** Why nothing can be done to a note: the vault never opened, or none is in front. */
  noVault: 'No vault open',
  noNote: 'No active note',
  /** The steps a command asks for: the chip beside the field, and the field. */
  command: 'Command',
  typeCommand: 'Type a command',
  naming: 'Name',
  typeName: 'Name',
  callIt: 'Name it',
  givesIt: 'Set',
  leavesOut: 'Clear',
  typeNote: 'Search notes',
  /** One of a list the window holds: the field, and what Enter does. */
  typeChoice: 'Search',
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
  /** The two origins the themes are drawn in, and where a person's own go. */
  shipping: 'Built-in',
  owned: 'Custom themes',
  noneOwned: 'Add .css files to numen/themes/ next to numen.json',
  /** The group the three halves are drawn in. */
  half: 'Light and dark',
  /** The three halves, and why the theme worn leaves nothing to choose. */
  system: 'System',
  light: 'Light',
  dark: 'Dark',
  pinned: 'Set by the theme',
  /** The group each of the two sizes is drawn in. */
  drawing: 'Interface size',
  setting: 'Reading size',
  /** The themes could not be listed, and a theme's file could not be read. */
  unlisted: 'Themes unavailable',
  unworn: 'Theme unreadable',
  /** The group the setting is drawn in, and the two it is. */
  syncingGroup: 'Sync title and filename',
  on: 'On',
  off: 'Off',
  unturned: 'Setting not saved:',
  secretFailed: 'Key not changed:',
  /** Plugin notices: "<Name> stopped: <reason>", and errors raised by plugin code. */
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
  removeVaultAction: 'Remove',
  stays: 'The folder is not deleted',
  exactly: 'This cannot be undone',
  typeBack: 'Type the note name to confirm',
  destroys: 'Destroy',
  forever: 'The note file is deleted',
  typeVaultBack: 'Type the vault name to confirm',
  erases: 'Erase',
  binned: 'The folder moves to the trash',
  /** The note a search did not find, offered as one to make. */
  creating: 'No results',
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
  errors: ERRORS,
  dangling: 'Broken links in:',
  /** The vault opens with no note at all. */
  nowhere: 'No start note',
  unanswered: 'Not saved: resolve the file conflict in its tab first',
  stale: 'Not saved: the note changed on disk',
  /** A file landed where something of its name is filed, and stayed where it was. */
  alreadyExists: 'Not moved: a file with this name exists',
  /** What the action panel of the palette is called. */
  actions: 'Actions',
  findAction: 'Search actions',
  noAction: 'No actions found',
  /** The corner where what is running behind the window is shown. */
  wordsOnly: 'Text search only: no embedding model set',
  isWorking: 'Background work',
  /** The status bar group of the core's own tasks. */
  coreTasks: 'Numen',
  /** The action on a plugin task that failed for want of configuration. */
  configure: 'Configure in settings',
  dismiss: 'Dismiss',
  more: 'more',
}
