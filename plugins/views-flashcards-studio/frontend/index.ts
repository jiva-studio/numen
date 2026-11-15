import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { cards } from './api/cards'
import { presets } from './api/presets.wire'
import { useDeckTabs, createCardModelTabs } from './editor'
import { usePresetTab } from './preset'
import {
  createNotices,
  createReviewKind,
  createVaultCounts,
  createVaultSummaryView,
} from './review'

export {
  DECK,
  CARD_MODEL,
  DeckTab,
  CardModelTab,
  useDeckTabs,
  createCardModelTabs,
  type DeckTabState,
  type CardModelTabState,
} from './editor'
export { PRESET, PresetTab, usePresetTab, type PresetTabState } from './preset'
export { REVIEW } from './review'

export function activate(plugin: PluginContext, host: HostServices): void {
  const { workspace } = host.ui
  const { buffers } = host.vault
  if (!workspace) {
    const { notices } = host.ui
    const { close: closeTab } = host.tabs
    const { connections } = host.work
    const { relations } = host.vault
    if (!connections || !relations) {
      throw new Error('the review needs the connections and the links of the window it is drawn in')
    }
    const counts = createVaultCounts({ reportError: createNotices(notices).reportError })
    plugin.identity.subscriptions.push({ dispose: counts.dispose })
    plugin.register.view(createVaultSummaryView(counts))
    plugin.register.tabKind(
      createReviewKind({
        counts,
        notices,
        connections,
        relations,
        mountView: (capability, element, view) => host.ui.mountView(capability, element, view),
        closeTab: (id) => closeTab(id),
      }),
    )
    return
  }
  if (!buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => host.tabs.close(id)
  const decks = useDeckTabs(cards, presets, {
    buffers,
    closeTab,
    writeMessage: workspace.messages.getWriter('deck'),
  })
  const cardModels = createCardModelTabs(cards, {
    buffers,
    closeTab,
    writeMessage: workspace.messages.getWriter('card-model'),
  })
  const schedules = usePresetTab({
    core: presets,
    buffers,
    closeTab,
    writeMessage: workspace.messages.getWriter('preset'),
    today: () => workspace.window.getToday(),
  })
  plugin.register.tabKind(decks.kind)
  plugin.register.tabKind(cardModels.kind)
  plugin.register.tabKind(schedules.kind)
  plugin.register.command({
    id: 'preset.open',
    handler: async (note) => {
      await schedules.openPresetOf(note, (kind, path) => host.tabs.open?.(kind, path))
    },
  })
}

export default definePlugin({
  activate,
})
