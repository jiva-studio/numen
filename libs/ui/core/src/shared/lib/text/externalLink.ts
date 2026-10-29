/**
 * A link that leads out of the application.
 *
 * The window has no address bar, so a page opened inside it is shown in the
 * application frame. The window stays on content it serves itself; an address a
 * browser can open is passed to the system browser.
 */

/** The schemes passed to the browser. Any other scheme is not opened. */
const ALLOWED_SCHEMES = new Set(['http:', 'https:', 'mailto:', 'tel:'])

/** Passes an address to the system browser. */
export type LinkOpener = (href: string) => void

/** Whether an address names somewhere this page is not served from. */
export const isExternalLink = (target: URL, here: URL): boolean =>
  target.protocol !== here.protocol || target.host !== here.host

/**
 * Holds the window on the pages it serves, for as long as the returned way of
 * stopping is not called.
 *
 * A press is read before anything on the page sees it, so nothing on the page
 * can carry the window off. What a link inside the application means is still
 * the application's, and is left alone.
 */
export const interceptExternalLinks = (
  openLink: LinkOpener,
  root: Document = document,
): (() => void) => {
  const onPress = (press: MouseEvent) => {
    const link = (press.target as Element | null)?.closest?.('a[href]')
    const href = link?.getAttribute('href')
    if (!href) return

    const here = new URL(root.location.href)
    let target: URL
    try {
      target = new URL(href, here)
    } catch {
      // An href that is no address points nowhere outward, so the press is the
      // page's own and is left to it.
      return
    }
    if (!isExternalLink(target, here)) return

    press.preventDefault()
    if (ALLOWED_SCHEMES.has(target.protocol)) openLink(target.href)
  }

  root.addEventListener('click', onPress, true)
  root.addEventListener('auxclick', onPress, true)
  return () => {
    root.removeEventListener('click', onPress, true)
    root.removeEventListener('auxclick', onPress, true)
  }
}
