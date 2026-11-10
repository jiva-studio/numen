import type { ComponentPublicInstance } from 'vue'

/** What a function ref receives: the element or component instance, and null when it is removed. */
export type RefTarget = Element | ComponentPublicInstance | null

/**
 * Function refs for elements drawn in a loop, one per key.
 *
 * A key returns the same function on every request until its element is
 * removed; `onChange` is called with each element when it is set and with null
 * when it is removed.
 */
export interface RefRegistry<K> {
  readonly getRef: (key: K) => (element: RefTarget) => void
}

export const createRefRegistry = <K>(
  onChange: (key: K, element: RefTarget) => void,
): RefRegistry<K> => {
  const refs = new Map<K, (element: RefTarget) => void>()

  const getRef = (key: K): ((element: RefTarget) => void) => {
    let found = refs.get(key)
    if (!found) {
      found = (element: RefTarget) => {
        onChange(key, element)
        if (!element) refs.delete(key)
      }
      refs.set(key, found)
    }
    return found
  }

  return { getRef }
}
