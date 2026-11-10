/**
 * The commands the features declare, collected in the order the palette draws them, and the
 * handlers that carry them out.
 */
import { createCommands as createPaletteCommands, type Words } from '@/features/command-palette'
import { createFileCommands, type FileCommandsDeps } from '@/features/file-commands'
import { createNoteCommands, type NoteCommandsDeps } from '@/features/note-commands'
import { createVaultCommands, type VaultCommandsDeps } from '@/features/vault-commands'
import type { FeatureCommands } from '@/entities/command'

/** What the commands of the features use of the window. */
export interface FeatureCommandsDeps {
  readonly notes: NoteCommandsDeps
  readonly files: FileCommandsDeps
  readonly vaults: VaultCommandsDeps
}

/**
 * The rows of the note, file, palette and vault commands, in that order, and the handler of each
 * command that has one. A group draws its commands in the order of this list.
 */
export function collectFeatureCommands(
  deps: FeatureCommandsDeps,
  words: Words,
  agent: string = navigator.userAgent,
): FeatureCommands {
  const notes = createNoteCommands(deps.notes, agent)
  const files = createFileCommands(deps.files)
  const vaults = createVaultCommands(deps.vaults, agent)
  return {
    commands: [
      ...notes.commands,
      ...files.commands,
      ...createPaletteCommands(words, agent),
      ...vaults.commands,
    ],
    handlers: { ...notes.handlers, ...files.handlers, ...vaults.handlers },
  }
}
