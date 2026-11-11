/**
 * A link that leads out of the application.
 *
 * The window has no address bar, so a page opened inside it is shown in the
 * application frame. The window stays on content it serves itself; an address a
 * browser can open is passed to the system browser.
 */

/** Whether an address names somewhere this page is not served from. */
export const isExternalLink = (target: URL, here: URL): boolean =>
  target.protocol !== here.protocol || target.host !== here.host
