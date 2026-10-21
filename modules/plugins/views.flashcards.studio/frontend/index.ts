import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { cards, presets } from './cards'
import { useDeckTabs } from './deck/model/useDeckTabs'
import { useStencilTabs } from './stencil/kind'
import { usePresetTab } from './preset/kind'

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

export function activate(context: PluginContext): void {
  const { workspace, buffers } = context
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
}

export default definePlugin({
  activate,
})
