/**
 * Function refs for elements drawn in a loop, one per key.
 *
 * A key returns the same function on every request until its element is
 * removed; `onChange` is called with each element when it is set and with null
 * when it is removed.
 */
export interface RefRegistry<K> {
  /** The function ref for the key. */
  readonly getRef: (key: K) => (element: unknown) => void
}

export const createRefRegistry = <K>(
  onChange: (key: K, element: unknown) => void,
): RefRegistry<K> => {
  const refs = new Map<K, (element: unknown) => void>()

  const getRef = (key: K): ((element: unknown) => void) => {
    let found = refs.get(key)
    if (!found) {
      found = (element: unknown) => {
        onChange(key, element)
        if (!element) refs.delete(key)
      }
      refs.set(key, found)
    }
    return found
  }

  return { getRef }
}
