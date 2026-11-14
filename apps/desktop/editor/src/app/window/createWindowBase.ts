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
import type { WindowApi } from './createWindowApi'
import { createWindowTabParts } from './createWindowTabParts'

export function createWindowBase(api: WindowApi) {
  const foundation = createWindowFoundation(api)
  const { log, rest, writeMessage, status, registry, settingsRequests } = foundation
  const tabs = createTabsPart({
    api,
    writeMessage,
    rest,
    status,
    openPluginSettings: (pluginId) => settingsRequests.emit({ pluginId }),
  })
  const { windowTabs, windowId, tabOpeners, tabLinks } = tabs
  const vault = createVaultServices({
    api,
    log,
    windowTabs,
    tabOpeners,
    navigation: tabs.navigation,
  })
  const { editing, notes, window, vaults, itemLists } = vault
  const tabParts = createWindowTabParts({ api, windowTabs, windowId, notes, writeMessage })
  const features = collectFeatureCommands(
    {
      notes: {
        files: api.file,
        renameNote: api.note.renameNote,
        making: editing.making,
        notes,
        navigator: vault.navigator,
        writeMessage,
      },
      files: { files: api.file, notes, writeMessage },
      vaults: {
        vaults: api.vaults,
        emptyTrash: api.file.emptyTrash,
        tabsHold: tabParts.openTabs,
        setVaultName: vaults.setVaultName,
        reload: vaults.reload,
        writeMessage,
      },
      tabs: createTabCommandsDeps({ tabLinks, windowTabs, moves: tabParts.moves }),
      versions: {
        runDeclaredHandler: api.declaredCommands.runDeclaredHandler,
        listItems: itemLists.listItems,
        writeMessage,
      },
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
      themes: api.themes,
      watchSettings: api.settings.watchSettings,
      rest,
      connections: foundation.connections,
      words,
      log,
      onSizeChanged: () => measureTabs(windowTabs),
    }),
  }
}
