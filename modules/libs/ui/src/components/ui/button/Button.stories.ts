/**
 * Every colour and size a button is drawn in. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`.
 *
 * Hover is one expression for both sets of tokens — a little of a variant's own
 * text mixed into its ground — so what it comes to is a browser's answer and
 * nothing else can give it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { Button } from '.'
import type { ButtonVariants } from '.'
import { hovered, lightness } from '@/fixtures/colour'
import { DARK, drawnDark } from '@/fixtures/theme'

type Variant = NonNullable<ButtonVariants['variant']>

interface Knobs {
  variant: Variant
  size: NonNullable<ButtonVariants['size']>
  disabled: boolean
}

/** The three variants side by side, each under its own name. */
const VARIANTS: readonly Variant[] = ['solid', 'outline', 'ghost']

const row = (args: Knobs) => ({
  components: { Button },
  setup: () => ({ args, variants: VARIANTS }),
  template: `
    <div class="numen" style="display:flex;align-items:center;gap:16px;padding:32px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
      <Button
        v-for="variant in variants"
        :key="variant"
        :variant="variant"
        :size="args.size"
        :disabled="args.disabled"
      >{{ variant }}</Button>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Generic/Button',
  component: Button,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A button in the sizes and colours the tokens name. Hover mixes a ' +
          "little of the variant's own text into its ground, and focus wears " +
          'the ring at the width the tokens give it.',
      },
    },
  },
  argTypes: {
    variant: { control: 'inline-radio', options: VARIANTS },
    size: { control: 'inline-radio', options: ['default', 'small', 'icon', 'icon-small'] },
    disabled: { control: 'boolean' },
  },
  args: { variant: 'solid', size: 'default', disabled: false },
  render: row,
}

export default meta
type Story = StoryObj<Knobs>

/** One of each, live. */
export const Playground: Story = {}

/** The small size, which is what a quiet action beside a heading takes. */
export const Small: Story = { args: { size: 'small' } }

/** Nothing to press: dimmed, and out of the pointer's way. */
export const Disabled: Story = {
  args: { disabled: true },
  play: async ({ canvasElement }) => {
    for (const button of within(canvasElement).getAllByRole('button')) {
      expect(button).toBeDisabled()
      expect(Number.parseFloat(getComputedStyle(button).opacity)).toBeLessThan(1)
      expect(getComputedStyle(button).pointerEvents).toBe('none')
    }
  },
}

/**
 * On the dark set of tokens, with the pointer walked along the row.
 *
 * Hover mixes a little of a button's own text into its ground, so the ground
 * moves towards that text. On the dark set a filled button is light and an
 * outlined one is dark, and the one expression has to answer for both ways
 * round. A button that is only text takes a ground it did not have.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)
    const canvas = within(canvasElement)

    // The way round the dark set is written: a filled button is dark on light
    // and an outlined one is light on dark.
    const solid = getComputedStyle(canvas.getByRole('button', { name: 'solid' }))
    expect(lightness(solid.color)).toBeLessThan(lightness(solid.backgroundColor))
    const outline = getComputedStyle(canvas.getByRole('button', { name: 'outline' }))
    expect(lightness(outline.color)).toBeGreaterThan(lightness(outline.backgroundColor))

    for (const name of ['solid', 'outline'] as const) {
      const button = canvas.getByRole('button', { name })
      const ink = lightness(getComputedStyle(button).color)
      const resting = lightness(getComputedStyle(button).backgroundColor)

      // The two are told apart, which is what makes the direction below mean
      // anything.
      expect(Math.abs(ink - resting)).toBeGreaterThan(24)

      await hovered(button)
      await waitFor(() => {
        const moved = lightness(getComputedStyle(button).backgroundColor) - resting
        expect(Math.abs(moved)).toBeGreaterThan(2)
        expect(Math.sign(moved)).toBe(Math.sign(ink - resting))
      })
    }

    // A button that is only text has no ground at rest, and gains one under
    // the hand.
    const ghost = canvas.getByRole('button', { name: 'ghost' })
    expect(getComputedStyle(ghost).backgroundColor).toBe('rgba(0, 0, 0, 0)')
    await hovered(ghost)
    await waitFor(() =>
      expect(getComputedStyle(ghost).backgroundColor).not.toBe('rgba(0, 0, 0, 0)'),
    )
  },
}
