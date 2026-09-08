/**
 * The shape a value will take, held open while the value is worked out.
 *
 * The stories are a list filling row by row, because that is where the claim
 * lives: a shape the size of what replaces it leaves the rows where they are.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Skeleton from './Skeleton.vue'
import { lightness } from '@/shared/fixtures/colour'
import { DARK, drawnDark } from '@/shared/fixtures/theme'

const meta = {
  title: 'Flash Cards/Skeleton',
  component: Skeleton,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A filled shape standing in the room a value will take while that ' +
          'value is being worked out. It is drawn at the size of what ' +
          'replaces it, so nothing moves when the value lands, and it says ' +
          'that something is coming.',
      },
    },
  },
  argTypes: {
    wide: { control: 'text' },
    high: { control: 'text' },
    pill: { control: 'boolean' },
  },
  args: { wide: '1.5rem', high: '0.75em', pill: true },
} satisfies Meta<typeof Skeleton>

export default meta
type Story = StoryObj<typeof meta>

/** On its own, at the size a count of two or three figures takes. */
export const Playground: Story = {}

/**
 * The claim, drawn twice: a list where every count has arrived, and the same
 * list where none has. The rows stand in the same places in both.
 */
export const ARowThatDoesNotMove: Story = {
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args, rows: ['Studies', 'Sanskrit', 'Птицы', 'ᬩᬮᬶ'] }),
    template: `
      <div class="numen" style="display:flex;gap:32px;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <ul v-for="(counted, at) in [true, false]" :key="at" style="margin:0;padding:0;list-style:none;inline-size:11rem">
          <li v-for="(name, row) in rows" :key="name" style="display:flex;align-items:center;gap:12px;padding:0.3rem 0.6rem">
            <span style="flex:1">{{ name }}</span>
            <span v-if="counted" style="flex:none;min-inline-size:1.5rem;padding:0.0625rem 0.4rem;text-align:center;border-radius:var(--numen-radius-pill);background:var(--numen-highlight);color:var(--numen-caution-fg);font-size:var(--numen-edge-label-size);font-variant-numeric:tabular-nums">{{ [7, 128, 0, 42][row] }}</span>
            <span v-else style="flex:none;min-inline-size:1.5rem;padding:0.0625rem 0.4rem;text-align:center;border-radius:var(--numen-radius-pill);background:var(--numen-highlight);color:var(--numen-caution-fg);font-size:var(--numen-edge-label-size)"><Skeleton v-bind="args" /></span>
          </li>
        </ul>
      </div>
    `,
  }),
}

/** A line of text, where what is coming is a name. */
export const InPlaceOfWords: Story = {
  args: { wide: '9rem', high: '1em', pill: false },
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="inline-size:280px;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <p style="margin:0 0 6px;color:var(--numen-edge-label);font-size:var(--numen-edge-label-size)">Preset</p>
        <Skeleton v-bind="args" />
      </div>
    `,
  }),
}

/** Filling the whole of what holds it, which is what it does given no width. */
export const AsWideAsWhatHoldsIt: Story = {
  args: { wide: '100%', high: '3rem', pill: false },
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="inline-size:320px;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <Skeleton v-bind="args" />
      </div>
    `,
  }),
}

/** On a filled ground, where the fill has to read against the colour under it. */
export const OnAFilledGround: Story = {
  args: { wide: '4rem', high: '1em', pill: true },
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-accent);color:var(--numen-accent-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <span style="display:inline-flex;align-items:center;gap:8px">
          <span>Counting</span>
          <Skeleton v-bind="args" />
        </span>
      </div>
    `,
  }),
}

/**
 * On the dark set of tokens, on the surface and on a filled ground. The fill is
 * taken from the text of whatever holds it, so it has to be told from the
 * ground under it in both places.
 */
export const Dark: Story = {
  globals: DARK,
  args: { wide: '4rem', high: '1em', pill: true },
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;flex-direction:column;gap:16px;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <span data-ground style="padding:8px;background:var(--numen-surface)"><Skeleton v-bind="args" /></span>
        <span data-ground style="padding:8px;background:var(--numen-accent);color:var(--numen-accent-ink)"><Skeleton v-bind="args" /></span>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)

    const grounds = canvasElement.querySelectorAll<HTMLElement>('[data-ground]')
    await expect(grounds).toHaveLength(2)
    for (const ground of grounds) {
      const shape = ground.querySelector<HTMLElement>('.skeleton')!
      const fill = getComputedStyle(shape).backgroundColor
      // Something is there, and it is not the colour of what it stands on.
      await expect(fill).not.toBe('rgba(0, 0, 0, 0)')
      await expect(fill).not.toBe(getComputedStyle(ground).backgroundColor)
    }

    // The shape is drawn from the ink of whatever it stands on. On the dark set
    // that is a light shape on the surface and a dark one on the accent.
    const laidOn = (ground: HTMLElement) => {
      const behind = getComputedStyle(ground).backgroundColor
      const shape = ground.querySelector<HTMLElement>('.skeleton')!
      return lightness(getComputedStyle(shape).backgroundColor, behind) - lightness(behind)
    }
    await expect(laidOn(grounds[0]!)).toBeGreaterThan(2)
    await expect(laidOn(grounds[1]!)).toBeLessThan(-2)
  },
}

/** Far too many of them at once, which is a list nothing has answered for yet. */
export const FarTooMany: Story = {
  args: { wide: '100%', high: '0.9rem', pill: false },
  render: (args) => ({
    components: { Skeleton },
    setup: () => ({ args, rows: Array.from({ length: 40 }, (_, at) => at) }),
    template: `
      <div class="numen" style="display:flex;flex-direction:column;gap:6px;inline-size:320px;padding:24px;background:var(--numen-surface);color:var(--numen-ink)">
        <Skeleton v-for="row in rows" :key="row" v-bind="args" />
      </div>
    `,
  }),
}
