/**
 * The commands of the features, collected in the order their groups draw them and registered with
 * the handlers that carry them out.
 */
import { runInBackground } from '@numen/ui'
import { createCommands as createPaletteCommands, PALETTE_WORDS } from '@/features/command-palette'
import { createFileCommands, type FileCommandsDeps } from '@/features/file-commands'
import { createNoteCommands, type NoteCommandsDeps } from '@/features/note-commands'
import { createTabCommands, type TabCommandsDeps } from '@/features/tab-commands'
import { createVersionCommands, type VersionCommandsDeps } from '@/features/version-commands'
import {
  createVaultCommands,
  type TabsHold,
  type VaultCommandsDeps,
} from '@/features/vault-commands'
import {
  registerCommand,
  type FeatureCommands,
  type Notes,
  type VaultRef,
} from '@/entities/command'
import type { NoteCreator } from '@/entities/note'
import type { WindowTabs } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WORDS } from '@/shared/words'

type Words = typeof WORDS

/** What the commands of the features use of the window. */
export interface FeatureCommandsDeps {
  readonly notes: NoteCommandsDeps
  readonly files: FileCommandsDeps
  readonly vaults: VaultCommandsDeps
  readonly tabs: TabCommandsDeps
  readonly versions: VersionCommandsDeps
}

/** The parts of the window the commands of the features are given. */
export interface FeatureWindow {
  readonly making: NoteCreator
  readonly notes: Notes
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly travel: (path: string) => Promise<void> | void
  readonly leavePath: (from: string, to: string) => Promise<void> | void
  readonly getOpeningNote: () => string
  readonly tabsHold: TabsHold
  readonly setVaultName: (vault: VaultRef) => void
  readonly reload: () => void
  readonly writeMessage: MessageWriter
  readonly tabs: TabCommandsDeps
  readonly listVersionItems: VersionCommandsDeps['listItems']
}

/** What each feature uses, taken from the parts of the window. */
export function createFeatureDeps(parts: FeatureWindow): FeatureCommandsDeps {
  const { notes, writeMessage } = parts
  return {
    notes: {
      making: parts.making,
      notes,
      navigator: {
        travel: async (path) => {
          await parts.travel(path)
        },
        leave: async (from, to) => {
          await parts.leavePath(from, to)
        },
        getOpeningNote: parts.getOpeningNote,
        showTab: (kind, path) =>
          runInBackground(parts.windowTabs.tabs.navigateTab(kind, path), (error) =>
            writeMessage(formatError(error), 'error'),
          ),
      },
      writeMessage,
    },
    files: { notes, writeMessage },
    vaults: {
      tabsHold: parts.tabsHold,
      setVaultName: parts.setVaultName,
      reload: parts.reload,
      writeMessage,
    },
    tabs: parts.tabs,
    versions: { listItems: parts.listVersionItems, writeMessage },
  }
}

/**
 * The rows of the note, file, palette, vault, tab and version commands, in that order, and the
 * handler of each command that has one. A group draws its commands in the order of this list.
 */
export function collectFeatureCommands(
  deps: FeatureCommandsDeps,
  words: Words,
  agent: string = navigator.userAgent,
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

/** Puts the handlers of the commands in the registry the palette carries commands out through. */
export function registerFeatureCommands(features: FeatureCommands): void {
  for (const [id, run] of Object.entries(features.handlers)) registerCommand(id, run)
}
