/**
 * The parts that read and draw the vault: the note editors, the display, the settings and the
 * vaults.
 */
import { formatError } from '@/shared/api/errors'
import { createTabNavigation, createTabSync } from '@/entities/tab'
import { syncTabPath } from '@/features/tab-linking'
import { useWindowDisplay } from '@/features/window-display'
import { useSettings } from '@/pages/settings'
import { createEmitter } from './createEmitter'
import type { PathRename } from '@/shared/paths'
import { WORDS as words } from '@/shared/words'
import { createNoteEditors } from '../tabs/createNoteEditors'
import { createDisplayOptions } from './displayOptions'
import { useVaults } from '@/entities/vault'
import { getVaultList } from './vaultList'
import type { WindowCore } from './windowCore'

export function createWindowServices(base: WindowCore) {
  const editing = createNoteEditors({
    log: base.log,
    tabOpeners: base.tabs.tabOpeners,
    windowTabs: base.tabs.windowTabs,
  })

  const tabSync = createTabSync({
    windowTabs: base.tabs.windowTabs,
    tabLinks: base.tabs.tabLinks,
    sendSync: syncTabPath,
    onError: (error) => base.writeMessage(formatError(error), 'error'),
  })

  const navigation = createTabNavigation({
    tabOpeners: base.tabs.tabOpeners,
    windowTabs: base.tabs.windowTabs,
    tabSync,
  })

  const renames = createEmitter<readonly PathRename[]>()

  const window = useWindowDisplay(
    createDisplayOptions({
      editing,
      navigation,
      renames,
      tabOpeners: base.tabs.tabOpeners,
      windowTabs: base.tabs.windowTabs,
      notifyPlugins: (event) => base.plugins.host.pluginHost.notifyVaultChange(event),
      writeMessage: base.writeMessage,
    }),
  )

  const settings = useSettings({
    rest: base.rest,
    words,
    log: base.log,
    windowTabs: base.tabs.windowTabs,
    onSizeChanged: () => editing.noted.measureAll(),
    getSettingsViews: (pluginId) =>
      base.plugins.host.pluginHost.getRegisteredSettingsViews(pluginId),
    getPlatform: () => base.frame.platform.value,
  })

  const vaults = useVaults({
    listVaults: getVaultList,
    log: base.log,
    chunks: window.vault.chunks,
    embedded: window.vault.embedded,
    isEmbedding: window.vault.isEmbedding,
  })

  return { editing, tabSync, navigation, renames, window, settings, vaults }
}

export type WindowServices = ReturnType<typeof createWindowServices>
