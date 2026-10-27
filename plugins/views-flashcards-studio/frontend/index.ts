import { Gauge, Layers, LayoutTemplate } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { cards, presets } from './cards'
import { useDeckTabs } from './deck/model/useDeckTabs'
import { useCardModelTabs } from './card-model/kind'
import { usePresetTab } from './preset/kind'
import { WORDS as presetWords } from './preset/words'
import {
  createNotices,
  createReviewKind,
  createVaultCounts,
  createVaultSummaryView,
  FLASHCARDS_TARGET,
} from './review'

export { DECK } from './deck/kind'
export { CARD_MODEL } from './card-model/kind'
export { PRESET } from './preset/kind'
export { default as DeckTab } from './deck/ui/DeckTab.vue'
export { default as CardModelTab } from './card-model/ui/CardModelTab.vue'
export { default as PresetTab } from './preset/ui/PresetTab.vue'
export { useDeckTabs } from './deck/model/useDeckTabs'
export { useCardModelTabs } from './card-model/kind'
export { usePresetTab } from './preset/kind'
export type { DeckTabState } from './deck/types'
export type { CardModelTabState } from './card-model/types'
export type { PresetTabState } from './preset/types'

export { REVIEW } from './review'

export function activate(context: PluginContext): void {
  if (context.target === FLASHCARDS_TARGET) {
    const { notices, closeTab, agent } = context
    if (!notices || !closeTab || !agent) {
      throw new Error(
        'the review needs the notices, the tabs and the agent of the window it is drawn in',
      )
    }
    const counts = createVaultCounts({ reportError: createNotices(notices).reportError })
    context.subscriptions.push({ dispose: counts.dispose })
    context.registerView(createVaultSummaryView(counts))
    context.registerTabKind(
      createReviewKind({
        counts,
        notices,
        agent,
        mountView: (capability, element, view) => context.mountView(capability, element, view),
        closeTab: (id) => closeTab(id),
      }),
    )
    return
  }
  const { workspace, buffers, files } = context
  if (!workspace || !buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => context.closeTab?.(id)
  const decks = useDeckTabs(cards, presets, { buffers, closeTab })
  const cardModels = useCardModelTabs(cards, {
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('card-model'),
  })
  const schedules = usePresetTab({
    core: presets,
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('preset'),
    today: () => workspace.getToday(),
  })
  context.registerTabKind(decks.kind)
  context.registerTabKind(cardModels.kind)
  context.registerTabKind(schedules.kind)
  context.registerCommand({
    id: 'preset',
    title: presetWords.openPreset,
    scope: 'note',
    icon: Gauge,
    handler: async (note) => {
      await schedules.openPresetOf(note, (kind, path) => context.openTab?.(kind, path))
    },
  })

  if (files) {
    const getFolder = (path: string, isFolder: boolean): string => {
      if (isFolder) return path
      const idx = path.lastIndexOf('/')
      return idx < 0 ? '' : path.slice(0, idx)
    }

    context.registerContextMenu({
      id: 'newDeck',
      text: 'New deck',
      icon: Layers,
      parentId: 'new',
      target: 'all',
      handler: async (target) => {
        await cards.createDeck('New deck', getFolder(target.path, target.isFolder))
      },
    })
    context.registerContextMenu({
      id: 'newCardModel',
      text: 'New card model',
      icon: LayoutTemplate,
      parentId: 'new',
      target: 'all',
      handler: async (target) => {
        await cards.createCardModel('New card model', getFolder(target.path, target.isFolder), [])
      },
    })
    context.registerContextMenu({
      id: 'newPreset',
      text: 'New preset',
      icon: Gauge,
      parentId: 'new',
      target: 'all',
      handler: async (target) => {
        await presets.createPreset('New preset', getFolder(target.path, target.isFolder))
      },
    })
  }
}

export default definePlugin({
  activate,
})
