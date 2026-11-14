/**
 * Whether the parts of a node are drawn in the layer over the others, which is while it is open,
 * and the Tab order through them there. The layer comes after every node in the document, so
 * the node moves Tab to its handle and back.
 */
import { computed, nextTick, watch } from 'vue'

const HANDLE = '.plex__handle'

export interface LiftSite {
  readonly layer: () => Element | null
  readonly open: () => number
  readonly group: () => SVGElement | null
  readonly lifted: () => Element | null
  readonly focus: {
    readonly onFocusIn: (event: FocusEvent) => void
    readonly onFocusOut: (event: FocusEvent) => void
  }
}

export function useLift(site: LiftSite) {
  const isLifted = computed(() => site.open() > 0 && site.layer() !== null)

  const getHandle = (): SVGElement | null =>
    (isLifted.value ? site.lifted() : site.group())?.querySelector<SVGElement>(HANDLE) ?? null

  const isLiftedHandle = (target: EventTarget | null): boolean =>
    target instanceof Element && target.matches(HANDLE) && !!site.lifted()?.contains(target)

  // The handle is made again where it moves to, and a focus on it moves with it. Chromium blurs
  // a focused element as it is taken out, which would close the node, so the node holds the
  // focus while the parts move.
  watch(isLifted, async () => {
    const group = site.group()
    const active = group?.ownerDocument.activeElement
    const hasFocus =
      !!active && active !== group && !!(group?.contains(active) || site.lifted()?.contains(active))
    if (!hasFocus) return
    group?.focus()
    await nextTick()
    getHandle()?.focus()
  })

  // Chromium makes an SVG element with a focus listener a Tab stop, so focus in the lifted parts
  // is received from the document.
  const onDocumentFocus = (event: FocusEvent) => {
    const lifted = site.lifted()
    if (!lifted || !(event.target instanceof Node) || !lifted.contains(event.target)) return
    if (event.type === 'focusin') site.focus.onFocusIn(event)
    else site.focus.onFocusOut(event)
  }
  watch(
    () => site.lifted(),
    (lifted, _, onCleanup) => {
      const document = lifted?.ownerDocument
      if (!document) return
      document.addEventListener('focusin', onDocumentFocus, true)
      document.addEventListener('focusout', onDocumentFocus, true)
      onCleanup(() => {
        document.removeEventListener('focusin', onDocumentFocus, true)
        document.removeEventListener('focusout', onDocumentFocus, true)
      })
    },
    { flush: 'sync' },
  )

  /**
   * Tab on the node goes to its lifted handle. Tab on that handle goes on from the node in the
   * browser's own order, which the lifted handle is not in, and Shift+Tab goes back to the node.
   */
  const onLiftKeyDown = (event: KeyboardEvent) => {
    if (event.key !== 'Tab') return
    const group = site.group()
    if (isLiftedHandle(event.target)) {
      group?.focus()
      if (event.shiftKey) event.preventDefault()
      return
    }
    if (event.shiftKey || event.target !== group || !isLifted.value) return
    const handle = getHandle()
    if (!handle) return
    event.preventDefault()
    handle.focus()
  }

  const liftTarget = computed(() => site.layer())

  return { isLifted, liftTarget, onLiftKeyDown }
}
