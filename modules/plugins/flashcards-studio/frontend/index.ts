import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import DeckEditorTab from './DeckEditorTab.vue'
import StencilEditorTab from './StencilEditorTab.vue'
import PresetEditorTab from './PresetEditorTab.vue'
import { DEFAULTS } from './types'

export { DeckEditorTab, StencilEditorTab, PresetEditorTab, DEFAULTS }
export { DeckEditorTab as DeckTab, StencilEditorTab as StencilTab, PresetEditorTab as PresetTab }
export * from './types'

export function activate(context: PluginContext): void {
  context.registerTabKind({
    kind: 'deck',
    component: DeckEditorTab,
    title: 'Deck Editor',
  })
  context.registerTabKind({
    kind: 'stencil',
    component: StencilEditorTab,
    title: 'Card Stencil Editor',
  })
  context.registerTabKind({
    kind: 'preset',
    component: PresetEditorTab,
    title: 'Review Presets',
  })
}

export default definePlugin({
  activate,
})
