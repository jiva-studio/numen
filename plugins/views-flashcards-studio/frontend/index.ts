import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { cards, presets } from './cards'
import { useDeckTabs } from './deck/model/useDeckTabs'
import { useStencilTabs } from './stencil/kind'
import { usePresetTab } from './preset/kind'
import { createReviewKind, FLASHCARDS_TARGET } from './review'

export { DECK } from './deck/kind'
export { STENCIL } from './stencil/kind'
export { PRESET } from './preset/kind'
export { default as DeckTab } from './deck/ui/DeckTab.vue'
export { default as StencilTab } from './stencil/ui/StencilTab.vue'
export { default as PresetTab } from './preset/ui/PresetTab.vue'
export { useDeckTabs } from './deck/model/useDeckTabs'
export { useStencilTabs } from './stencil/kind'
export { usePresetTab } from './preset/kind'
export type { DeckTabState } from './deck/types'
export type { StencilTabState } from './stencil/types'
export type { PresetTabState } from './preset/types'

export { REVIEW } from './review'

export function activate(context: PluginContext): void {
  if (context.target === FLASHCARDS_TARGET) {
    context.registerTabKind(createReviewKind())
    return
  }
  const { workspace, buffers, files } = context
  if (!workspace || !buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => context.closeTab?.(id)
  const decks = useDeckTabs(cards, presets, { buffers, closeTab })
  const stencils = useStencilTabs(cards, {
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('stencil'),
  })
  const schedules = usePresetTab({
    core: presets,
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('preset'),
    today: () => workspace.getToday(),
  })
  context.registerTabKind(decks.kind)
  context.registerTabKind(stencils.kind)
  context.registerTabKind(schedules.kind)

  if (files) {
    const getFolder = (path: string, isFolder: boolean): string => {
      if (isFolder) return path
      const idx = path.lastIndexOf('/')
      return idx < 0 ? '' : path.slice(0, idx)
    }

    context.registerContextMenu({
      id: 'newDeck',
      text: 'New deck',
      parentId: 'new',
      target: 'all',
      handler: async (target) => {
        await cards.createDeck('New deck', getFolder(target.path, target.isFolder))
      },
    })
    context.registerContextMenu({
      id: 'newStencil',
      text: 'New stencil',
      parentId: 'new',
      target: 'all',
      handler: async (target) => {
        await cards.createStencil('New stencil', getFolder(target.path, target.isFolder), [])
      },
    })
    context.registerContextMenu({
      id: 'newPreset',
      text: 'New preset',
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
