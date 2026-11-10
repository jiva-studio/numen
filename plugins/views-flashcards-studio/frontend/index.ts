import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { cards } from './cards/api/cards'
import { presets } from './cards/api/presets.wire'
import { useDeckTabs } from './deck/model/useDeckTabs'
import { createCardModelTabs } from './card-model/kind'
import { usePresetTab } from './preset/kind'
import {
  createNotices,
  createReviewKind,
  createVaultCounts,
  createVaultSummaryView,
} from './review'

export { DECK } from './deck/kind'
export { CARD_MODEL } from './card-model/kind'
export { PRESET } from './preset/kind'
export { default as DeckTab } from './deck/ui/DeckTab.vue'
export { default as CardModelTab } from './card-model/ui/CardModelTab.vue'
export { default as PresetTab } from './preset/ui/PresetTab.vue'
export { useDeckTabs } from './deck/model/useDeckTabs'
export { createCardModelTabs } from './card-model/kind'
export { usePresetTab } from './preset/kind'
export type { DeckTabState } from './deck/types'
export type { CardModelTabState } from './card-model/types'
export type { PresetTabState } from './preset/types'

export { REVIEW } from './review'

export function activate(context: PluginContext): void {
  const { workspace } = context.ui
  const { buffers } = context.vault
  if (!workspace) {
    const { notices } = context.ui
    const { close: closeTab } = context.tabs
    const { agent } = context.work
    if (!notices || !closeTab || !agent) {
      throw new Error(
        'the review needs the notices, the tabs and the agent of the window it is drawn in',
      )
    }
    const counts = createVaultCounts({ reportError: createNotices(notices).reportError })
    context.plugin.subscriptions.push({ dispose: counts.dispose })
    context.register.view(createVaultSummaryView(counts))
    context.register.tabKind(
      createReviewKind({
        counts,
        notices,
        agent,
        mountView: (capability, element, view) => context.ui.mountView(capability, element, view),
        closeTab: (id) => closeTab(id),
      }),
    )
    return
  }
  if (!buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => context.tabs.close?.(id)
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
