import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
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

export function activate(context: PluginContext): void {
  const { workspace } = context.ui
  const { buffers } = context.vault
  if (!workspace) {
    const { notices } = context.ui
    const { close: closeTab } = context.tabs
    const { connections } = context.work
    const { relations } = context.vault
    if (!connections || !relations) {
      throw new Error('the review needs the connections and the links of the window it is drawn in')
    }
    const counts = createVaultCounts({ reportError: createNotices(notices).reportError })
    context.plugin.subscriptions.push({ dispose: counts.dispose })
    context.register.view(createVaultSummaryView(counts))
    context.register.tabKind(
      createReviewKind({
        counts,
        notices,
        connections,
        relations,
        mountView: (capability, element, view) => context.ui.mountView(capability, element, view),
        closeTab: (id) => closeTab(id),
      }),
    )
    return
  }
  if (!buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => context.tabs.close(id)
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
  context.register.tabKind(decks.kind)
  context.register.tabKind(cardModels.kind)
  context.register.tabKind(schedules.kind)
  context.register.command({
    id: 'preset.open',
    handler: async (note) => {
      await schedules.openPresetOf(note, (kind, path) => context.tabs.open?.(kind, path))
    },
  })
}

export default definePlugin({
  activate,
})
