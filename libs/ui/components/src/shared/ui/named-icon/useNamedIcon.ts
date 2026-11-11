import { shallowRef, watch, type Component, type ShallowRef } from 'vue'
import { findNamedIcon } from './icons'

/** The icon drawn for a name the set does not have. */
const PLACEHOLDER_ICON = 'circle-question-mark'

/** The icon of a name, null until it is found and for an empty name. A name changed meanwhile drops the older answer. */
export function useNamedIcon(getName: () => string): ShallowRef<Component | null> {
  const icon = shallowRef<Component | null>(null)

  watch(
    getName,
    async (name) => {
      icon.value = null
      const found =
        (await findNamedIcon(name)) ?? (name === '' ? null : await findNamedIcon(PLACEHOLDER_ICON))
      if (getName() === name) icon.value = found
    },
    { immediate: true },
  )

  return icon
}
