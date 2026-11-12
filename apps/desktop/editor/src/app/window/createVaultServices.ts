/**
 * The parts that read the vault: the note editors, the notes the commands reach, the display, the
 * vaults and the lists of the items the commands offer.
 */
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { getFileKinds } from '@/entities/file'
import { listDeclaredItemLists } from '@/entities/plugin'
import type { createTabNavigation, FileOpeners, WindowTabs } from '@/entities/tab'
import { useVaults } from '@/entities/vault'
import { createCommandNotes, createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createNoteNavigator } from '@/features/note-commands'
import { createCommandItemLists } from '@/features/plugin-commands'
import { createDisplayOptions, getVaultList, useWindowDisplay } from '@/features/window-display'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import type { MessageLog } from '@/shared/model/notices/messages'

/** What the parts that read the vault use of the tabs and the log of the window. */
export interface VaultServicesDeps {
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly tabOpeners: FileOpeners
  readonly navigation: ReturnType<typeof createTabNavigation>
}

export function createVaultServices(deps: VaultServicesDeps) {
  const { log, windowTabs, tabOpeners, navigation } = deps
  const writeMessage = log.getWriter('command')
  const noteEditors = createNoteEditors({ log, windowTabs })
  const editing = { ...noteEditors, reached: createNotes([noteEditors.store], tabOpeners) }
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
  return {
    editing,
    notes: createCommandNotes(editing.reached, windowTabs),
    window,
    vaults: useVaults({
      listVaults: getVaultList,
      log,
      coverage: window.vault.coverage,
    }),
    renames,
    vaultChanges,
    itemLists: createCommandItemLists({
      listLists: listDeclaredItemLists,
      fileKinds: getFileKinds,
    }),
    navigator: createNoteNavigator({
      navigation,
      opening: window.opening,
      windowTabs,
      writeMessage,
    }),
  }
}
