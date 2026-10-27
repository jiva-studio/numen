import { BookOpen } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import BookTab from './BookTab.vue'

export { BookTab }

export default definePlugin({
  activate(context: PluginContext) {
    context.registerReader({
      id: 'reader-epub',
      component: BookTab,
      title: 'EPUB Reader',
      icon: BookOpen,
    })
  },
  deactivate() {
    // Cleanup if needed
  },
})
