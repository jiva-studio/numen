/**
 * How large a plex is drawn.
 *
 * A node's label is set in the window's type, and the box around it is a
 * number the arrangement is handed. Every length here is multiplied by what
 * the type was multiplied by, so a box goes on holding its label.
 */
import { doc, hasDocument } from '@numen/ui'
import { onScopeDispose, ref, type Ref } from 'vue'
import { DEFAULT_OPTIONS, type PlexOptions } from '../../lib/arrange'
import { DESIGNED_TYPE, scaleOptions } from './typeScale'

/** A plex drawn to hold a label set at this size. */
export const getOptionsForType = (
  type: number,
  options: PlexOptions = DEFAULT_OPTIONS,
): PlexOptions => scaleOptions(options, type > 0 ? type / DESIGNED_TYPE : 1)

/** A box one em on a side, set in the type a node's label is set in. */
const PROBE =
  'position:fixed;inset-block-start:0;inset-inline-start:0;inline-size:1em;' +
  'block-size:1em;font-size:var(--numen-font-size);visibility:hidden;pointer-events:none'

/**
 * The size a node's label is being set at, followed for as long as the caller lives.
 * It is measured off a box of the window's own type, so it follows a multiplier on the root and a theme naming a size of its own alike.
 * Where nothing is drawn (a test, a page rendered on a server) it is the size the plex was designed at.
 */
export function useTypeSize(): Ref<number> {
  const type = ref(DESIGNED_TYPE)
  if (!hasDocument() || typeof ResizeObserver === 'undefined') return type

  const probe = doc.createElement('div')
  probe.setAttribute('aria-hidden', 'true')
  probe.style.cssText = PROBE
  const observer = new ResizeObserver(([entry]) => {
    const box = entry?.contentRect
    if (box && box.width > 0) type.value = box.width
  })
  doc.body.appendChild(probe)
  observer.observe(probe)

  onScopeDispose(() => {
    observer.disconnect()
    probe.remove()
  }, true)

  return type
}
