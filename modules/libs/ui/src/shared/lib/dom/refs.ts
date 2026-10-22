/**
 * Function refs for elements drawn in a loop, one per key.
 *
 * A key is handed the same function every time it is asked for, until its
 * element goes; `onChange` hears of each element as it is set and with null as
 * it goes.
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
