import {
  definePlugin,
  SETTINGS_OPEN_COMMAND,
  type HostServices,
  type PluginContext,
} from '@numen/plugin-sdk'
import {
  acceptInvitationClient,
  folderSharesClient,
  shareInvitationClient,
  vaultsClient,
} from './api/wire'
import { createAcceptanceView } from './model/acceptance'
import { createInvitationView } from './model/invitation'
import { createSyncState } from './model/sync'
import { createSyncedFlags } from './model/syncedFlags'
import { finishSync, startSync } from './model/syncVault'
import { createSubscriptionView } from './ui/subscriptionView'
import { createSharesPage } from './ui/sharesPage'
import { registerUnsync } from './ui/unsyncCommand'
import AcceptInvitationDialog from './ui/AcceptInvitationDialog.vue'
import GetVaultDialog from './ui/GetVaultDialog.vue'
import ShareInvitationDialog from './ui/ShareInvitationDialog.vue'
import SyncChoiceDialog from './ui/SyncChoiceDialog.vue'
import { createSyncView } from './ui/syncStatusView'
import VaultsView from './ui/VaultsView.vue'
import { WORDS as words } from './words'
import './styles.css'

export const SYNC_VIEW = 'sync'
export const VAULTS_VIEW = 'vaults'
export const SUBSCRIPTION_VIEW = 'subscription'
export const SHARES_VIEW = 'shares'
export const SHARE_COMMAND = 'share.open'
export const SYNC_COMMAND = 'vault.sync'
export const STOP_SYNC_COMMAND = 'vault.stopSync'
export const ACCEPT_COMMAND = 'share.accept'
export const CLOUD_COMMAND = 'vault.fromCloud'

const createSettingsOpener = (
  plugin: PluginContext,
  host: Pick<HostServices, 'ui'>,
): (() => void) | undefined => {
  const commands = host.ui.commands
  if (commands === undefined) return undefined
  return () => {
    commands
      .run(SETTINGS_OPEN_COMMAND, undefined, undefined, { plugin: plugin.identity.id })
      .catch((error: unknown) => host.ui.notices.show(String(error), 'error'))
  }
}

const getCurrentVault = (host: HostServices): string =>
  host.vault.vaults?.getState().current?.id ?? ''

/** The folder of the vault the command ran on: the vault root is the empty folder. */
const getFolder = (path: string): string => {
  const trimmed = path.replace(/^\/+|\/+$/g, '')
  return trimmed === '.' ? '' : trimmed
}

const registerShare = (plugin: PluginContext, host: HostServices): void => {
  plugin.register.command({
    id: SHARE_COMMAND,
    handler: (note) => {
      const dialogs = host.ui.dialogs
      if (dialogs === undefined) return Promise.resolve()
      const view = createInvitationView({
        invitations: shareInvitationClient,
        shares: folderSharesClient,
        target: { vaultId: getCurrentVault(host), folder: getFolder(note.path) },
        vaultName: host.vault.vaults?.getState().current?.name ?? '',
        close: () => {
          view.stop()
          dialog.close()
        },
      })
      const dialog = dialogs.open({ component: ShareInvitationDialog, props: { state: view } })
      view.start().catch(view.fail)
      return Promise.resolve()
    },
  })
}

const registerDialogs = (plugin: PluginContext, host: HostServices): void => {
  plugin.register.command({
    id: ACCEPT_COMMAND,
    handler: () => {
      const dialogs = host.ui.dialogs
      if (dialogs === undefined) return Promise.resolve()
      const view = createAcceptanceView({
        client: acceptInvitationClient,
        close: () => {
          view.stop()
          dialog.close()
        },
      })
      const dialog = dialogs.open({ component: AcceptInvitationDialog, props: { view } })
      return Promise.resolve()
    },
  })
  plugin.register.command({
    id: CLOUD_COMMAND,
    handler: () => {
      const dialog = host.ui.dialogs?.open({
        component: GetVaultDialog,
        props: { onClose: () => dialog?.close() },
      })
      return Promise.resolve()
    },
  })
}

const getOutcomeText = (kind: 'synced' | 'already'): string =>
  kind === 'synced' ? words.syncStarted : words.syncedAlready

const registerSync = (
  plugin: PluginContext,
  host: HostServices,
  refreshFlags: () => Promise<void>,
): void => {
  const show = (failure: unknown): void =>
    host.ui.notices.show(failure instanceof Error ? failure.message : String(failure), 'error')
  const choose = (vaultId: string, name: string, remoteId: string): void => {
    const dialogs = host.ui.dialogs
    if (dialogs === undefined) return
    const finish = (id: string): void => {
      dialog.close()
      finishSync(vaultsClient, vaultId, id)
        .then(() => {
          host.ui.notices.show(words.syncStarted)
          return refreshFlags()
        })
        .catch(show)
    }
    const dialog = dialogs.open({
      component: SyncChoiceDialog,
      props: {
        name,
        onUse: () => finish(remoteId),
        onSend: () => finish(''),
        onCancel: () => dialog.close(),
      },
    })
  }
  plugin.register.command({
    id: SYNC_COMMAND,
    handler: async () => {
      const vaultId = getCurrentVault(host)
      try {
        const outcome = await startSync(vaultsClient, vaultId)
        if (outcome.kind === 'choose') choose(vaultId, outcome.name, outcome.remoteId)
        else host.ui.notices.show(getOutcomeText(outcome.kind))
        await refreshFlags()
      } catch (failure) {
        show(failure)
      }
    },
  })
}

export const activate = (plugin: PluginContext, host: HostServices): void => {
  const state = createSyncState(vaultsClient)
  plugin.register.slotView({ id: SYNC_VIEW, component: createSyncView(state) })
  plugin.register.settingsView({ id: VAULTS_VIEW, component: VaultsView })
  plugin.register.settingsView({ id: SHARES_VIEW, component: createSharesPage() })
  plugin.register.settingsView({
    id: SUBSCRIPTION_VIEW,
    component: createSubscriptionView(plugin, host, createSettingsOpener(plugin, host)),
  })
  registerShare(plugin, host)
  const flags = host.ui.vaultFlags
  if (flags === undefined) throw new Error('the window gives no vault flags')
  const synced = createSyncedFlags({
    client: vaultsClient,
    flags,
    getVaultIds: () => host.vault.vaults?.getState().vaults.map((one) => one.id) ?? [],
  })
  registerSync(plugin, host, synced.refresh)
  registerUnsync(plugin, host, STOP_SYNC_COMMAND)
  registerDialogs(plugin, host)
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  synced.start()
  plugin.identity.subscriptions.push({ dispose: () => synced.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
