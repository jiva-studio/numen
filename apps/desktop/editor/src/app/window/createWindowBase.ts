/**
 * The base of the window: what every part reads, the tabs, the parts that read the vault, the
 * settings of the window and the commands of the features.
 */
import { measureTabs } from '@/entities/tab'
import { createWindowSettings } from '@/features/settings-commands'
import { createTabCommandsDeps } from '@/features/tab-commands'
import { createTargetResolver } from '@/features/window-display'
import { WORDS as words } from '@/shared/ui/words'
import { collectFeatureCommands } from './createFeatureCommands'
import { createTabsPart } from './createTabsPart'
import { createVaultServices } from './createVaultServices'
import { createWindowFoundation } from './createWindowFoundation'
import { createWindowTabParts } from './createWindowTabParts'

export function createWindowBase() {
  const foundation = createWindowFoundation()
  const { log, rest, writeMessage, status, registry, settingsRequests } = foundation
  const tabs = createTabsPart({
    writeMessage,
    rest,
    status,
    openPluginSettings: (pluginId) => settingsRequests.emit({ pluginId }),
  })
  const { windowTabs, windowId, tabOpeners, tabLinks } = tabs
  const vault = createVaultServices({ log, windowTabs, tabOpeners, navigation: tabs.navigation })
  const { editing, notes, window, vaults, itemLists } = vault
  const tabParts = createWindowTabParts({ windowTabs, windowId, notes, writeMessage })
  const features = collectFeatureCommands(
    {
      notes: { making: editing.making, notes, navigator: vault.navigator, writeMessage },
      files: { notes, writeMessage },
      vaults: {
        tabsHold: tabParts.openTabs,
        setVaultName: vaults.setVaultName,
        reload: vaults.reload,
        writeMessage,
      },
      tabs: createTabCommandsDeps({ tabLinks, windowTabs, moves: tabParts.moves }),
      versions: { listItems: itemLists.listItems, writeMessage },
    },
    words,
  )
  registry.registerAll(features.handlers)
  return {
    foundation,
    tabs,
    vault,
    tabParts,
    appCommands: features.commands,
    getTarget: createTargetResolver(windowTabs, vaults, window),
    windowSettings: createWindowSettings({
      rest,
      connections: foundation.connections,
      words,
      log,
      onSizeChanged: () => measureTabs(windowTabs),
    }),
  }
}
