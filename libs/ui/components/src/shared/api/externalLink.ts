/**
 * A link that leads out of the application.
 *
 * The window has no address bar, so a page opened inside it is shown in the
 * application frame. The window stays on content it serves itself; an address a
 * browser can open is passed to the system browser.
 */

import { isExternalLink } from '../lib/text/externalLink'

/** The schemes passed to the browser. Any other scheme is not opened. */
const ALLOWED_SCHEMES = new Set(['http:', 'https:', 'mailto:', 'tel:'])

/** Passes an address to the system browser. */
export type LinkOpener = (href: string) => void

/**
 * Keeps the window on the pages it serves until the returned function is called.
 * A press is read before anything on the page sees it, so nothing on the page can navigate the window away.
 * Links that mean something inside the application are left to the application.
 */
export const interceptExternalLinks = (
  openLink: LinkOpener,
  root: Document = document,
): (() => void) => {
  const onPress = (press: MouseEvent) => {
    const pressed = press.target as Partial<Pick<Element, 'closest'>> | null
    const link = pressed?.closest?.('a[href]')
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
