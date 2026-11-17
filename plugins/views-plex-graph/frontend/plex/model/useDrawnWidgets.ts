/** Whether any widget in a group of widget rows draws an element or text, read from the document; a widget that renders nothing leaves a comment node. */
import { onScopeDispose, ref, watch, type Ref } from 'vue'

const isDrawn = (node: Node): boolean =>
  node.nodeType === Node.ELEMENT_NODE ||
  (node.nodeType === Node.TEXT_NODE && (node.textContent ?? '').trim() !== '')

const hasDrawing = (rows: Element): boolean =>
  [...rows.children].some((row) => [...row.childNodes].some(isDrawn))

/**
 * `change` is given `rows` once a widget draws and `null` when none does or the rows are gone; `isDrawing` follows it.
 */
export function useDrawnWidgets(
  rows: Readonly<Ref<HTMLElement | null>>,
  change: (element: HTMLElement | null) => void,
): { isDrawing: Readonly<Ref<boolean>> } {
  const isDrawing = ref(false)
  let given: HTMLElement | null = null

  const update = () => {
    const element = rows.value
    const next = element && hasDrawing(element) ? element : null
    isDrawing.value = next !== null
    if (next === given) return
    given = next
    change(next)
  }

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

  return { isDrawing }
}
