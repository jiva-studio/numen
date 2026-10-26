/**
 * The width the pages of a document are asked for at.
 *
 * One width serves the whole document, so a page turned to is already drawn at
 * it, and it follows the row only once the row has stood still.
 */
import { computed, onBeforeUnmount, ref, watch, type ComputedRef, type Ref } from 'vue'
import { WIDTH_DEBOUNCE, WIDTH_STEP, type Row } from '../lib/strip'

/** How many device pixels a CSS pixel is. One, where there is no window to ask. */
const pixelRatio = (): number =>
  typeof window === 'undefined' || !Number.isFinite(window.devicePixelRatio)
    ? 1
    : window.devicePixelRatio

/** The width a page is asked for at, staged. */
const roundUpWidth = (pixels: number): number => Math.ceil(pixels / WIDTH_STEP) * WIDTH_STEP

export interface PageWidthState {
  /** What the row asks for its pages at, in device pixels. */
  readonly needed: ComputedRef<number>
  /** What each page is asked for at, which follows that once it has settled. */
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

  // The first width is asked for at once: a document opening has nothing drawn
  // and nothing to wait for.
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
