import BookTab from './BookTab.vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { setBookPluginActive } from '@/pages/book-reader/model/pluginState'

export { BookTab }

export function activate(_context?: PluginContext): void {
  setBookPluginActive(true)
}

export function deactivate(): void {
  setBookPluginActive(false)
}

export default definePlugin({
  activate,
  deactivate,
})
