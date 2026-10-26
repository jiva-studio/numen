/**
 * Where the tooltip stands: beside the thing it is about, on the near side
 * where the far one has no room, and inside the edge where neither has.
 *
 * The document a test runs in lays nothing out, so the size the tooltip turned
 * out to be is given to it here. Everything the placement does follows from
 * that size and the room it is given.
 */
import { mount } from '@vue/test-utils'
import { browserViewport, type Viewport } from '@numen/ui'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Tooltip from './Tooltip.vue'
import type { Box } from '@numen/ui'

/** How large the tooltip turned out to be, once it was drawn. */
const WIDE = 120
const HIGH = 40

beforeEach(() => {
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue(
    new DOMRect(0, 0, WIDE, HIGH),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
})

const ROOM = { width: 1000, height: 800 }

/** A window of the size a test names, whatever the machine laid out. */
const createViewport = (size: { width: number; height: number }): Viewport => ({
  watch: () => () => {},
  watchWhole: () => () => {},
  watchRoom: (took) => {
    took(size)
    return () => {}
  },
})

const createBox = (overrides: Partial<Box> = {}): Box => ({
  x: 100,
  y: 200,
  width: 20,
  height: 20,
  ...overrides,
})

/** Drawn, and placed: the size it turned out to be reaches the style on the
 *  frame after the one it was measured on. */
const mountTooltip = async (props: Record<string, unknown> = {}) => {
  const tooltip = mount(Tooltip, {
    props: { at: createBox(), viewport: createViewport(ROOM), ...props },
    slots: { default: 'What this is' },
  })
  await nextTick()
  return tooltip
}

/** Where it was placed, as the two numbers the style carries. */
const getPlacement = (tooltip: Awaited<ReturnType<typeof mountTooltip>>) => {
  const style = tooltip.get('[role="tooltip"]').element as HTMLElement
  return { x: style.style.insetInlineStart, y: style.style.insetBlockStart }
}

describe('the tooltip itself', () => {
  it('is announced as one, and says what it was given to say', async () => {
    const tooltip = await mountTooltip()
    expect(tooltip.findAll('[role="tooltip"]')).toHaveLength(1)
    expect(tooltip.text()).toBe('What this is')
  })
})

describe('at its current position', () => {
  it('runs on from the far end of the thing it is about, and level with its top', async () => {
    expect(getPlacement(await mountTooltip())).toEqual({ x: '128px', y: '200px' })
  })

  // The far side is off the edge, so it runs back from the near end instead.
  it('takes the near side where the far side has no room for it', async () => {
    expect(getPlacement(await mountTooltip({ at: createBox({ x: 900 }) })).x).toBe('772px')
  })

  it('is brought inside the edge where neither side has room', async () => {
    const tooltip = await mountTooltip({
      at: createBox({ x: 60, y: 10 }),
      viewport: createViewport({ width: 200, height: 800 }),
    })
    expect(getPlacement(tooltip).x).toBe('72px')
  })

  it('folds up from the foot of the room and not running past it', async () => {
    const tooltip = await mountTooltip({
      at: createBox({ y: 90 }),
      viewport: createViewport({ width: 1000, height: 100 }),
    })
    expect(getPlacement(tooltip).y).toBe('50px')
  })

  it('clamps tooltip position when larger than viewport bounds', async () => {
    const tooltip = await mountTooltip({
      at: createBox({ x: 10, y: 10 }),
      viewport: createViewport({ width: 60, height: 800 }),
    })
    expect(getPlacement(tooltip).x).toBe('8px')
  })
})

describe('the thing it is about moving', () => {
  it('is placed again alongside where that thing now is', async () => {
    const tooltip = await mountTooltip()
    expect(getPlacement(tooltip).x).toBe('128px')

    await tooltip.setProps({ at: createBox({ x: 400 }) })
    await nextTick()
    expect(getPlacement(tooltip)).toEqual({ x: '428px', y: '200px' })
  })
})

describe('the room it is placed in', () => {
  // Told nothing about the room, it stays inside the window, and the window is
  // measured again whenever it changes size.
  it('is the window where a caller measures none of its own', async () => {
    const was = window.innerWidth
    window.innerWidth = 2000
    const tooltip = await mountTooltip({ at: createBox({ x: 900 }), viewport: browserViewport })
    expect(getPlacement(tooltip).x).toBe('928px')

    window.innerWidth = 1000
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(getPlacement(tooltip).x).toBe('772px')

    window.innerWidth = was
  })
})
