/** Every cap that can be drawn. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import KeyCap from './KeyCap.vue'
import { ICONS } from './icons'
import type { Keystroke, KeyIcon } from './keys'
import { DARK, expectDark } from '@/shared/fixtures/theme'

const meta = {
  title: 'Application/KeyCap',
  component: KeyCap,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof KeyCap>

export default meta
type Story = StoryObj<typeof meta>

/** Every key that can be held, in the order the table declares them. */
const EVERY = Object.keys(ICONS) as readonly KeyIcon[]

const row = (caps: readonly Keystroke[]) => ({
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

/** A cap that is icons alone, which is what the foot of a list draws. */
export const Icons: Story = { args: { keys: { icons: ['shift', 'return'], letter: '' } } }

/** A cap with every icon at once, a combination no keyboard produces. */
export const FarTooMany: Story = { args: { keys: { icons: EVERY, letter: 'W' } } }

/** Every key on its own, beside the letter it would be held with. */
export const EveryIcon: Story = {
  args: { keys: { icons: [], letter: 'W' } },
  render: ({ keys }) => row(EVERY.map((icon) => ({ icons: [icon], letter: keys.letter }))),
  play: async ({ canvasElement }) => {
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    await expect(caps).toHaveLength(EVERY.length)
  },
}

/** Every key with the dark token set, where the cap background is a blend. */
export const Dark: Story = {
  globals: DARK,
  args: { keys: { icons: [], letter: 'W' } },
  render: ({ keys }) => row(EVERY.map((icon) => ({ icons: [icon], letter: keys.letter }))),
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const caps = Array.from(canvasElement.querySelectorAll('.cap'))
    await expect(caps).toHaveLength(EVERY.length)
  },
}

/** The text alternative that a screen reader announces. */
export const Announced: Story = {
  args: { keys: { icons: ['control', 'shift'], letter: 'W' } },
  play: async ({ canvasElement }) => {
    // Everything drawn is hidden from a reader who is listening, so the
    // keystroke is heard once and in the order it is held.
    await expect(within(canvasElement).getByText('Control Shift W')).toHaveClass('sr-only')

    // Two icons and the letter held with them.
    const viewElement = Array.from(canvasElement.querySelectorAll('.cap > :not(.sr-only)'))
    await expect(viewElement).toHaveLength(3)
    for (const one of viewElement) {
      await expect(one).toHaveAttribute('aria-hidden', 'true')
    }
  },
}
