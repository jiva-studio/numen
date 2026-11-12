/**
 * What the composer looks like. What it does is asserted in
 * `MessageComposer.test.ts`.
 *
 * `working` and `disabled` are controls, not stories.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, within } from 'storybook/test'
import { ref } from 'vue'
import MessageComposer from './MessageComposer.vue'
import { STORY_BOXES } from '../../fixtures/frame'
import { ARABIC, DEVANAGARI, LINK, LONG, MULTILINE } from '../../fixtures/prose'

const meta = {
  title: 'Chat/MessageComposer',
  component: MessageComposer,
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
          'its way the disc stops it.',
      },
    },
  },
  argTypes: {
    placeholder: { control: 'text' },
    isAnswerPending: { control: 'boolean' },
    isDisabled: { control: 'boolean' },
  },
  args: {
    placeholder: 'Write a message',
    isAnswerPending: false,
    isDisabled: false,
    onSubmit: fn(),
    onStop: fn(),
  },
} satisfies Meta<typeof MessageComposer>

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

const renderComposers =
  (...starts: string[]): Render =>
  (args) => ({
    components: { MessageComposer },
    setup: () => ({ args, texts: starts.map((start) => ref(start)), STORY_BOXES }),
    template: `
    <div :class="STORY_BOXES.composers">
      <MessageComposer
        v-for="(text, index) in texts"
        :key="index"
        v-bind="args"
        v-model="text.value"
      />
    </div>
  `,
  })

/** Empty, and with a line in it. Turn `working` on for the disc that stops. */
export const Playground: Story = {
  render: renderComposers('', 'What does a plex draw?'),
  play: async ({ canvasElement }) => {
    // The field is called by the words standing in it.
    await expect(
      within(canvasElement).getAllByRole('textbox', { name: 'Write a message' }),
    ).toHaveLength(2)
    for (const field of canvasElement.querySelectorAll('textarea')) {
      const composer = field.closest('.composer')
      await expect(composer).toBeInTheDocument()
      const button = composer?.querySelector('button')
      await expect(button).toBeInTheDocument()
    }
  },
}

/**
 * Text of several lines and text longer than the field allows: the field holds all of it.
 */
export const Grown: Story = {
  render: renderComposers(MULTILINE, `${LONG}\n\n${LONG}`),
  play: async ({ canvasElement }) => {
    const fields = [...canvasElement.querySelectorAll('textarea')]
    await expect(fields.map((field) => field.value)).toEqual([MULTILINE, `${LONG}\n\n${LONG}`])
  },
}

/** A word with nowhere to break, a script that is not Latin, and one that
 *  runs the other way. */
export const AwkwardText: Story = { render: renderComposers(LINK, DEVANAGARI, ARABIC) }

