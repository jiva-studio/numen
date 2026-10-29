/**
 * Viewport positioning and resize tracking for link inspector popover.
 */
import { measureRect, measureWindow, win } from '../dom'
import { computed, nextTick, onMounted, onScopeDispose, ref, watch } from 'vue'
import type { Ref } from 'vue'
import { resolvePopoverPosition } from './linkInspector'

export interface PopoverPositionOptions {
  readonly target: Ref<HTMLElement | null>
  readonly at: () => { readonly x: number; readonly y: number }
  readonly watchTriggers?: readonly unknown[]
}

function getElementSize(el: HTMLElement): { width: number; height: number } {
  const rect = measureRect(el)
  return {
    width: rect.width || el.offsetWidth || 320,
    height: rect.height || el.offsetHeight || 160,
  }
}

function getContainerSize(parent: HTMLElement | null): { width: number; height: number } {
  const rect = measureRect(parent)
  const inner = measureWindow()
  return { width: rect?.width || inner.width, height: rect?.height || inner.height }
}

export function usePopoverPosition(options: PopoverPositionOptions) {
  const { target, at: getAnchor, watchTriggers = [] } = options

  const placedPosition = ref<{ left: number; top: number }>(
    resolvePopoverPosition({
      at: getAnchor(),
      popoverSize: { width: 320, height: 160 },
      containerSize: measureWindow(),
    }),
  )

  const positionStyle = computed(() => ({
    left: `${placedPosition.value.left}px`,
    top: `${placedPosition.value.top}px`,
  }))

  function updatePosition() {
    const el = target.value
    if (!el) return
    const parent = (el.offsetParent as HTMLElement | null) || el.parentElement
    placedPosition.value = resolvePopoverPosition({
      at: getAnchor(),
      popoverSize: getElementSize(el),
      containerSize: getContainerSize(parent),
    })
  }

  let resizeObserver: ResizeObserver | null = null

  onMounted(() => {
    win.addEventListener('resize', updatePosition)
    void nextTick(() => {
      updatePosition()
      if (typeof ResizeObserver !== 'undefined' && target.value) {
        resizeObserver = new ResizeObserver(() => {
          updatePosition()
        })
        resizeObserver.observe(target.value)
        const parent = target.value.offsetParent || target.value.parentElement
        if (parent) resizeObserver.observe(parent)
      }
    })
  })

  onScopeDispose(() => {
    win.removeEventListener('resize', updatePosition)
    resizeObserver?.disconnect()
  })

  for (const trigger of watchTriggers) {
    if (typeof trigger === 'function') {
      watch(
        trigger as () => unknown,
        async () => {
          await nextTick()
          updatePosition()
        },
        { deep: true },
      )
    }
  }

  return {
    placedPosition,
    positionStyle,
    updatePosition,
  }
}
