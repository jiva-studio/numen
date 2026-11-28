import { defineComponent, h, onMounted } from 'vue'
import { createSharesView, type SharesViewOptions } from '../model/shares'
import SharesView from './SharesView.vue'

export function createSharesPage(options: SharesViewOptions = {}) {
  return defineComponent({
    name: 'SharesPage',
    inheritAttrs: false,
    setup: () => {
      const view = createSharesView(options)
      onMounted(() => {
        view.load().catch(view.fail)
      })
      return () => h(SharesView, { view })
    },
  })
}
