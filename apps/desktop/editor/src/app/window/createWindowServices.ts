/**
 * The parts that read and draw the vault: the note editors, the display, the settings and the
 * vaults.
 */
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { formatError } from '@/shared/api/errors'
import { createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createWindowSettings } from '@/features/settings-commands'
import { createTabNavigation, createTabSync, measureTabs } from '@/entities/tab'
import { useVaults } from '@/entities/vault'
import { syncTabPath } from '@/features/tab-linking'
import { createDisplayOptions, getVaultList, useWindowDisplay } from '@/features/window-display'
import { createSettingsPage } from '@/pages/settings'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import { WORDS as words } from '@/shared/ui/words'
import type { WindowCore, WindowFoundation } from './windowCore'

export function createVaultServices(base: WindowFoundation) {
  const { windowTabs, tabLinks, tabOpeners } = base.tabs
  const noteEditors = createNoteEditors({ log: base.log, windowTabs })
  const editing = { ...noteEditors, reached: createNotes([noteEditors.store], tabOpeners) }

  const tabSync = createTabSync({
    windowTabs,
    tabLinks,
    sendSync: syncTabPath,
    onError: (error) => base.writeMessage(formatError(error), 'error'),
  })

  const navigation = createTabNavigation({ tabOpeners, windowTabs, tabSync })

  const renames = createEmitter<readonly PathRename[]>()
  const vaultChanges = createEmitter<VaultChangeEvent>()

  const window = useWindowDisplay(
    createDisplayOptions({
      editing,
      navigation,
      renames,
      tabOpeners,
      windowTabs,
      notifyPlugins: (event) => vaultChanges.emit(event),
      writeMessage: base.writeMessage,
    }),
  )

  const windowSettings = createWindowSettings({
    rest: base.rest,
    words,
    log: base.log,
    onSizeChanged: () => measureTabs(windowTabs),
  })

  const vaults = useVaults({
    listVaults: getVaultList,
    log: base.log,
    chunks: window.vault.chunks,
    embedded: window.vault.embedded,
    isEmbedding: window.vault.isEmbedding,
  })

  return { editing, tabSync, navigation, renames, vaultChanges, window, windowSettings, vaults }
}

export type VaultServices = ReturnType<typeof createVaultServices>

export function createWindowServices(base: WindowCore, vaultServices: VaultServices) {
  const page = createSettingsPage({
    rest: base.rest,
    appearance: vaultServices.windowSettings.appearance,
    log: base.log,
    windowTabs: base.tabs.windowTabs,
    getSettingsViews: (pluginId) =>
      base.plugins.host.pluginHost.registered.getRegisteredSettingsViews(pluginId),
    getPlatform: () => base.frame.platform.value,
  })
  const settings = { rest: base.rest, ...vaultServices.windowSettings, ...page }
  return {
    editing: vaultServices.editing,
    tabSync: vaultServices.tabSync,
    navigation: vaultServices.navigation,
    renames: vaultServices.renames,
    window: vaultServices.window,
    settings,
    vaults: vaultServices.vaults,
  }
}

export type WindowServices = ReturnType<typeof createWindowServices>
