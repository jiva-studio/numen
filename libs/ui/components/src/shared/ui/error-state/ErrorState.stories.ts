/**
 * The state a pane shows when its content cannot be opened.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import ErrorState from './ErrorState.vue'

const meta = {
  title: 'ErrorState',
  component: ErrorState,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A tab that cannot open its content says so with an icon and a message, in the ' +
          'middle of the room it was given, and the words carry the meaning without the colour.',
      },
    },
  },
  render: (args) => ({
    components: { ErrorState },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="block-size:320px;background:var(--numen-surface)">
        <ErrorState v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof ErrorState>

export default meta
type Story = StoryObj<typeof meta>

/** A message alone. */
export const MissingNote: Story = {
  args: { message: 'There is no note at Notes/Reef.md' },
  play: async ({ canvasElement }) => {
    const alert = within(canvasElement).getByRole('alert')

    await expect(alert).toHaveTextContent('There is no note at Notes/Reef.md')
  },
}

/** A heading above the message. */
export const WithHeading: Story = {
  args: { heading: 'Could not restore this tab', message: 'The saved state is unreadable.' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('heading')).toHaveTextContent(
      'Could not restore this tab',
    )
  },
}
