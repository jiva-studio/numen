import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { measureWindow } from '@numen/ui'
import { getDraggedHeight, getTallest, LEAST } from '../lib/frame'
import { createHeightStore, type HeightStore } from '../model/heightStore'

interface HeldBar {
  bar: HTMLElement
  pointer: number
  from: number
  was: number
}

function findPane(root: HTMLElement | null): HTMLElement | null {
  for (let at = root?.parentElement ?? null; at; at = at.parentElement) {
    if (getComputedStyle(at).containerType.split(' ').includes('size')) return at
  }
  return null
}

/**
 * The height a player is drawn at, which a person changes by dragging the bar below it or with the
 * arrow keys, and which is remembered in the store. `media` is the player element.
 */
export function useFrameHeight(
  givenStore: HeightStore | undefined,
  root: Readonly<Ref<HTMLElement | null>>,
  media: Readonly<Ref<HTMLElement | null>>,
) {
  const store = givenStore ?? createHeightStore()
  /** How tall the player is drawn, remembered on this machine. */
  const tall = ref(store.read())
  const held = ref<HeldBar | null>(null)
  /** The rendered height of the player and the tallest it can be drawn, for the resize handle. */
  const renderedHeight = ref<number | null>(null)
  const tallest = ref(LEAST)
  const frameStyle = computed(() =>
    tall.value === null ? undefined : { blockSize: `${tall.value}px` },
  )
  const handleValue = computed(() =>
    Math.min(tallest.value, Math.max(LEAST, renderedHeight.value ?? tall.value ?? LEAST)),
  )
  const isHeld = computed(() => held.value !== null)

  const getPaneHeight = (): number => findPane(root.value)?.clientHeight || measureWindow().height

  watch(
    media,
    (element, _, onCleanup) => {
      if (!element) return
      const sizes = new ResizeObserver(() => {
        renderedHeight.value = Math.round(element.clientHeight)
        tallest.value = getTallest(getPaneHeight())
        if (tall.value !== null) tall.value = Math.min(tallest.value, tall.value)
      })
      sizes.observe(element)
      const pane = findPane(root.value)
      if (pane) sizes.observe(pane)
      onCleanup(() => sizes.disconnect())
    },
    { flush: 'post' },
  )

  onMounted(() => {
    if (tall.value !== null) {
      tall.value = Math.min(getTallest(getPaneHeight()), Math.max(LEAST, tall.value))
    }
  })

  // A drag the page is taken out from under lets go of the bar it was holding.
  onBeforeUnmount(() => onPointerUp())

  function onPointerDown(at: PointerEvent): void {
    const bar = at.currentTarget as HTMLElement
    bar.setPointerCapture(at.pointerId)
    held.value = {
      bar,
      pointer: at.pointerId,
      from: at.clientY,
      was: bar.previousElementSibling?.clientHeight ?? LEAST,
    }
    bar.addEventListener('pointermove', onPointerMove)
    bar.addEventListener('pointerup', onPointerUp)
    bar.addEventListener('pointercancel', onPointerUp)
  }

  function onPointerMove(at: PointerEvent): void {
    if (!held.value) return
    tall.value = getDraggedHeight(held.value.was, at.clientY - held.value.from, getPaneHeight())
  }

  function onPointerUp(): void {
    const grip = held.value
    if (!grip) return
    held.value = null
    grip.bar.removeEventListener('pointermove', onPointerMove)
    grip.bar.removeEventListener('pointerup', onPointerUp)
    grip.bar.removeEventListener('pointercancel', onPointerUp)
    if (grip.bar.hasPointerCapture(grip.pointer)) grip.bar.releasePointerCapture(grip.pointer)
    if (tall.value !== null) store.write(tall.value)
  }

  function onHandleKeydown(event: KeyboardEvent): void {
    if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return
    event.preventDefault()
    const bar = event.currentTarget as HTMLElement
    const current = tall.value ?? bar.previousElementSibling?.clientHeight ?? LEAST
    const delta = event.key === 'ArrowDown' ? 16 : -16
    tall.value = getDraggedHeight(current, delta, getPaneHeight())
    store.write(tall.value)
  }

  return { tall, isHeld, tallest, handleValue, frameStyle, onPointerDown, onHandleKeydown }
}
