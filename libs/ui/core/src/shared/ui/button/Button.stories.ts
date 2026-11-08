/**
 * Every color and size a button is rendered in. This is also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * Hover is one expression for both token sets, and the result is a
 * browser-computed value that nothing else can provide. A filled button moves
 * away from its text and an outlined one toward it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { Button } from '.'
import type { ButtonVariants } from '.'
import { hoverOver, lightness } from '@/shared/fixtures/colour'
import { DARK, expectDark } from '@/shared/fixtures/theme'

type Variant = NonNullable<ButtonVariants['variant']>

interface Knobs {
  variant: Variant
  size: NonNullable<ButtonVariants['size']>
  isDisabled: boolean
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
        :disabled="args.isDisabled"
      >{{ variant }}</Button>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Controls/Button',
  component: Button,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A button in the sizes and colours the tokens name. Hover moves the ground a ' +
          'little without lowering the contrast of its text, and focus uses ' +
          'the ring at the width the tokens give it.',
      },
    },
  },
  argTypes: {
    variant: { control: 'inline-radio', options: VARIANTS },
    size: { control: 'inline-radio', options: ['default', 'small', 'icon', 'icon-small'] },
    isDisabled: { control: 'boolean' },
  },
  args: { variant: 'solid', size: 'default', isDisabled: false },
  render: row,
}

export default meta
type Story = StoryObj<Knobs>

/** One of each, live. */
export const Playground: Story = {}

/** The small size, which is what a quiet action beside a heading takes. */
export const Small: Story = { args: { size: 'small' } }

/**
 * The solid button is the default one, such as the answer of a banner. Its
 * ground is painted, so its text does not rest on the window; the text
 * contrast on that ground is measured in `Contrast.stories.ts`.
 */
export const Filled: Story = {
  args: { size: 'small' },
  play: async ({ canvasElement }) => {
    const solid = getComputedStyle(within(canvasElement).getByRole('button', { name: 'solid' }))
    await expect(solid.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  },
}

/** Nothing to press: dimmed, and out of the pointer's way. */
export const Disabled: Story = {
  args: { isDisabled: true },
  play: async ({ canvasElement }) => {
    for (const button of within(canvasElement).getAllByRole('button')) {
      await expect(button).toBeDisabled()
      await expect(Number.parseFloat(getComputedStyle(button).opacity)).toBeLessThan(1)
      await expect(getComputedStyle(button).pointerEvents).toBe('none')
    }
  },
}

/**
 * The dark token set, with the pointer moved along the row.
 * A filled button is light and an outlined one dark on this set, so the hover expression must move the background
 * away from the text color for filled and toward it for outlined, in both directions.
 * A text-only button gains a background.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const canvas = within(canvasElement)

    // The way round the dark set is written: a filled button is dark on light
    // and an outlined one is light on dark.
    const solid = getComputedStyle(canvas.getByRole('button', { name: 'solid' }))
    await expect(lightness(solid.color)).toBeLessThan(lightness(solid.backgroundColor))
    const outline = getComputedStyle(canvas.getByRole('button', { name: 'outline' }))
    await expect(lightness(outline.color)).toBeGreaterThan(lightness(outline.backgroundColor))

    for (const name of ['solid', 'outline'] as const) {
      const button = canvas.getByRole('button', { name })
      const ink = lightness(getComputedStyle(button).color)
      const restingLightness = lightness(getComputedStyle(button).backgroundColor)

      // The two differ, which gives the direction checked below its meaning.
      await expect(Math.abs(ink - restingLightness)).toBeGreaterThan(24)

      await hoverOver(button)
      await waitFor(async () => {
        const moved = lightness(getComputedStyle(button).backgroundColor) - restingLightness
        await expect(Math.abs(moved)).toBeGreaterThan(2)
        const toward = Math.sign(ink - restingLightness)
        await expect(Math.sign(moved)).toBe(name === 'solid' ? -toward : toward)
      })
    }

    // A text-only button has no background at rest and gains one on hover.
    const ghost = canvas.getByRole('button', { name: 'ghost' })
    await expect(getComputedStyle(ghost).backgroundColor).toBe('rgba(0, 0, 0, 0)')
    await hoverOver(ghost)
    await waitFor(() =>
      expect(getComputedStyle(ghost).backgroundColor).not.toBe('rgba(0, 0, 0, 0)'),
    )
  },
}
