import { computed, ref, useId } from 'vue'
import { measureRect } from '@/shared/api/dom'
import type { Box } from '@/shared/lib/dom/place'

/**
 * The state of one tooltip beside the element that raises it: where it is drawn while it is open,
 * the text it shows when several elements share it, and the id the element points to it with.
 */
export function useTooltip() {
  const id = useId()
  const at = ref<Box | null>(null)
  const source = ref<string | (() => string)>('')
  const text = computed(() => (typeof source.value === 'function' ? source.value() : source.value))
  const describedBy = computed(() => (at.value ? id : undefined))

  function open(event: Event, label: string | (() => string) = ''): void {
    const resolved = typeof label === 'function' ? label() : label
    if (arguments.length >= 2 && resolved === '') {
      close()
      return
    }
    const element = (event.currentTarget ?? event.target) as HTMLElement | null
    const rect = measureRect(element)
    at.value = rect ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null
    source.value = label
  }

  function close(): void {
    at.value = null
    source.value = ''
  }

  return { id, at, text, describedBy, open, close }
}
