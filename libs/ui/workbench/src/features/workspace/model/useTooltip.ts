import { computed, ref, useId } from 'vue'
import { measureRect, type Box } from '@numen/ui'

/**
 * The state of one tooltip beside the element that raises it: where it is drawn while it is open,
 * and the id the element points to it with.
 */
export function useTooltip() {
  const id = useId()
  const at = ref<Box | null>(null)
  const describedBy = computed(() => (at.value ? id : undefined))

  function open(event: Event): void {
    const { x, y, width, height } = measureRect(event.currentTarget as HTMLElement)
    at.value = { x, y, width, height }
  }

  function close(): void {
    at.value = null
  }

  return { id, at, describedBy, open, close }
}
