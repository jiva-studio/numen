/**
 * A link activated in a book's text. Nothing a book contains navigates the
 * window: a link inside the book moves within the book, and a link leading out
 * of it is passed to the window.
 *
 * A link to another document of the book names a location inside markup that has
 * not been rendered yet, so the link is held until that document is available.
 */
import { isExternalHref, parseBookLink, type BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/** What the reader reports when a link is followed. */
export interface BookLinkReporter {
  /** A link to another document of the book, named as the archive names it. */
  readonly reportMove: (pos: number) => void
  /** A link targetLink to another document of the book, named as the archive names it. */
  readonly reportFollow: (path: string) => void
}

export function createBookLinks(props: SettledBookProps, listeners: BookLinkReporter) {
  /** Where a link targetLink, held until the document holding that place is drawn. */
  let targetLink: BookLink | undefined

  const takeLed = (): BookLink | undefined => {
    const place = targetLink
    targetLink = undefined
    return place
  }

  const follow = (
    press: MouseEvent,
    findFragmentOffset: (fragment: string) => number | undefined,
  ) => {
    const link = (press.target as Element | null)?.closest?.('a[href]')
    if (!link) return
    // `a[href]` matches only anchors that carry the attribute.
    const href = link.getAttribute('href') as string

    press.preventDefault()
    if (isExternalHref(href)) return

    const place = parseBookLink(href)
    if (place.path !== '' && place.path !== props.path) {
      targetLink = place
      listeners.reportFollow(place.path)
      return
    }
    listeners.reportMove(findFragmentOffset(place.fragment) ?? props.span.start)
  }

  return { takeLed, follow }
}
