/**
 * The base of the window: the log of messages, the store of the settings, the frame and the
 * registry of the commands, then the tabs, the parts that read the vault, the settings of the
 * window and the commands of the features.
 */
import { loadFrameRuntime, useWindowFrame } from '@numen/workbench'
import { createConnectionStore } from '@/entities/connection'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { measureTabs } from '@/entities/tab'
import { createTargetResolver } from '@/features/command-target'
import { createWindowPluginStatus } from '@/features/plugin-host'
import { createWindowSettings } from '@/features/settings-commands'
import { createTabCommandsDeps } from '@/features/tab-commands'
import { createEmitter } from '@/shared/lib/emitter'
import { createCommandRegistry, type SettingsRequest } from '@/shared/model/command'
import { createMessageLog } from '@/shared/model/notices/messages'
import { WORDS as words } from '@/shared/ui/words'
import { collectFeatureCommands } from './createFeatureCommands'
import { createTabsPart } from './createTabsPart'
import { createVaultServices } from './createVaultServices'
import { createWindowTabParts } from './createWindowTabParts'

export function createWindowBase() {
  const log = createMessageLog()
  const rest = createSettingsStore(SETTINGS_WORDS, log.getWriter('configured'))
  const connections = createConnectionStore(log.getWriter('connections'))
  const writeMessage = log.getWriter('command')
  const frame = useWindowFrame(loadFrameRuntime)
  const status = createWindowPluginStatus(rest)
  const registry = createCommandRegistry()
  /** The settings a part asks for: the page of a plugin and a setting of it, or the settings tab. */
  const settingsRequests = createEmitter<SettingsRequest>()
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
    log,
    rest,
    connections,
    writeMessage,
    frame,
    status,
    registry,
    settingsRequests,
    tabs,
    vault,
    tabParts,
    appCommands: features.commands,
    getTarget: createTargetResolver(windowTabs, vaults, window.health),
    windowSettings: createWindowSettings({
      rest,
      connections,
      words,
      log,
      onSizeChanged: () => measureTabs(windowTabs),
    }),
  }
}
