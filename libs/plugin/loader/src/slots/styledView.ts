/**
 * The view of a slot drawn with the stylesheet of the plugin that registered it. The owner of a slot
 * draws the views of other plugins inside its own bar or menu, where nothing else adds their styles.
 */
import { defineComponent, h, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import { adoptPluginStyle, getPluginStyle, type StyleRoot } from '../styles'

const styled = new WeakMap<object, Component>()

function addStyle(pluginId: string, element: HTMLElement): () => void {
  const release = adoptPluginStyle(pluginId, element.getRootNode() as StyleRoot)
  if (release) return release
  const href = getPluginStyle(pluginId)
  if (!href) return () => undefined
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = href
  link.dataset['pluginId'] = pluginId
  element.prepend(link)
  return () => link.remove()
}

/**
 * The component of a slot view behind the stylesheet of its plugin, the same component for the same
 * view on every read.
 */
export function createStyledView(pluginId: string, component: Component): Component {
  const existing = styled.get(component)
  if (existing) return existing
  const view = defineComponent({
    name: 'StyledSlotView',
    inheritAttrs: false,
    setup(_, { attrs, slots }) {
      const element = ref<HTMLElement>()
      let release: (() => void) | undefined
      onMounted(() => {
        if (element.value) release = addStyle(pluginId, element.value)
      })
      onBeforeUnmount(() => release?.())
      return () =>
        h('span', { ref: element, style: { display: 'contents' }, 'data-slot-view': pluginId }, [
          h(component, attrs, slots),
        ])
    },
  })
  styled.set(component, view)
  return view
}
