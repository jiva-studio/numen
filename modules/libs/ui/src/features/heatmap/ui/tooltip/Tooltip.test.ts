/**
 * Where the tooltip stands: beside the thing it is about, on the near side
 * where the far one has no room, and inside the edge where neither has.
 *
 * The document a test runs in lays nothing out, so the size the tooltip turned
 * out to be is given to it here. Everything the placement does follows from
 * that size and the room it is given.
 */
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Tooltip from './Tooltip.vue'
import type { Box } from '@/shared/lib/place'

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

const about = (held: Partial<Box> = {}): Box => ({
  x: 100,
  y: 200,
  width: 20,
  height: 20,
  ...held,
})

/** Drawn, and placed: the size it turned out to be reaches the style on the
 *  frame after the one it was measured on. */
const mountTooltip = async (props: Record<string, unknown> = {}) => {
  const tooltip = mount(Tooltip, {
    props: { at: about(), viewport: ROOM, ...props },
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

describe('where it stands', () => {
  it('runs on from the far end of the thing it is about, and level with its top', async () => {
    expect(getPlacement(await mountTooltip())).toEqual({ x: '128px', y: '200px' })
  })

  // The far side is off the edge, so it runs back from the near end instead.
  it('takes the near side where the far side has no room for it', async () => {
    expect(getPlacement(await mountTooltip({ at: about({ x: 900 }) })).x).toBe('772px')
  })

  it('is brought inside the edge where neither side has room', async () => {
    const tooltip = await mountTooltip({
      at: about({ x: 60, y: 10 }),
      viewport: { width: 200, height: 800 },
    })
    expect(getPlacement(tooltip).x).toBe('72px')
  })

  it('folds up from the foot of the room rather than running past it', async () => {
    const tooltip = await mountTooltip({
      at: about({ y: 90 }),
      viewport: { width: 1000, height: 100 },
    })
    expect(getPlacement(tooltip).y).toBe('50px')
  })

  it('stands clear of the edge it is against where it is larger than the room', async () => {
    const tooltip = await mountTooltip({
      at: about({ x: 10, y: 10 }),
      viewport: { width: 60, height: 800 },
    })
    expect(getPlacement(tooltip).x).toBe('8px')
  })
})

describe('the thing it is about moving', () => {
  it('is placed again beside where that thing now is', async () => {
    const tooltip = await mountTooltip()
    expect(getPlacement(tooltip).x).toBe('128px')

    await tooltip.setProps({ at: about({ x: 400 }) })
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
    const tooltip = await mountTooltip({ at: about({ x: 900 }), viewport: null })
    expect(getPlacement(tooltip).x).toBe('928px')

    window.innerWidth = 1000
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(getPlacement(tooltip).x).toBe('772px')

    window.innerWidth = was
  })
})
