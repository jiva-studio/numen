import { BookOpen } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import BookTab from './ui/BookTab.vue'

export { BookTab }

export default definePlugin({
  activate(context: PluginContext) {
    context.register.reader({
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
