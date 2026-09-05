/**
 * The width the pages of a document are asked for at.
 *
 * One width serves the whole document, so a page turned to is already drawn at
 * it, and it follows the row only once the row has stood still.
 */
import { computed, onBeforeUnmount, ref, watch, type ComputedRef, type Ref } from 'vue'
import { SETTLED, STAGE, type Row } from './strip'

/** How many device pixels a CSS pixel is. One, where there is no window to ask. */
const pixelRatio = (): number =>
  typeof window === 'undefined' || !Number.isFinite(window.devicePixelRatio)
    ? 1
    : window.devicePixelRatio

/** The width a page is asked for at, staged. */
const staged = (pixels: number): number => Math.ceil(pixels / STAGE) * STAGE

export interface PageWidthState {
  /** What the row asks for its pages at, in device pixels. */
  readonly asking: ComputedRef<number>
  /** What each page is asked for at, which follows that once it has settled. */
  readonly drawnAt: Ref<number>
}

export function useAsking(laid: () => Row, wide: (pixels: number) => void): PageWidthState {
  /** The widest page there is, staged, in device pixels. */
  const asking = computed(() => {
    const widest = laid().widths.reduce((most, each) => Math.max(most, each), 0)
    return widest > 0 ? staged(widest * pixelRatio()) : 0
  })

  const drawnAt = ref(0)
  let settling: ReturnType<typeof setTimeout> | undefined

  // The first width is asked for at once: a document opening has nothing drawn
  // and nothing to wait for.
  watch(asking, (pixels, before) => {
    if (!pixels) return
    if (!before) {
      drawnAt.value = pixels
      wide(pixels)
      return
    }
    clearTimeout(settling)
    settling = setTimeout(() => {
      drawnAt.value = pixels
      wide(pixels)
    }, SETTLED)
  })

  onBeforeUnmount(() => {
    clearTimeout(settling)
  })

  return { asking, drawnAt }
}
