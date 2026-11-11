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
import type { createWindowTabMover } from '@/features/tab-move'
import type { FeatureCommands } from '@/entities/command'
import type { Notes } from '@/shared/model/command'
import { formatError } from '@/shared/api/errors'
import type { WORDS } from '@/shared/ui/words'
import { getUserAgent } from './getUserAgent'
import type { CommandItemLists } from '@/features/plugin-commands'
import type { VaultServices } from './createWindowServices'
import type { WindowFoundation } from './windowCore'

/** What the commands of the features use of the window. */
interface FeatureCommandsDeps {
  readonly notes: NoteCommandsDeps
  readonly files: FileCommandsDeps
  readonly vaults: VaultCommandsDeps
  readonly tabs: TabCommandsDeps
  readonly versions: VersionCommandsDeps
}

/** The parts of the window the commands of the features are given. */
interface FeatureWindow {
  readonly base: Pick<WindowFoundation, 'writeMessage' | 'tabs'>
  readonly itemLists: CommandItemLists
  readonly services: Pick<VaultServices, 'editing' | 'window' | 'navigation' | 'vaults'>
  readonly moves: ReturnType<typeof createWindowTabMover>
  readonly tabsHold: TabsHold
  readonly notes: Notes
  readonly tabLinks: WindowFoundation['tabs']['tabLinks']
}

export function createFeatureDeps(parts: FeatureWindow): FeatureCommandsDeps {
  const { base, itemLists, services, moves, notes } = parts
  const { writeMessage } = base
  const { windowTabs } = base.tabs
  const { editing, window, navigation, vaults } = services
  return {
    notes: {
      making: editing.making,
      notes,
      navigator: {
        travel: async (path) => {
          await navigation.travel(path)
        },
        leave: async (from, to) => {
          await navigation.leavePath(from, to)
        },
        getOpeningNote: () => window.opening.path.value,
        showTab: (kind, path) =>
          runInBackground(windowTabs.tabs.navigateTab(kind, path), (error) =>
            writeMessage(formatError(error), 'error'),
          ),
      },
      writeMessage,
    },
    files: { notes, writeMessage },
    vaults: {
      tabsHold: parts.tabsHold,
      setVaultName: vaults.setVaultName,
      reload: vaults.reload,
      writeMessage,
    },
    tabs: {
      tabLinks: parts.tabLinks,
      getTabs: () => windowTabs.tabs.tabs.value,
      getKind: (id) => windowTabs.tabs.getTab(id)?.kind.kind,
      isLinkable: (id) => windowTabs.tabs.getTab(id)?.kind.address?.navigate !== undefined,
      windowId: moves.windowId,
      isAsking: moves.isTabAsking,
      detach: moves.detachTabInPlace,
      move: moves.moveTab,
    },
    versions: { listItems: itemLists.listItems, writeMessage },
  }
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
