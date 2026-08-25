/**
 * Every cap the palette can be asked to draw, and the two measurements that say
 * a cap is one object: every mark is centred on the letter's ink, and every
 * mark is stroked as thick as the letter's stem.
 *
 * Those measurements are why these are stories rather than tests in jsdom. A
 * mark's ink is where a browser paints it, and nothing else can say where.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import KeyCap from './KeyCap.vue'
import { MARKS } from './marks'
import type { PaletteKeys, PaletteMark } from './model'
import { letterInk, markInk } from '@/fixtures/ink'

const meta = {
  title: 'Generic/KeyCap',
  component: KeyCap,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof KeyCap>

export default meta
type Story = StoryObj<typeof meta>

/** Every key that can be held, in the order the table declares them. */
const EVERY = Object.keys(MARKS) as readonly PaletteMark[]

/** A row of caps, so a page of them can be looked at side by side. */
const row = (caps: readonly PaletteKeys[]) => ({
  components: { KeyCap },
  setup: () => ({ caps }),
  template: `<div style="display:flex;align-items:center;gap:1rem;padding:2rem">
    <KeyCap v-for="(one, at) in caps" :key="at" :keys="one" />
  </div>`,
})

/** A cap holding nothing but a letter. */
export const Bare: Story = { args: { keys: { marks: [], letter: 'K' } } }

/** A cap holding one key. */
export const One: Story = { args: { keys: { marks: ['command'], letter: 'K' } } }

/** A cap that is marks alone, which is what the foot of the palette draws. */
export const Marks: Story = { args: { keys: { marks: ['shift', 'return'], letter: '' } } }

/** A cap holding every key there is, which no keyboard asks for. */
export const FarTooMany: Story = { args: { keys: { marks: EVERY, letter: 'W' } } }

/** Every key on its own, beside the letter it would be held with. */
export const EveryMark: Story = {
  args: { keys: { marks: [], letter: 'W' } },
  render: ({ keys }) => row(EVERY.map((mark) => ({ marks: [mark], letter: keys.letter }))),
  play: async ({ canvasElement }) => {
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    await expect(caps).toHaveLength(EVERY.length)

    for (const cap of caps) {
      // The letter beside it is what a mark is measured against.
      const letter = letterInk(cap.querySelector('span[aria-hidden="true"]')!)
      const mark = await markInk(cap.querySelector('svg')!)
      const set = parseFloat(getComputedStyle(cap).fontSize)

      // One line: a mark's ink is centred on the letter's, within a fiftieth
      // of the type they are both drawn at.
      await expect(Math.abs(mark.middle - letter.middle) / set).toBeLessThanOrEqual(0.02)

      // One weight: a mark is stroked within a tenth of the letter's stem.
      await expect(Math.abs(mark.stroke - letter.stroke) / letter.stroke).toBeLessThanOrEqual(0.1)
    }
  },
}

/** A cap is one height whatever it holds. */
export const OneHeight: Story = {
  args: { keys: { marks: [], letter: 'W' } },
  render: ({ keys }) =>
    row([
      { marks: [], letter: keys.letter },
      { marks: ['command'], letter: keys.letter },
      { marks: ['control', 'shift'], letter: keys.letter },
      { marks: EVERY, letter: keys.letter },
      { marks: ['shift', 'return'], letter: '' },
    ]),
  play: async ({ canvasElement }) => {
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    const heights = new Set(caps.map((cap) => Math.round(cap.getBoundingClientRect().height)))
    await expect(heights.size).toBe(1)
  },
}

/** What a reader who is listening rather than looking is told. */
export const Announced: Story = {
  args: { keys: { marks: ['control', 'shift'], letter: 'W' } },
  play: async ({ canvasElement }) => {
    // Everything drawn is hidden from a reader who is listening, so the
    // keystroke is heard once and in the order it is held.
    await expect(within(canvasElement).getByText('Control Shift W')).toHaveClass('sr-only')
    for (const drawn of canvasElement.querySelectorAll('.cap > :not(.sr-only)')) {
      await expect(drawn).toHaveAttribute('aria-hidden', 'true')
    }
  },
}
