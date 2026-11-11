/**
 * The parts that read and draw the vault: the note editors, the display, the settings and the
 * vaults.
 */
import { formatError } from '@/shared/api/errors'
import { createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createTabNavigation, createTabSync, measureTabs } from '@/entities/tab'
import { useVaults } from '@/entities/vault'
import { syncTabPath } from '@/features/tab-linking'
import { createDisplayOptions, getVaultList, useWindowDisplay } from '@/features/window-display'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import { WORDS as words } from '@/shared/ui/words'
import { createWindowSettingsPart } from './windowSettings'
import type { WindowCore } from './windowCore'

export function createWindowServices(base: WindowCore) {
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

  const window = useWindowDisplay(
    createDisplayOptions({
      editing,
      navigation,
      renames,
      tabOpeners,
      windowTabs,
      notifyPlugins: (event) => base.plugins.host.pluginHost.notifyVaultChange(event),
      writeMessage: base.writeMessage,
    }),
  )

  const settings = createWindowSettingsPart({
    rest: base.rest,
    words,
    log: base.log,
    windowTabs,
    onSizeChanged: () => measureTabs(windowTabs),
    getSettingsViews: (pluginId) =>
      base.plugins.host.pluginHost.registered.getRegisteredSettingsViews(pluginId),
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
