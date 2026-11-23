/** Which widget rows of a group draw an element that takes room or text, read from the document; a widget that renders nothing leaves a comment node. */
import { measureRect } from '@numen/ui'
import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'

const isDrawn = (node: Node): boolean =>
  node.nodeType === Node.ELEMENT_NODE ||
  (node.nodeType === Node.TEXT_NODE && (node.textContent ?? '').trim() !== '')

const hasSize = (row: Element): boolean =>
  row.getClientRects().length === 0 || measureRect(row).height > 0

const isFilled = (row: Element): boolean => hasSize(row) && [...row.childNodes].some(isDrawn)

const findFilledRows = (rows: Element): Set<string> =>
  new Set(
    [...rows.children]
      .filter(isFilled)
      .map((row, index) => (row as HTMLElement).dataset.plexWidget ?? String(index)),
  )

/**
 * `change` is given `rows` once a widget draws and `null` when none does or the rows are gone; `isDrawing` follows it.
 * `rows` is given after the document has drawn the group the widgets sit in, so what `change` measures includes it,
 * and `isMeasured` holds from then on. `refresh` reads the document at once, where the observer would after the change,
 * and gives the group to `change` again when it draws, so a size it has grown to is measured without waiting for the window.
 */
export function useDrawnWidgets(
  rows: Readonly<Ref<HTMLElement | null>>,
  change: (element: HTMLElement | null) => void,
): {
  isDrawing: Readonly<Ref<boolean>>
  isMeasured: Readonly<Ref<boolean>>
  filled: Readonly<Ref<ReadonlySet<string>>>
  refresh: () => void
} {
  const isDrawing = ref(false)
  const filled = shallowRef<ReadonlySet<string>>(new Set())
  let known: ReadonlySet<string> = filled.value
  const isMeasured = ref(true)
  const drawn = shallowRef<HTMLElement | null>(null)

  const update = () => {
    const element = rows.value
    const keys = element ? findFilledRows(element) : new Set<string>()
    if (keys.size !== known.size || [...keys].some((key) => !known.has(key))) {
      known = keys
      filled.value = keys
    }
    const next = element && keys.size > 0 ? element : null
    isDrawing.value = next !== null
    if (next === drawn.value) return
    drawn.value = next
    isMeasured.value = next === null
    if (next === null) change(null)
  }

  const refresh = () => {
    update()
    if (drawn.value && isMeasured.value) change(drawn.value)
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
  const resizer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(update)

  watch(
    rows,
    (element) => {
      observer.disconnect()
      resizer?.disconnect()
      if (element) {
        observer.observe(element, { childList: true, subtree: true, characterData: true })
        resizer?.observe(element)
      }
      update()
    },
    { flush: 'sync' },
  )

  onScopeDispose(() => {
    observer.disconnect()
    resizer?.disconnect()
    if (drawn.value) change(null)
  })

  return { isDrawing, isMeasured, filled, refresh }
}
