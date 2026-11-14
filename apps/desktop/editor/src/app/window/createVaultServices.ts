/**
 * The parts that read the vault: the note editors, the notes the commands reach, the display, the
 * vaults and the lists of the items the commands offer.
 */
import type { VaultChangeEvent } from '@numen/plugin-loader'
import type { createTabNavigation, FileOpeners, WindowTabs } from '@/entities/tab'
import { useVaults } from '@/entities/vault'
import { createCommandNotes, createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createNoteNavigator } from '@/features/note-commands'
import { createCommandItemLists } from '@/features/plugin-commands'
import {
  createDisplayOptions,
  createVaultListReader,
  useWindowDisplay,
} from '@/features/window-display'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowApi } from './createWindowApi'

/** What the parts that read the vault use of the tabs and the log of the window. */
export interface VaultServicesDeps {
  readonly api: WindowApi
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly tabOpeners: FileOpeners
  readonly navigation: ReturnType<typeof createTabNavigation>
}

export function createVaultServices(deps: VaultServicesDeps) {
  const { api, log, windowTabs, tabOpeners, navigation } = deps
  const writeMessage = log.getWriter('command')
  const noteEditors = createNoteEditors({
    api: {
      notes: api.note,
      making: { createNote: api.note.createNote, relations: api.relations },
      quit: api.quit,
    },
    log,
    windowTabs,
  })
  const editing = {
    ...noteEditors,
    reached: createNotes([noteEditors.store], tabOpeners, writeMessage),
  }
  const listVaults = createVaultListReader({
    vaults: api.vaults,
    getShownVault: api.shownVault.getShownVault,
  })
  const renames = createEmitter<readonly PathRename[]>()
  const vaultChanges = createEmitter<VaultChangeEvent>()
  const window = useWindowDisplay(
    {
      ...api.vaultState,
      ...api.focus,
      getInitialOpenPath: api.note.getInitialOpenPath,
      watchEdits: api.note.watchEdits,
      watchTasks: api.tasks.watchTasks,
      watchPluginHealth: api.health.watchPluginHealth,
    },
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
      listVaults,
      log,
    }),
    listVaults,
    renames,
    vaultChanges,
    itemLists: createCommandItemLists({
      listLists: api.declaredCommands.listDeclaredItemLists,
      fileKinds: api.file.getFileKinds,
    }),
    navigator: createNoteNavigator({
      navigation,
      opening: window.opening,
      windowTabs,
      writeMessage,
    }),
  }
}
