/**
 * Palette types: the context the commands run over. The `Command` type is in `entities/command`,
 * the engine in `shared/model/command`.
 *
 * This file defines types only. The commands the palette owns are `lib/commands/table.ts`, display
 * words are `words.ts`, and the services
 * available while a command runs are `model/deps.ts`.
 */
import type { VaultList } from '@/entities/vault'
import type { NameMatch } from './model/search/search'
import type { StepGroup } from '@/shared/model/command'

export type { StepRow } from '@/shared/model/command'

/** What the commands ask of the application before anything is chosen. */
export interface CommandsDeps {
  /** The names in the vault that match. */
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  /** Every vault the installation holds, and which of them this window shows. */
  vaults(): Promise<VaultList>
}

/**
 * The lists the window holds, and what it does with the entry under the keyboard
 * selection. A list is re-read every time the step is drawn, so its contents
 * may change while the step is open.
 */
export interface PaletteLists {
  /**
   * What this command offers now, in the groups it is drawn in. The words typed
   * come too: a list may hold a row made out of them.
   */
  getStepGroups(command: string, typed: string): readonly StepGroup[]
  /** The one the keyboard is standing on, and nothing where it relies on none. */
  previewItem(command: string, item: string): void
}

/**
 * What the window records about a note by its file name. The step stays open
 * while the vault changes, so this is re-read each time the step is drawn and
 * once more when the command is invoked.
 */
export interface NoteLookup {
  /** What it is called now, and nothing where the window names it nothing. */
  getTitle(path: string): string
  /** The identity of the tab holding it, and nothing where none holds it. */
  getTabAt(path: string): string | null
}

/**
 * One store of open files, as accessed by commands. Each note-type editor
 * keeps one.
 */
export interface Store {
  /** Whether this store holds a file open under that identity. */
  has(id: string): boolean
  getPath(id: string): string
  /** What it is called, under the identity it opened under. */
  getTitle(id: string): string
  asking(id: string): boolean
  /** Answers once nothing of it is on its way to the file. */
  flushPending(id: string): Promise<void>
  /** The tab holding it releases it. */
  close(id: string): void
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
}
