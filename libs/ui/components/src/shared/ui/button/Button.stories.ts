/** Every color and size a button is rendered in. This is also the test corpus: each story is run in a browser by `@storybook/addon-vitest`. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { Button } from '.'
import type { ButtonVariants } from '.'
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

/** Nothing to press. */
export const Disabled: Story = {
  args: { isDisabled: true },
  play: async ({ canvasElement }) => {
    for (const button of within(canvasElement).getAllByRole('button')) {
      await expect(button).toBeDisabled()
    }
  },
}

/** The dark token set, with every variant still a button. */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const canvas = within(canvasElement)
    for (const name of ['solid', 'outline', 'ghost']) {
      await expect(canvas.getByRole('button', { name })).toBeEnabled()
    }
  },
}
