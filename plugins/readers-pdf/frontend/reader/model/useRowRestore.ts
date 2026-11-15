/** Keeps the row where the person left it through zooms, turns from outside and a viewport that is reported again. */
import { watch, type Ref } from 'vue'
import type { Paint, Size } from '@numen/ui'
import {
  getAcrossScroll,
  getAcrossShare,
  getAnchor,
  getAnchorScroll,
  getCurrentPage,
  getScrollTarget,
  type Anchor,
  type Row,
} from '../lib/strip'
import type { PanScroll } from './scroll'

export interface RowRestoreSources {
  readonly area: Readonly<Ref<HTMLElement | null>>
  readonly viewport: Ref<Size>
  readonly measureViewport: () => void
  readonly pageLayout: Readonly<Ref<Row>>
  readonly getAt: () => number
  readonly getPaint: () => Paint
  readonly scroll: PanScroll['moves']
}

export interface RowRestore {
  readonly measure: () => void
  readonly isPutting: () => boolean
}

export function useRowRestore(sources: RowRestoreSources): RowRestore {
  const { area, viewport, measureViewport, pageLayout, getAt, getPaint, scroll } = sources

  /**
   * How many puts of the row back wait for a frame. The browser queues a scroll event when the row
   * scrolls and delivers it with the next frame. An event queued before the row was hidden or moved
   * reaches the handler after the browser has scrolled the row to its start, and reports the start as
   * a place the person scrolled to.
   */
  let pendingPuts = 0

  /** The point kept through zooms whose redraw has not run yet, so a second zoom keeps the first one's. */
  let pendingZoom: { anchor: Anchor | undefined; across: number } | undefined

  function putRowBack(): void {
    if (!scroll.isMoved()) {
      getPaint()(scroll.restore)
      return
    }
    pendingPuts += 1
    getPaint()(() => {
      scroll.restore()
      pendingPuts -= 1
    })
  }

  function stand(page: number, how: Parameters<PanScroll['moves']['send']>[1]): void {
    const to = getScrollTarget(pageLayout.value, viewport.value, page)
    if (to !== undefined) scroll.send(to, how)
  }

  // A page requested from outside is scrolled to. A page reached by the user's
  // scrolling is already in view.
  watch(getAt, (page) => {
    if (page === getCurrentPage(pageLayout.value, viewport.value, scroll.getScrollOffset())) return
    // A redraw that has not run yet holds the place the row was at when the layout changed, and
    // would send the row back there once the turn has started. The turn is where the row goes.
    if (pendingZoom) pendingZoom.anchor = undefined
    stand(page, 'smooth')
  })

  // Zooming keeps the point under the center of the viewport where it is. Every
  // page changes width, so the scroll position shifts. A row with no pages before
  // it has no point to keep, and the current page is shown instead.
  watch(pageLayout, (now, was) => {
    if (now.height === was.height) return
    pendingZoom ??= {
      anchor: getAnchor(was, viewport.value, scroll.getScrollOffset()),
      across: getAcrossShare(was, viewport.value, area.value?.scrollTop ?? 0),
    }
    getPaint()(() => {
      const point = pendingZoom
      pendingZoom = undefined
      if (!point) return
      const to = point.anchor && getAnchorScroll(pageLayout.value, viewport.value, point.anchor)
      if (to === undefined) stand(getAt(), 'auto')
      else scroll.send(to, 'auto')
      if (area.value && point.anchor) {
        area.value.scrollTop = getAcrossScroll(pageLayout.value, viewport.value, point.across)
      }
    })
  })

  // A viewport that is reported again was hidden or moved, and the browser
  // scrolled its row to the start without a scroll event.
  watch(viewport, putRowBack)

  return {
    measure: () => {
      measureViewport()
      putRowBack()
    },
    isPutting: () => pendingPuts > 0,
  }
}
