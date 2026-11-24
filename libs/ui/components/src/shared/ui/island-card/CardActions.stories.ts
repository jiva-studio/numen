/**
 * The actions of a card header at the widths of the card: words, icons only, and icons with the
 * overflow menu.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { KeyRound, Plus, QrCode } from '@lucide/vue'
import { expect, within } from 'storybook/test'
import CardActions from './CardActions.vue'
import IslandCard from './IslandCard.vue'
import type { CardActionItem } from './cardActionItem'

const ITEMS: readonly CardActionItem[] = [
  {
    id: 'restore',
    label: 'Restore',
    icon: KeyRound,
    choices: [
      { id: 'words', label: 'From 12 words' },
      { id: 'typed', label: 'Type sheets' },
    ],
  },
  { id: 'join', label: 'Join', icon: QrCode },
  { id: 'create', label: 'Create', icon: Plus, isPrimary: true, isLabelKept: true },
]

const meta: Meta = {
  title: 'Controls/IslandCard/Actions',
  component: CardActions,
  render: (_args, { parameters }) => ({
    components: { CardActions, IslandCard },
    setup: () => ({ items: ITEMS, width: parameters.cardWidth as string }),
    template: `
      <div class="numen" style="padding:1rem;background:var(--numen-surface)">
        <div :style="{ inlineSize: width }">
          <IslandCard title="Identity">
            <template #aside><CardActions :items="items" more-label="More actions" /></template>
            <p style="margin:0">Rows</p>
          </IslandCard>
        </div>
      </div>
    `,
  }),
}
export default meta
type Story = StoryObj

/** Wide: icons with their words. */
export const Wide: Story = {
  parameters: { cardWidth: '60rem' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('Join')).toBeVisible()
  },
}

/** Narrow: the words give way to icons except on Create, every button keeps its name. */
export const Narrow: Story = {
  parameters: { cardWidth: '24rem' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.queryByText('Join')).not.toBeVisible()
    await expect(canvas.getByText('Create')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Join' })).toBeVisible()
    await expect(canvas.getByRole('heading', { name: 'Identity' })).toBeVisible()
  },
}

/** Narrower still: the secondary actions move into the overflow menu. */
export const Narrowest: Story = {
  parameters: { cardWidth: '14rem' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.queryByRole('button', { name: 'Join' })).toBeNull()
    await expect(canvas.getByRole('button', { name: 'Restore' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'More actions' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Create' })).toBeVisible()
  },
}
