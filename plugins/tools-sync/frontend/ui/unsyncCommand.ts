import type { HostServices, PluginContext } from '@numen/plugin-sdk'
import { receivedSharesClient, vaultsClient } from '../api/wire'
import { WORDS as words } from '../words'
import EndSyncDialog from './EndSyncDialog.vue'

const getCurrentVault = (host: HostServices): string =>
  host.vault.vaults?.getState().current?.id ?? ''

/**
 * Registers the command that ends the sync of the vault the file tree shows: it asks first, a synced vault stops
 * syncing and a received folder is left, and any other vault is told it is not synced. The files stay.
 */
export const registerUnsync = (plugin: PluginContext, host: HostServices, id: string): void => {
  const show = (failure: unknown): void =>
    host.ui.notices.show(failure instanceof Error ? failure.message : String(failure), 'error')
  const ask = (kind: 'stop' | 'leave', vaultId: string): void => {
    const dialogs = host.ui.dialogs
    if (dialogs === undefined) return
    const end = (): Promise<void> =>
      kind === 'stop'
        ? vaultsClient.stopSyncingVault({ vaultId })
        : receivedSharesClient.leaveShare({ vaultId })
    const dialog = dialogs.open({
      component: EndSyncDialog,
      props: {
        kind,
        onConfirm: () => {
          dialog.close()
          end().catch(show)
        },
        onCancel: () => dialog.close(),
      },
    })
  }
  plugin.register.command({
    id,
    handler: async () => {
      const vaultId = getCurrentVault(host)
      try {
        const vault = (await vaultsClient.listVaults()).vaults.find((one) => one.id === vaultId)
        if (vault?.state === 'connected') ask('stop', vaultId)
        else if (vault?.state === 'received') ask('leave', vaultId)
        else host.ui.notices.show(words.notSyncedHere)
      } catch (failure) {
        show(failure)
      }
    },
  })
}
