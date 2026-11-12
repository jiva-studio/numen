/**
 * The commands of the features, collected in the order their groups draw them and registered with
 * the handlers that carry them out.
 */
import { createCommands as createPaletteCommands, PALETTE_WORDS } from '@/features/command-palette'
import { createFileCommands, type FileCommandsDeps } from '@/features/file-commands'
import { createNoteCommands, type NoteCommandsDeps } from '@/features/note-commands'
import { createTabCommands, type TabCommandsDeps } from '@/features/tab-commands'
import { createVersionCommands, type VersionCommandsDeps } from '@/features/version-commands'
import { createVaultCommands, type VaultCommandsDeps } from '@/features/vault-commands'
import type { FeatureCommands } from '@/entities/command'
import type { WORDS } from '@/shared/ui/words'
import { getUserAgent } from './getUserAgent'

/** What the commands of the features use of the window. */
export interface FeatureCommandsDeps {
  readonly notes: NoteCommandsDeps
  readonly files: FileCommandsDeps
  readonly vaults: VaultCommandsDeps
  readonly tabs: TabCommandsDeps
  readonly versions: VersionCommandsDeps
}

/**
 * The rows of the note, file, palette, vault, tab and version commands, in that order, and the
 * handler of each command that has one. A group draws its commands in the order of this list.
 */
export function collectFeatureCommands(
  deps: FeatureCommandsDeps,
  words: typeof WORDS,
  agent: string = getUserAgent(),
): FeatureCommands {
  const notes = createNoteCommands(deps.notes, agent)
  const files = createFileCommands(deps.files)
  const vaults = createVaultCommands(deps.vaults, agent)
  const tabs = createTabCommands(deps.tabs)
  const versions = createVersionCommands(deps.versions)
  return {
    commands: [
      ...notes.commands,
      ...files.commands,
      ...createPaletteCommands({ ...words, ...PALETTE_WORDS }, agent),
      ...vaults.commands,
      ...tabs.commands,
      ...versions.commands,
    ],
    handlers: {
      ...notes.handlers,
      ...files.handlers,
      ...vaults.handlers,
      ...tabs.handlers,
      ...versions.handlers,
    },
  }
}
