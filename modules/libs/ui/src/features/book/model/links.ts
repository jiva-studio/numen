/**
 * A link pressed in a book's text. Nothing a book contains navigates the
 * window: a link inside the book is a move within the book, and one leading out
 * of it is the window's own to hand on.
 *
 * A link to another document of the book names a place inside markup that has
 * not been drawn yet, so where it led is held until that document is there.
 */
import { isOutwardHref, placeIn, type BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/** What the reader says when a link is followed. */
export interface BookLinkSaid {
  /** The offset now in front, in bytes of the book's text. */
  readonly moved: (at: number) => void
  /** A link led to another document of the book, named as the archive names it. */
  readonly followed: (path: string) => void
}

export function createBookLinks(props: SettledBookProps, listeners: BookLinkSaid) {
  /** Where a link led, held until the document holding that place is drawn. */
  let led: BookLink | undefined

  const takeLed = (): BookLink | undefined => {
    const place = led
    led = undefined
    return place
  }

  /** `placeAt` is the offset a place named inside the drawn document stands at. */
  const follow = (press: MouseEvent, placeAt: (fragment: string) => number | undefined) => {
    const link = (press.target as Element | null)?.closest?.('a[href]')
    const href = link?.getAttribute('href')
    if (href === null || href === undefined) return

    press.preventDefault()
    if (isOutwardHref(href)) return

    const place = placeIn(href)
    if (place.path !== '' && place.path !== props.path) {
      led = place
      listeners.followed(place.path)
      return
    }
    listeners.moved(placeAt(place.fragment) ?? props.span.begins)
  }

  return { takeLed, follow }
}
