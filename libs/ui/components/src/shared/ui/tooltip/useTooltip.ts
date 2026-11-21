import { computed, ref, useId, watch } from 'vue'
import { measureRect } from '@/shared/api/dom'
import type { Box } from '@/shared/lib/dom/place'

const INPUT_EVENTS = ['pointerdown', 'keydown', 'wheel'] as const

/**
 * The state of one tooltip beside the element that raises it: where it is drawn while it is open,
 * the text it shows when several elements share it, and the id the element points to it with.
 *
 * It closes when the element leaves the document, on a key press, a click or a wheel turn, when the
 * window loses focus, and when the document or an ancestor of the element scrolls. The blur or the
 * scroll of an unrelated element leaves it open.
 */
export function useTooltip() {
  const id = useId()
  const at = ref<Box | null>(null)
  const source = ref<string | (() => string)>('')
  const text = computed(() => (typeof source.value === 'function' ? source.value() : source.value))
  const describedBy = computed(() => (at.value ? id : undefined))
  let anchor: HTMLElement | null = null

  function open(event: Event, label: string | (() => string) = ''): void {
    const resolved = typeof label === 'function' ? label() : label
    if (arguments.length >= 2 && resolved === '') {
      close()
      return
    }
    const element = (event.currentTarget ?? event.target) as HTMLElement | null
    anchor = element
    const rect = measureRect(element)
    at.value = rect ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null
    source.value = label
  }

  function close(): void {
    at.value = null
    source.value = ''
    anchor = null
  }

  watch(at, (value, _, onCleanup) => {
    if (!value) return
    const attached = anchor?.isConnected === true
    const observer = new MutationObserver(() => {
      if (attached && !anchor?.isConnected) close()
    })
    observer.observe(document.body, { childList: true, subtree: true })
    const onScroll = (event: Event) => {
      const target = event.target
      if (target === document || (target instanceof Node && target.contains(anchor))) close()
    }
    for (const name of INPUT_EVENTS) window.addEventListener(name, close, true)
    window.addEventListener('scroll', onScroll, true)
    window.addEventListener('blur', close)
    onCleanup(() => {
      observer.disconnect()
      for (const name of INPUT_EVENTS) window.removeEventListener(name, close, true)
      window.removeEventListener('scroll', onScroll, true)
      window.removeEventListener('blur', close)
    })
  })

  return { id, at, text, describedBy, open, close }
}
