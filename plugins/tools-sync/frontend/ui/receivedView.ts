import { defineComponent, h, onMounted } from 'vue'
import { createReceivedView, type ReceivedViewOptions } from '../model/shares'
import ReceivedView from './ReceivedView.vue'

export function createReceivedPage(options: ReceivedViewOptions = {}) {
  return defineComponent({
    name: 'ReceivedPage',
    inheritAttrs: false,
    setup: () => {
      const view = createReceivedView(options)
      onMounted(() => {
        view.load().catch(view.fail)
      })
      return () => h(ReceivedView, { view })
    },
  })
}
