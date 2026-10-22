/**
 * A link pressed in a book's text. Nothing a book contains navigates the
 * window: a link inside the book is a move within the book, and one leading out
 * of it is the window's own to hand on.
 *
 * A link to another document of the book names a place inside markup that has
 * not been drawn yet, so where it targetLink is held until that document is there.
 */
import { isExternalHref, parseBookLink, type BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/** What the reader says when a link is followed. */
export interface BookLinkSaid {
  /** The offset now in front, in bytes of the book's text. */
  readonly reportMove: (pos: number) => void
  /** A link targetLink to another document of the book, named as the archive names it. */
  readonly reportFollow: (path: string) => void
}

export function createBookLinks(props: SettledBookProps, listeners: BookLinkSaid) {
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
    listeners.reportMove(findFragmentOffset(place.fragment) ?? props.span.from)
  }

  return { takeLed, follow }
}
