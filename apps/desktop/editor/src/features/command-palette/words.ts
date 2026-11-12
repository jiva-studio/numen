/**
 * The palette's display strings: the commands it owns and the steps it draws.
 *
 * Contains no decisions. `types.ts` defines what a command is and
 * `lib/table.ts` lists which commands exist; this file holds only their text.
 */
// Stryker disable StringLiteral: interface copy
import type { Keystroke } from '@numen/ui'
import type { EmptyWords } from './model/search/search'

/** Everything the commands say in the window's voice. */
export interface Words extends EmptyWords {
  readonly read: string
  readonly readDocument: string
  readonly beside: string
  readonly reveal: string
  readonly travel: string
  readonly child: string
  readonly parent: string
  readonly jump: string
  readonly title: string
  readonly remove: string
  readonly copy: string
  readonly newNote: string
  readonly newFolder: string
  readonly close: string
  readonly settings: string
  readonly find: string
  readonly openSettings: string
  readonly searchVault: string
  readonly findKeys: Keystroke
  readonly goto: string
  readonly openVault: string
  readonly newVault: string
  readonly renameVault: string
  readonly removeVault: string
  readonly eraseVault: string
  readonly emptyTrash: string
  readonly noteGroupTitle: string
  readonly overFile: string
  readonly overWindow: string
  readonly overSettings?: string
  readonly overVault: string
  readonly noVault: string
  readonly noNote: string
  readonly command: string
  readonly typeCommand: string
  readonly naming: string
  readonly typeName: string
  readonly callIt: string
  readonly givesIt: string
  readonly leavesOut: string
  readonly names: string
  readonly typeNote: string
  readonly typeItems: string
  readonly chooses: string
  readonly vaults: string
  readonly typeVault: string
  readonly gone: string
  readonly current: string
  readonly asking: string
  readonly several: (files: number) => string
  readonly answer: string
  readonly retainedNotice: string
  readonly keepsVault: string
  readonly keepsTrash: string
  readonly emptiesTrash: string
  readonly removeVaultAction: string
  readonly stays: string
  readonly exactly: string
  readonly typeBack: string
  readonly typeVaultBack: string
  readonly erases: string
  readonly binned: string
  readonly creates: string
  readonly asChild: string
  readonly asParent: string
  readonly asJump: string
}

/** The words only the palette says; the window's own words are joined to them where it is built. */
export const PALETTE_WORDS = {
  meaning: 'Meaning',
  readAt: 'Open at this heading',
  noneFound: 'No results found',
  noQuery: 'Search failed',
  nothingFound: (query: string) => `No results for “${query}”`,
  notEmbedded: 'Semantic index not built yet',
  wordsOnly: 'Text search only: no embedding model set',
  noteGroupTitle: 'Note',
  overFile: 'File',
  overWindow: 'General',
  overSettings: 'Settings',
  overVault: 'Vault',
  noVault: 'No vault open',
  noNote: 'No active note',
  typeCommand: 'Type a command',
  callIt: 'Name it',
  givesIt: 'Set',
  leavesOut: 'Clear',
  typeNote: 'Search notes',
  typeItems: 'Search',
  typeVault: 'Search vaults',
  typeBack: 'Type the note name to confirm',
  chooses: 'Select',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
  findAction: 'Search actions',
  noAction: 'No actions found',
}
