/**
 * The commands the features declare, collected in the order the palette draws them, and the
 * handlers that carry them out.
 */
import {
  createCommands as createPaletteCommands,
  registerCommand,
} from '@/features/command-palette'
import type { Words } from '@/features/command-palette'
import { createFileCommands, type FileCommandsDeps } from '@/features/file-commands'
import { createNoteCommands, type NoteCommandsDeps } from '@/features/note-commands'
import { createTabCommands, type TabCommandsDeps } from '@/features/tab-commands'
import { getAllTabs, getEligibleTabs, getWindowName } from '@/features/tab-linking'
import { createVersionCommands, type VersionCommandsDeps } from '@/features/version-commands'
import {
  createVaultCommands,
  type TabsHold,
  type VaultCommandsDeps,
} from '@/features/vault-commands'
import type { FeatureCommands, Notes, VaultRef } from '@/entities/command'
import type { WindowTabs } from '@/entities/tab'
import type { NoteCreator } from '@/entities/note'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import { runInBackground } from '@numen/ui'

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
  readonly windowTabs: Pick<WindowTabs, 'navigateTab'>
  readonly travel: (path: string) => Promise<void> | void
  readonly leavePath: (from: string, to: string) => Promise<void> | void
  readonly getOpeningNote: () => string
  readonly tabsHold: TabsHold
  readonly setVaultName: (vault: VaultRef) => void
  readonly reload: () => void
  readonly writeMessage: MessageWriter
  readonly tabs: Omit<TabCommandsDeps, 'listing'>
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
          runInBackground(parts.windowTabs.navigateTab(kind, path), (error) =>
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
    tabs: { ...parts.tabs, listing: { getAllTabs, getEligibleTabs, getWindowName } },
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
      ...createPaletteCommands(words, agent),
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
