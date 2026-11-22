/**
 * The island card: a titled header over a body of rows.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import IslandCard from './IslandCard.vue'

interface Knobs {
  title: string
  aside: string
}

const meta: Meta<Knobs> = {
  title: 'Controls/IslandCard',
  component: IslandCard,
  argTypes: { title: { control: 'text' }, aside: { control: 'text' } },
  args: { title: 'Below a node', aside: '' },
  render: (args) => ({
    components: { IslandCard },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="max-inline-size:28rem;padding:1rem;background:var(--numen-surface)">
        <IslandCard :title="args.title">
          <template v-if="args.aside" #aside>{{ args.aside }}</template>
          <ul style="margin:0;padding:0;list-style:none">
            <li>Tags</li>
            <li>Backlinks</li>
          </ul>
        </IslandCard>
      </div>
    `,
  }),
}
export default meta
type Story = StoryObj<Knobs>

/** A title over two rows. */
export const Titled: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.getByRole('heading')).toHaveTextContent('Below a node')
    await expect(canvas.getByText('Tags')).toBeVisible()
  },
}

/** A description at the end of the header. */
export const WithAside: Story = {
  args: { aside: '2 views' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('2 views')).toBeVisible()
  },
}
