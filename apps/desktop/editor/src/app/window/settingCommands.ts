/**
 * The palette commands of the settings, made from their declarations.
 *
 * One adapter reads the settings the core declares for the owner `app` and the settings each
 * plugin declares, and offers every one flagged `palette` as a command with one choice step: a
 * boolean as On and Off, an enum as its options in force, a range as the values from its minimum
 * to its maximum by its step. A setting flagged `preview` is applied while its row is focused and
 * restored when the step is left without a choice.
 */
import type { Command, PaletteLists } from '@/features/command-palette'
import type { PluginSettingInfo, SettingEdit } from '@/entities/settings'

/** The owner whose settings the core declares. */
export const APP_OWNER = 'app'

/** An owner of settings and the settings it declares, with the values in force the core resolved. */
export interface SettingOwner {
  /** `app`, or the id of a plugin. */
  readonly id: string
  /** The name a command is prefixed with, `Name: Title`. Empty for `app`, whose commands are the title alone. */
  readonly name: string
  readonly settings: readonly PluginSettingInfo[]
}

/** The words a row is drawn with. */
export interface SettingWords {
  readonly current: string
  readonly on: string
  readonly off: string
}

/** What a setting flagged `preview` is applied and restored through. */
export interface SettingPreview {
  /** The window shows the value at the path of a setting while its row is focused. */
  apply(at: readonly string[], value: unknown): void
  /** The window shows the value in force at the path of a setting again. */
  restore(at: readonly string[]): void
}

export interface SettingPaletteDeps {
  /** The owners as they stand now; read each time a command or a list is asked for. */
  readonly getOwners: () => readonly SettingOwner[]
  readonly words: SettingWords
  readonly preview: SettingPreview
  /** Writes the settings; rejects with the reason a write is refused. */
  readonly write: (edits: readonly SettingEdit[]) => Promise<void>
}

export interface SettingPalette {
  /** The commands of the settings flagged for the palette. */
  getCommands(): readonly Command[]
  /** The rows each command offers and the preview of the focused row. */
  readonly lists: PaletteLists
  /** Writes the row chosen in the step of a command. */
  choose(command: string, item: string): Promise<void>
}

/** The id of the command of a setting: `setting:<owner>:<key>`, the key a path joined by dots. */
export const getSettingCommandId = (owner: string, key: string): string =>
  `setting:${owner}:${key}`

/** The owner and the key a command id names, and nothing for an id of another command. */
export const parseSettingCommandId = (
  id: string,
): { readonly owner: string; readonly key: string } | null => {
  const match = /^setting:([^:]+):(.+)$/.exec(id)
  return match?.[1] && match[2] ? { owner: match[1], key: match[2] } : null
}

export function createSettingPalette(deps: SettingPaletteDeps): SettingPalette {
  void deps
  return {
    getCommands: () => [],
    lists: { getStepGroups: () => [], previewItem: () => undefined },
    choose: () => Promise.resolve(),
  }
}
