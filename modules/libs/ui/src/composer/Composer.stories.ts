/**
 * What the composer looks like. What it does is asserted in `Composer.test.ts`.
 *
 * `working` and `disabled` are controls, not stories.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn } from 'storybook/test'
import { ref } from 'vue'
import Composer from './Composer.vue'
import { ARABIC, DEVANAGARI, LINK, LONG, MULTILINE } from '@/fixtures/prose'

const meta = {
  title: 'Chat/Composer',
  component: Composer,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'The field a message is written in, and the button that sends it. ' +
          'Enter sends and Shift+Enter breaks the line; while an input ' +
          'method is composing, Enter belongs to the input method and never ' +
          'reaches the message. It grows with what is typed until it reaches ' +
          'the height the tokens allow, then scrolls. While an answer is on ' +
          'its way the button gives its place to three dots.',
      },
    },
  },
  argTypes: {
    placeholder: { control: 'text' },
    working: { control: 'boolean' },
    disabled: { control: 'boolean' },
  },
  args: { placeholder: 'Write a message', working: false, disabled: false, onSubmit: fn() },
} satisfies Meta<typeof Composer>

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

/**
 * Every story holds its own text, as the application would: the composer says
 * what was written and leaves clearing it to whoever answers.
 */
const holding = (...starts: string[]): Render => (args) => ({
  components: { Composer },
  setup: () => ({ args, texts: starts.map((start) => ref(start)) }),
  template: `
    <div class="numen flex w-[420px] max-w-[calc(100vw-2rem)] flex-col gap-4">
      <Composer v-for="(text, index) in texts" :key="index" v-bind="args" v-model="text.value" />
    </div>
  `,
})

/** Empty, and with a line in it. Turn `working` on for the dots. */
export const Playground: Story = {
  render: holding('', 'What does a plex draw?'),
  play: async ({ canvasElement }) => {
    // One line of typing stands as tall as the button, which is what puts the
    // two on the same middle.
    for (const field of canvasElement.querySelectorAll('textarea')) {
      const button = field.closest('.composer')?.querySelector('button')
      const middle = (box: DOMRect) => box.top + box.height / 2
      await expect(
        Math.abs(middle(field.getBoundingClientRect()) - middle(button!.getBoundingClientRect())),
      ).toBeLessThan(1)
    }
  },
}

/**
 * Grown by what was typed, and then past the height it is allowed — where it
 * stops growing and scrolls instead.
 */
export const Grown: Story = {
  render: holding(MULTILINE, `${LONG}\n\n${LONG}`),
  play: async ({ canvasElement }) => {
    for (const field of canvasElement.querySelectorAll('textarea')) {
      const within = field.closest('.composer')?.getBoundingClientRect().bottom ?? 0
      await expect(field.getBoundingClientRect().bottom).toBeLessThanOrEqual(within)
    }
  },
}

/** A word with nowhere to break, a script that is not Latin, and one that
 *  runs the other way. */
export const AwkwardText: Story = { render: holding(LINK, DEVANAGARI, ARABIC) }
