/**
 * The headings of a note as the slot under a node writes them. Also the test corpus: each story is run in a
 * browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import InlineHeadings from './InlineHeadings.vue'

const meta = {
  title: 'Plex/Inline headings',
  component: InlineHeadings,
  args: {
    headings: [
      { text: 'What it measures', level: 2, line: 4 },
      { text: "Boltzmann's formula", level: 3, line: 19 },
      { text: 'The arrow of time', level: 2, line: 57 },
    ],
    onChoose: fn(),
  },
  render: (args) => ({
    components: { InlineHeadings },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;justify-content:center;inline-size:12rem;padding:1rem;background:var(--numen-surface)">
        <InlineHeadings v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof InlineHeadings>

export default meta
type Story = StoryObj<typeof meta>

/** Three headings under a note, each after a hash; choosing one reports its line. */
export const ChoosesAHeading: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'headings', width: 320, height: 200, drawn: 1_000 }],
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: '# The arrow of time' }))
    await expect(args.onChoose).toHaveBeenCalledWith(57)
    await expect(args.onChoose).toHaveBeenCalledTimes(1)
  },
}

/** A note with no headings draws nothing. */
export const NoHeadings: Story = {
  args: { headings: [] },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).queryByRole('button')).toBeNull()
  },
}
