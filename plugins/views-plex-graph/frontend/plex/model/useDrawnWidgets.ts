/** Whether any widget in a group of widget rows draws an element or text, read from the document; a widget that renders nothing leaves a comment node. */
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'

const isDrawn = (node: Node): boolean =>
  node.nodeType === Node.ELEMENT_NODE ||
  (node.nodeType === Node.TEXT_NODE && (node.textContent ?? '').trim() !== '')

const hasDrawing = (rows: Element): boolean =>
  [...rows.children].some((row) => [...row.childNodes].some(isDrawn))

/**
 * `change` is given `rows` once a widget draws and `null` when none does or the rows are gone; `isDrawing` follows it.
 * `rows` is given after the document has drawn the group the widgets sit in, so what `change` measures includes it,
 * and `isMeasured` holds from then on. `refresh` reads the document at once, where the observer would after the change.
 */
export function useDrawnWidgets(
  rows: Readonly<Ref<HTMLElement | null>>,
  change: (element: HTMLElement | null) => void,
): { isDrawing: Readonly<Ref<boolean>>; isMeasured: Readonly<Ref<boolean>>; refresh: () => void } {
  const isDrawing = ref(false)
  const isMeasured = ref(true)
  const drawn = shallowRef<HTMLElement | null>(null)

  const update = () => {
    const element = rows.value
    const next = element && hasDrawing(element) ? element : null
    isDrawing.value = next !== null
    if (next === drawn.value) return
    drawn.value = next
    isMeasured.value = next === null
    if (next === null) change(null)
  }

  watch(
    drawn,
    (element) => {
      if (!element) return
      change(element)
      isMeasured.value = true
    },
    { flush: 'post' },
  )

  const observer = new MutationObserver(update)

  watch(
    rows,
    (element) => {
      observer.disconnect()
      if (element)
        observer.observe(element, { childList: true, subtree: true, characterData: true })
      update()
    },
    { flush: 'sync' },
  )

  onScopeDispose(() => observer.disconnect())

  return { isDrawing, isMeasured, refresh: update }
}
