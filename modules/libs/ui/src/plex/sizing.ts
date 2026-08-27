/**
 * How large a plex is drawn.
 *
 * A node's label is set in the window's type, and the box around it is a
 * number the arrangement is handed. Every length here is multiplied by what
 * the type was multiplied by, so a box goes on holding its label.
 */
import { onScopeDispose, ref, type Ref } from 'vue'
import { DEFAULT_OPTIONS, type PlexOptions, type Size } from './arrange'

/**
 * The size a node's label is set at with nothing multiplying it, which is the
 * chrome's type at a root of 16px. The boxes in `DEFAULT_OPTIONS` hold a label
 * of this size.
 */
export const DESIGNED_TYPE = 13

const larger = (size: Size, by: number): Size => ({
  width: size.width * by,
  height: size.height * by,
})

/**
 * The same plex, drawn larger or smaller.
 *
 * Every length is multiplied. The counts, the fractions and the directions
 * stand as they are, and so does the window, which is measured in the pixels
 * it really has.
 */
export const scaleOptions = (options: PlexOptions, by: number): PlexOptions => ({
  ...options,
  focusSize: larger(options.focusSize, by),
  nodeSize: larger(options.nodeSize, by),
  minWidth: options.minWidth * by,
  iconWidth: options.iconWidth * by,
  partHeight: options.partHeight * by,
  partIndent: options.partIndent * by,
  gap: options.gap * by,
  lineGap: options.lineGap * by,
  focusGap: options.focusGap * by,
  margin: options.margin * by,
  routing: {
    ...options.routing,
    minReach: options.routing.minReach * by,
    arrowRoom: options.routing.arrowRoom * by,
  },
})

/** A plex drawn to hold a label set at this size. */
export const optionsForType = (
  type: number,
  options: PlexOptions = DEFAULT_OPTIONS,
): PlexOptions => scaleOptions(options, type > 0 ? type / DESIGNED_TYPE : 1)

/** A box one em on a side, set in the type a node's label is set in. */
const PROBE =
  'position:fixed;inset-block-start:0;inset-inline-start:0;inline-size:1em;' +
  'block-size:1em;font-size:var(--numen-font-size);visibility:hidden;pointer-events:none'

/**
 * The size a node's label is being set at, followed for as long as the caller
 * lives. It is measured off a box of the window's own type, so it answers a
 * multiplier on the root and a theme naming a size of its own alike.
 *
 * Where nothing is drawn — a test, a page rendered on a server — it is the
 * size the plex was designed at.
 */
export function useTypeSize(): Ref<number> {
  const type = ref(DESIGNED_TYPE)
  if (typeof document === 'undefined' || typeof ResizeObserver === 'undefined') return type

  const probe = document.createElement('div')
  probe.setAttribute('aria-hidden', 'true')
  probe.style.cssText = PROBE
  const observer = new ResizeObserver(([entry]) => {
    const box = entry?.contentRect
    if (box && box.width > 0) type.value = box.width
  })
  document.body.appendChild(probe)
  observer.observe(probe)

  onScopeDispose(() => {
    observer.disconnect()
    probe.remove()
  }, true)

  return type
}
