/**
 * What the window takes for a web address.
 *
 * It is the shape alone. The vault reads the address again and is what refuses
 * one nothing can be fetched from, so this is only what decides whether the
 * person is offered the import at all.
 */

export const isWebUrl = (typed: string): boolean => {
  try {
    const address = new URL(typed.trim())
    return (address.protocol === 'http:' || address.protocol === 'https:') && address.hostname !== ''
  } catch {
    // A string a URL cannot be constructed from is not a web url.
    return false
  }
}

export const isWebAddress = isWebUrl

