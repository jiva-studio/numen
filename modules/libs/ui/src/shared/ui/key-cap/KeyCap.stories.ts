/**
 * Every cap the palette can be asked to draw, and the two measurements that say
 * a cap is one object: every icon is centred on the line the letter is set on,
 * and every icon is stroked as thick as every other.
 *
 * Both are measured in the browser that paints the ink, against the cap's own
 * boxes, which come from the tokens.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import KeyCap from './KeyCap.vue'
import { ICONS } from './icons'
import type { PaletteKeys, PaletteIcon } from './keys'
import { lightness } from '@/shared/fixtures/colour'
import { drawingInk } from '@/shared/fixtures/ink'
import { DARK, drawnDark } from '@/shared/fixtures/theme'

const meta = {
  title: 'Application/KeyCap',
  component: KeyCap,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof KeyCap>

export default meta
type Story = StoryObj<typeof meta>

/** Every key that can be held, in the order the table declares them. */
const EVERY = Object.keys(ICONS) as readonly PaletteIcon[]

/** A row of caps, so a page of them can be looked at side by side. */
const row = (caps: readonly PaletteKeys[]) => ({
  components: { KeyCap },
  setup: () => ({ caps }),
  template: `<div style="display:flex;align-items:center;gap:1rem;padding:2rem">
    <KeyCap v-for="(one, at) in caps" :key="at" :keys="one" />
  </div>`,
})

/** A cap holding nothing but a letter. */
export const Bare: Story = { args: { keys: { icons: [], letter: 'K' } } }

/** A cap holding one key. */
export const One: Story = { args: { keys: { icons: ['command'], letter: 'K' } } }

/** A cap that is icons alone, which is what the foot of the palette draws. */
export const Icons: Story = { args: { keys: { icons: ['shift', 'return'], letter: '' } } }

/** A cap holding every key there is, which no keyboard asks for. */
export const FarTooMany: Story = { args: { keys: { icons: EVERY, letter: 'W' } } }

/** Every key on its own, beside the letter it would be held with. */
export const EveryIcon: Story = {
  args: { keys: { icons: [], letter: 'W' } },
  render: ({ keys }) => row(EVERY.map((icon) => ({ icons: [icon], letter: keys.letter }))),
  play: async ({ canvasElement }) => {
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    await expect(caps).toHaveLength(EVERY.length)

    const weights: number[] = []
    for (const cap of caps) {
      const set = parseFloat(getComputedStyle(cap).fontSize)
      const box = cap.getBoundingClientRect()
      const middle = box.top + box.height / 2

      // The line the letter is set on is the middle of the box holding it,
      // which is one line-height tall whatever the face.
      const line = cap.querySelector('span[aria-hidden="true"]')!.getBoundingClientRect()
      await expect(Math.abs(line.top + line.height / 2 - middle) / set).toBeLessThanOrEqual(0.02)

      // One line: an icon's ink is centred on that line, within a fiftieth of
      // the type the cap is set in. What is left over is the asymmetry of the
      // icon's own grid.
      const icon = await drawingInk(cap.querySelector('svg')!)
      await expect(Math.abs(icon.middle - middle) / set).toBeLessThanOrEqual(0.02)
      weights.push(icon.stroke / set)
    }

    // One weight: every icon's ink is as thick as every other's, which is what
    // the thinner stroke of an icon drawn larger is for.
    await expect(Math.max(...weights) / Math.min(...weights) - 1).toBeLessThanOrEqual(0.05)
  },
}

/** A cap is one height whatever it holds. */
export const OneHeight: Story = {
  args: { keys: { icons: [], letter: 'W' } },
  render: ({ keys }) =>
    row([
      { icons: [], letter: keys.letter },
      { icons: ['command'], letter: keys.letter },
      { icons: ['control', 'shift'], letter: keys.letter },
      { icons: EVERY, letter: keys.letter },
      { icons: ['shift', 'return'], letter: '' },
    ]),
  play: async ({ canvasElement }) => {
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    const heights = new Set(caps.map((cap) => Math.round(cap.getBoundingClientRect().height)))
    await expect(heights.size).toBe(1)
  },
}

/** Every key on the dark set of tokens, where the cap's ground is a blend. */
export const Dark: Story = {
  globals: DARK,
  args: { keys: { icons: [], letter: 'W' } },
  render: ({ keys }) => row(EVERY.map((icon) => ({ icons: [icon], letter: keys.letter }))),
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    await expect(caps).toHaveLength(EVERY.length)

    // A cap keeps a ground of its own, told from the surface the row of them
    // stands on, and is read in ink standing above that ground — which is the
    // way round the dark set is written.
    const behind = getComputedStyle(canvasElement.ownerDocument.documentElement).backgroundColor
    const surface = lightness(behind)
    for (const cap of caps) {
      const drawn = getComputedStyle(cap)
      const ground = lightness(drawn.backgroundColor, behind)
      await expect(Math.abs(ground - surface)).toBeGreaterThan(4)
      await expect(lightness(drawn.color, drawn.backgroundColor)).toBeGreaterThan(ground + 24)
    }
  },
}

/** What a reader who is listening rather than looking is told. */
export const Announced: Story = {
  args: { keys: { icons: ['control', 'shift'], letter: 'W' } },
  play: async ({ canvasElement }) => {
    // Everything drawn is hidden from a reader who is listening, so the
    // keystroke is heard once and in the order it is held.
    await expect(within(canvasElement).getByText('Control Shift W')).toHaveClass('sr-only')

    // Two icons and the letter held with them.
    const drawn = Array.from(canvasElement.querySelectorAll('.cap > :not(.sr-only)'))
    await expect(drawn).toHaveLength(3)
    for (const one of drawn) {
      await expect(one).toHaveAttribute('aria-hidden', 'true')
    }
  },
}
