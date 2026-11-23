import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { folderSharesClient, vaultsClient } from './api/wire'
import { createShareView, type ShareView } from './model/shares'
import { createSyncState } from './model/sync'
import { createSubscriptionView } from './ui/subscriptionView'
import { createReceivedPage } from './ui/receivedView'
import ShareDialog from './ui/ShareDialog.vue'
import { createSyncView } from './ui/syncStatusView'
import VaultsView from './ui/VaultsView.vue'
import { WORDS as words } from './words'
import './styles.css'

export const SYNC_VIEW = 'sync'
export const VAULTS_VIEW = 'vaults'
export const SUBSCRIPTION_VIEW = 'subscription'
export const SHARES_VIEW = 'shares'
export const SHARE_KIND = 'share'
export const SHARE_COMMAND = 'share.open'

const registerShare = (plugin: PluginContext, host: HostServices): void => {
  plugin.register.tabKind<ShareView>({
    kind: SHARE_KIND,
    title: words.shareTitle,
    component: ShareDialog,
    isTool: true,
    identity: (folder) => `${SHARE_KIND}:${folder}`,
    getTitle: (state) => words.shareFolderTitle(state.target.folder),
    open: (folder, id) => {
      const view = createShareView({
        client: folderSharesClient,
        target: { vaultId: host.vault.vaults?.getState().current?.id ?? '', folder },
        close: () => host.tabs.close(id),
      })
      view.load().catch(view.fail)
      return view
    },
    onKeyPress: (state, event) => {
      if (event.key !== 'Escape') return false
      state.close()
      return true
    },
  })
  plugin.register.command({
    id: SHARE_COMMAND,
    handler: async (note) => {
      await host.tabs.open?.(SHARE_KIND, note.path)
    },
  })
}

export const activate = (plugin: PluginContext, host: HostServices): void => {
  const state = createSyncState(vaultsClient)
  plugin.register.slotView({ id: SYNC_VIEW, component: createSyncView(state) })
  plugin.register.settingsView({ id: VAULTS_VIEW, component: VaultsView })
  plugin.register.settingsView({ id: SHARES_VIEW, component: createReceivedPage() })
  plugin.register.settingsView({
    id: SUBSCRIPTION_VIEW,
    component: createSubscriptionView(plugin.identity.openExternal),
  })
  registerShare(plugin, host)
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
