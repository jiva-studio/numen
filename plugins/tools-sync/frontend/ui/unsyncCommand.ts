import { formatErrorMessage } from '@numen/plugin-sdk'
import type { HostServices, PluginContext } from '@numen/plugin-sdk'
import type { ReceivedSharesClient } from '../api/shares'
import type { VaultsClient } from '../api/vaults'
import { WORDS as words } from '../words'
import EndSyncDialog from './EndSyncDialog.vue'

const getCurrentVault = (host: HostServices): string =>
  host.vault.vaults?.getState().current?.id ?? ''

/**
 * Registers the command that ends the sync of the vault the file tree shows: it asks first, a synced vault stops
 * syncing and a received folder is left, and any other vault gets a notice that it is not synced. The files stay.
 */
export const registerUnsync = (
  plugin: PluginContext,
  host: HostServices,
  clients: { vaults: VaultsClient; shareInbox: ReceivedSharesClient },
): void => {
  const { vaults: vaultsClient, shareInbox: shareInboxClient } = clients
  const show = (failure: unknown): void =>
    host.ui.notices.show(formatErrorMessage(failure), 'error')
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
