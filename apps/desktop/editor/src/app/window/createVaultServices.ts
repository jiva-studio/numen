/**
 * The parts that read the vault: the note editors, the notes the commands reach, the window
 * streams, the vaults and the lists of the items the commands offer.
 */
import type { VaultChangeEvent } from '@numen/plugin-loader'
import type { createTabNavigation, FileOpeners, WindowTabs } from '@/entities/tab'
import { useCloudVaults, useVaults } from '@/entities/vault'
import { createCommandNotes, createNotes } from '@/features/command-palette'
import { createNoteEditors } from '@/features/file-conflict'
import { createNoteNavigator } from '@/features/note-commands'
import { createCommandItemLists } from '@/features/plugin-commands'
import { createFocusHandlers } from '@/features/focus-requests'
import { createVaultChangeHandlers } from '@/features/vault-changes'
import { useWindowStreams } from '@/features/window-streams'
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
  const { vault } = api
  const writeMessage = log.getWriter('command')
  const vaults = useVaults({ listVaults: vault.listVaults, log })
  const cloud = useCloudVaults(vault.remoteVaults)
  void cloud.refresh()
  const noteEditors = createNoteEditors({
    api: {
      notes: vault.note,
      noteMaking: { createNote: vault.note.createNote, relations: vault.relations },
      quit: api.window.quit,
    },
    log,
    windowTabs,
    isReadOnly: () => vaults.isReadOnly.value,
  })
  const editing = {
    ...noteEditors,
    reached: createNotes([noteEditors.store], tabOpeners, writeMessage),
  }
  const renameEmitter = createEmitter<readonly PathRename[]>()
  const vaultChanges = createEmitter<VaultChangeEvent>()
  const window = useWindowStreams(
    {
      ...vault.vaultState,
      ...api.window.focus,
      getInitialOpenPath: vault.note.getInitialOpenPath,
      watchEdits: vault.note.watchEdits,
      watchTasks: api.window.tasks.watchTasks,
      watchPluginHealth: api.plugins.health.watchPluginHealth,
    },
    {
      ...createVaultChangeHandlers({
        editing,
        navigation,
        renames: renameEmitter,
        notifyPlugins: (event) => vaultChanges.emit(event),
        writeMessage,
      }),
      ...createFocusHandlers({ navigation, tabOpeners, windowTabs, writeMessage }),
    },
  )
  return {
    editing,
    notes: createCommandNotes(editing.reached, windowTabs),
    window,
    vaults,
    cloud,
    renames: renameEmitter,
    vaultChanges,
    itemLists: createCommandItemLists({
      listLists: api.plugins.declaredCommands.listDeclaredItemLists,
      readFileKinds: vault.file.getFileKinds,
    }),
    navigator: createNoteNavigator({
      navigation,
      opening: window.opening,
      windowTabs,
      writeMessage,
    }),
  }
}
