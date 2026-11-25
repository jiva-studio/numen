/**
 * The base of the window: what every part reads, the tabs, the parts that read the vault, the
 * settings of the window and the commands of the features.
 */
import { measureTabs } from '@/entities/tab'
import { createPickerSession } from '@/features/command-palette'
import { createTargetResolver } from '@/features/command-target'
import { createWindowSettings } from '@/features/settings-commands'
import { createTabCommandsDeps } from '@/features/tab-commands'
import { WORDS as words } from '@/shared/ui/words'
import { collectFeatureCommands } from './createFeatureCommands'
import { createTabsPart } from './createTabsPart'
import { createVaultServices } from './createVaultServices'
import { createWindowFoundation } from './createWindowFoundation'
import type { WindowApi } from './createWindowApi'
import { createWindowTabParts } from './createWindowTabParts'

export function createWindowBase(api: WindowApi) {
  const foundation = createWindowFoundation(api)
  const { log, rest, status, registry, settingsRequests } = foundation
  const writeMessage = log.getWriter('command')
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
  const picker = createPickerSession()
  const tabParts = createWindowTabParts({
    api: api.tabs,
    windowTabs,
    windowId,
    notes,
    writeMessage,
  })
  editing.fileFlush.addHandler(tabParts.openTabs.flush)
  const features = collectFeatureCommands(
    {
      notes: {
        files: api.vault.file,
        renameNote: api.vault.note.renameNote,
        making: editing.making,
        notes,
        navigator: vault.navigator,
        writeMessage,
      },
      files: { files: api.vault.file, notes, writeMessage },
      vaults: {
        vaults: api.vault.vaults,
        cloud: { known: vault.cloud, remote: api.vault.remoteVaults, ask: picker },
        clearTrash: api.vault.file.emptyTrash,
        tabsHold: tabParts.openTabs,
        setVaultName: vaults.setVaultName,
        reload: vaults.reload,
        writeMessage,
      },
      tabs: createTabCommandsDeps({ tabLinks, windowTabs, moves: tabParts.moves }),
      versions: {
        runDeclaredHandler: api.plugins.declaredCommands.runDeclaredHandler,
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
    picker,
    tabParts,
    appCommands: features.commands,
    getTarget: createTargetResolver(windowTabs, vaults, window.health),
    windowSettings: createWindowSettings({
      themes: api.configuration.themes,
      watchSettings: api.configuration.settings.watchSettings,
      rest,
      connections: foundation.connections,
      words,
      log,
      onSizeChanged: () => measureTabs(windowTabs),
    }),
  }
}
