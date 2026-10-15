/**
 * Virtual tree reactive state, viewport measurement and scrolling.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useVirtualTree } from './virtual'

describe('useVirtualTree', () => {
  it('falls back to showing all rows when container is unmeasured', () => {
    const container = ref<HTMLElement | null>(null)
    const virtual = useVirtualTree(
      container,
      () => 50,
      () => 24,
    )

    expect(virtual.window.value.startIndex).toBe(0)
    expect(virtual.window.value.endIndex).toBe(50)
    expect(virtual.window.value.totalHeight).toBe(1200)
    expect(virtual.window.value.offsetY).toBe(0)
  })

  it('updates window when viewport is measured', () => {
    const el = {
      clientHeight: 240,
      scrollTop: 0,
    } as unknown as HTMLElement

    const container = ref<HTMLElement | null>(el)
    const virtual = useVirtualTree(
      container,
      () => 50,
      () => 24,
    )

    virtual.measureViewport()

    expect(virtual.window.value.startIndex).toBe(0)
    expect(virtual.window.value.endIndex).toBe(15)
  })

  it('updates window on scroll', () => {
    const el = {
      clientHeight: 240,
      scrollTop: 0,
    } as unknown as HTMLElement

    const container = ref<HTMLElement | null>(el)
    const virtual = useVirtualTree(
      container,
      () => 50,
      () => 24,
    )

    virtual.measureViewport()

    el.scrollTop = 240
    virtual.onScroll()

    expect(virtual.window.value.startIndex).toBe(5)
    expect(virtual.window.value.endIndex).toBe(25)
    expect(virtual.window.value.offsetY).toBe(120)
  })

  it('scrolls to row below current viewport', () => {
    const el = {
      clientHeight: 240,
      scrollTop: 0,
    } as unknown as HTMLElement

    const container = ref<HTMLElement | null>(el)
    const virtual = useVirtualTree(
      container,
      () => 50,
      () => 24,
    )

    virtual.measureViewport()
    virtual.scrollToRow(20)

    expect(el.scrollTop).toBe(264)
    expect(virtual.window.value.startIndex).toBe(6)
  })

  it('does nothing when scrolling with null container', () => {
    const container = ref<HTMLElement | null>(null)
    const virtual = useVirtualTree(
      container,
      () => 50,
      () => 24,
    )

    virtual.scrollToRow(10)
    expect(virtual.window.value.startIndex).toBe(0)
  })
})
