/**
 * What the window takes for a web address.
 *
 * It is the shape alone. The vault reads the address again and is what refuses
 * one nothing can be fetched from, so this is only what decides whether the
 * person is offered the import at all.
 */

/** Whether these words are somewhere a browser would go. */
export const isWebAddress = (typed: string): boolean => {
  try {
    const address = new URL(typed.trim())
    return (address.protocol === 'http:' || address.protocol === 'https:') && address.hostname !== ''
  } catch {
    return false
  }
}
