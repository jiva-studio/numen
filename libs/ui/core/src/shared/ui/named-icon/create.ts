import { defineComponent, h, type Component } from 'vue'
import NamedIcon from './NamedIcon.vue'

const created = new Map<string, Component>()

/**
 * The component that draws the icon of the name, for whoever takes a component and no name. The
 * same name gives the same component.
 */
export function createNamedIcon(name: string): Component {
  const cached = created.get(name)
  if (cached) return cached
  const made = defineComponent({
    name: 'NamedIconOf',
    inheritAttrs: false,
    setup:
      (_props, { attrs }) =>
      () =>
        h(NamedIcon, { ...attrs, name }),
  })
  created.set(name, made)
  return made
}
