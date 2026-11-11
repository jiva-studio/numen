/**
 * The parts that read the vault: the note editors, the tab navigation and sync, the display and the
 * vaults.
 */
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { formatError } from '@/shared/api/errors'
import { createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createTabNavigation, createTabSync, type FileOpeners, type WindowTabs } from '@/entities/tab'
import { useVaults } from '@/entities/vault'
import { syncTabPath, type TabLinksState } from '@/features/tab-linking'
import { createDisplayOptions, getVaultList, useWindowDisplay } from '@/features/window-display'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'

/** What the parts that read the vault use of the tabs and the log of the window. */
export interface VaultServicesDeps {
  readonly log: MessageLog
  readonly writeMessage: MessageWriter
  readonly windowTabs: WindowTabs
  readonly tabLinks: TabLinksState
  readonly tabOpeners: FileOpeners
}

export function createVaultServices(deps: VaultServicesDeps) {
  const { log, writeMessage, windowTabs, tabLinks, tabOpeners } = deps
  const noteEditors = createNoteEditors({ log, windowTabs })
  const editing = { ...noteEditors, reached: createNotes([noteEditors.store], tabOpeners) }

  const tabSync = createTabSync({
    windowTabs,
    tabLinks,
    sendSync: syncTabPath,
    onError: (error) => writeMessage(formatError(error), 'error'),
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
      writeMessage,
    }),
  )

  const vaults = useVaults({
    listVaults: getVaultList,
    log,
    chunks: window.vault.chunks,
    embedded: window.vault.embedded,
    isEmbedding: window.vault.isEmbedding,
  })

  return { editing, tabSync, navigation, renames, vaultChanges, window, vaults }
}
