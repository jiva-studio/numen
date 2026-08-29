/**
 * What a person did on each day, as a grid of weeks.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws — and how many weeks fit the
 * room there is can only be answered by a browser that laid them out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'

import Heatmap from './Heatmap.vue'
import { names, ROWS } from './heatmap'

const meta = {
  title: 'Generic/Heatmap',
  component: Heatmap,
} satisfies Meta<typeof Heatmap>

export default meta
type Story = StoryObj<typeof meta>

/** A day this many days before the one the stories stand on. */
const now = new Date(2026, 7, 29, 12)

/** worked is a year of days, most of them answered on. */
function worked(): Map<string, number> {
  const out = new Map<string, number>()
  for (let back = 0; back < 300; back += 1) {
    const on = new Date(now)
    on.setDate(on.getDate() - back)
    // Sundays off, and the rest a hand of cards.
    if (on.getDay() === 0) continue
    out.set(names(on), ((back * 7) % 60) + 1)
  }
  return out
}

const cells = (canvas: HTMLElement) => canvas.querySelectorAll('.heatmap__day')

/** A year of answers, in the room a window gives it. */
export const AYear: Story = {
  args: { did: worked(), now },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 640px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = cells(canvasElement)
    await expect(drawn.length % ROWS).toBe(0)
    await expect(drawn.length).toBeGreaterThan(ROWS * 20)
    // Today is ringed, and there is one of it.
    await expect(canvasElement.querySelectorAll('[data-today]')).toHaveLength(1)
  },
}

/**
 * The same year in half the room. A narrow window shows fewer weeks rather than
 * a grid standing in the middle of empty room, and the cells are the size they
 * were.
 */
export const Narrow: Story = {
  args: { did: worked(), now },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 240px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = cells(canvasElement)
    await expect(drawn.length % ROWS).toBe(0)

    const one = drawn[0] as SVGRectElement | undefined
    await expect(one?.getAttribute('width')).toBe('11')

    // The grid reaches the far edge, rather than ending short of it.
    const grid = canvasElement.querySelector('.heatmap__grid') as SVGSVGElement
    await expect(Math.round(grid.getBoundingClientRect().width)).toBeGreaterThan(230)
  },
}

/** A vault nobody has answered: every day drawn, none of them filled. */
export const Nothing: Story = {
  args: { did: new Map<string, number>(), now },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 480px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = [...cells(canvasElement)]
    await expect(drawn.length).toBeGreaterThan(0)
    for (const one of drawn) {
      await expect(one.getAttribute('data-weight')).toBe('0')
    }
  },
}
