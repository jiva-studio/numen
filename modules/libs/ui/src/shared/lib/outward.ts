/**
 * A link that leads out of the application.
 *
 * A window has no address bar, so a page opened inside it wears the
 * application's own frame. The window is held on what it serves itself, and an
 * address a browser can be handed is handed to the person's browser instead.
 */

/** The schemes a browser is handed. Anything else is nobody's to open. */
const HANDED = new Set(['http:', 'https:', 'mailto:', 'tel:'])

/** Hands an address to wherever the person's own browser opens it. */
export type LinkOpener = (href: string) => void

/** Whether an address names somewhere this page is not served from. */
export const isOutward = (at: URL, here: URL): boolean =>
  at.protocol !== here.protocol || at.host !== here.host

/**
 * Holds the window on the pages it serves, for as long as the returned way of
 * stopping is not called.
 *
 * A press is read before anything on the page sees it, so nothing on the page
 * can carry the window off. What a link inside the application means is still
 * the application's, and is left alone.
 */
export const holdWindow = (openLink: LinkOpener, root: Document = document): (() => void) => {
  const onPress = (press: MouseEvent) => {
    const link = (press.target as Element | null)?.closest?.('a[href]')
    const href = link?.getAttribute('href')
    if (href === null || href === undefined) return

    const here = new URL(root.location.href)
    let at: URL
    try {
      at = new URL(href, here)
    } catch {
      // An href that is no address points nowhere outward, so the press is the
      // page's own and is left to it.
      return
    }
    if (!isOutward(at, here)) return

    press.preventDefault()
    if (HANDED.has(at.protocol)) openLink(at.href)
  }

  root.addEventListener('click', onPress, true)
  root.addEventListener('auxclick', onPress, true)
  return () => {
    root.removeEventListener('click', onPress, true)
    root.removeEventListener('auxclick', onPress, true)
  }
}
