/**
 * The commands of the features and what they are given of the window, registered with the handlers
 * that carry them out.
 */
import { getFileKinds } from '@/entities/file'
import { listDeclaredItemLists } from '@/entities/plugin'
import { createNoteNavigator } from '@/features/note-commands'
import { createCommandItemLists } from '@/features/plugin-commands'
import { createTabCommandsDeps } from '@/features/tab-commands'
import { createTargetResolver } from '@/features/window-display'
import { WORDS as words } from '@/shared/ui/words'
import { collectFeatureCommands } from './createFeatureCommands'
import type { WindowFoundation } from './createWindowFoundation'
import type { WindowVault } from './createWindowVault'

/** What the commands of the features use of the window. */
export interface WindowFeaturesDeps {
  readonly foundation: Pick<WindowFoundation, 'registry' | 'writeMessage'>
  readonly parts: Pick<WindowVault, 'tabsPart' | 'vault' | 'notes' | 'tabs'>
}

export function createWindowFeatures(deps: WindowFeaturesDeps) {
  const { registry, writeMessage } = deps.foundation
  const { tabsPart, vault, notes, tabs } = deps.parts
  const { windowTabs, tabLinks } = tabsPart
  const { editing, window, vaults } = vault
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: getFileKinds,
  })
  const featureCommands = collectFeatureCommands(
    {
      notes: {
        making: editing.making,
        notes,
        navigator: createNoteNavigator({
          navigation: vault.navigation,
          opening: window.opening,
          windowTabs,
          writeMessage,
        }),
        writeMessage,
      },
      files: { notes, writeMessage },
      vaults: {
        tabsHold: tabs.openTabs,
        setVaultName: vaults.setVaultName,
        reload: vaults.reload,
        writeMessage,
      },
      tabs: createTabCommandsDeps({ tabLinks, windowTabs, moves: tabs.moves }),
      versions: { listItems: itemLists.listItems, writeMessage },
    },
    words,
  )
  registry.registerAll(featureCommands.handlers)
  return {
    itemLists,
    featureCommands,
    getTarget: createTargetResolver(windowTabs, vaults, window),
  }
}

export type WindowFeatures = ReturnType<typeof createWindowFeatures>
