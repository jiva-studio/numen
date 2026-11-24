import type { HostServices, PluginContext } from '@numen/plugin-sdk'
import { shareInboxClient, vaultsClient } from '../api/wire'
import { WORDS as words } from '../words'
import EndSyncDialog from './EndSyncDialog.vue'

const getCurrentVault = (host: HostServices): string =>
  host.vault.vaults?.getState().current?.id ?? ''

/**
 * Registers the command that ends the sync of the vault the file tree shows: it asks first, a synced vault stops
 * syncing and a received folder is left, and any other vault gets a notice that it is not synced. The files stay.
 */
export const registerUnsync = (plugin: PluginContext, host: HostServices): void => {
  const show = (failure: unknown): void =>
    host.ui.notices.show(failure instanceof Error ? failure.message : String(failure), 'error')
  const ask = (kind: 'stop' | 'leave', vaultId: string, name: string): void => {
    const dialogs = host.ui.dialogs
    if (dialogs === undefined) return
    const end = (deleteCloudCopy: boolean): Promise<void> =>
      kind === 'stop'
        ? vaultsClient.stopSyncingVault({ vaultId, deleteCloudCopy })
        : shareInboxClient.leaveShare({ vaultId })
    const dialog = dialogs.open({
      component: EndSyncDialog,
      props: {
        kind,
        name,
        onConfirm: (deleteCloudCopy: boolean) => {
          dialog.close()
          end(deleteCloudCopy).catch(show)
        },
        onCancel: () => dialog.close(),
      },
    })
  }
  plugin.register.command({
    id: 'vault.stopSync',
    handler: async () => {
      const vaultId = getCurrentVault(host)
      try {
        const vault = (await vaultsClient.listVaults()).vaults.find((one) => one.id === vaultId)
        if (vault?.state === 'connected') ask('stop', vaultId, vault.name)
        else if (vault?.state === 'received') ask('leave', vaultId, vault.name)
        else host.ui.notices.show(words.notSyncedHere)
      } catch (failure) {
        show(failure)
      }
    },
  })
}
