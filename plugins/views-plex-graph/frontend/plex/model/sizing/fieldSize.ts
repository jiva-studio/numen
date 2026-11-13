/**
 * The size of the field the name of a new note is typed in: the width and height of the quick link
 * field, taken from the document so it follows the type of the window and no setting of the plex.
 */
import { doc, hasDocument } from '@numen/ui'
import { onScopeDispose, ref, type Ref } from 'vue'
import type { Size } from '../../lib/arrange'

/** The field at a root of 16px: 14rem wide and 2rem high. */
export const DESIGNED_FIELD: Size = { width: 224, height: 32 }

/** A box the size of the quick link field. */
const PROBE =
  'position:fixed;inset-block-start:0;inset-inline-start:0;inline-size:14rem;' +
  'block-size:var(--numen-action-size);visibility:hidden;pointer-events:none'

/** The size of the field, followed for as long as the caller lives. */
export function useFieldSize(): Ref<Size> {
  const size = ref<Size>(DESIGNED_FIELD)
  if (!hasDocument() || typeof ResizeObserver === 'undefined') return size

  const probe = doc.createElement('div')
  probe.setAttribute('aria-hidden', 'true')
  probe.style.cssText = PROBE
  const observer = new ResizeObserver(([entry]) => {
    const box = entry?.contentRect
    if (box && box.width > 0 && box.height > 0) size.value = { width: box.width, height: box.height }
  })
  doc.body.appendChild(probe)
  observer.observe(probe)

  onScopeDispose(() => {
    observer.disconnect()
    probe.remove()
  }, true)

  return size
}
