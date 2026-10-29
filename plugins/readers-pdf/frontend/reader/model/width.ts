/**
 * The width at which the pages of a document are requested.
 *
 * One width serves the whole document, so a page scrolled to is already
 * rendered at it, and the width follows the row only once the row has stopped
 * changing.
 */
import { hasWindow, win } from '../../dom'
import { computed, onBeforeUnmount, ref, watch, type ComputedRef, type Ref } from 'vue'
import { WIDTH_DEBOUNCE, WIDTH_STEP, type Row } from '../lib/strip'

/** How many device pixels a CSS pixel is. One, where there is no window to ask. */
const pixelRatio = (): number =>
  !hasWindow() || !Number.isFinite(win.devicePixelRatio) ? 1 : win.devicePixelRatio

/** Rounds a pixel width up to the next width step. */
const roundUpWidth = (pixels: number): number => Math.ceil(pixels / WIDTH_STEP) * WIDTH_STEP

export interface PageWidthState {
  /** The width the row requests for its pages, in device pixels. */
  readonly needed: ComputedRef<number>
  /** The width each page is requested at; follows `needed` once it has settled. */
  readonly drawnWidth: Ref<number>
}

export function usePageWidth(getRow: () => Row, atWidth: (pixels: number) => void): PageWidthState {
  /** The widest page there is, staged, in device pixels. */
  const needed = computed(() => {
    const widest = getRow().widths.reduce((most, each) => Math.max(most, each), 0)
    return roundUpWidth(widest * pixelRatio())
  })

  const drawnWidth = ref(0)
  /** Debounce before a changed width is drawn, once it stops changing. */
  let widthTimer: ReturnType<typeof setTimeout> | undefined

  // The first width applies immediately: a document that is opening has nothing
  // rendered and nothing to wait for.
  watch(needed, (pixels, before) => {
    if (!pixels) return
    if (!before) {
      drawnWidth.value = pixels
      atWidth(pixels)
      return
    }
    clearTimeout(widthTimer)
    widthTimer = setTimeout(() => {
      drawnWidth.value = pixels
      atWidth(pixels)
    }, WIDTH_DEBOUNCE)
  })

  onBeforeUnmount(() => {
    clearTimeout(widthTimer)
  })

  return { needed, drawnWidth }
}
