/**
 * The actions of a card header at the widths of the card: words, icons only, and icons with the
 * overflow menu.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { KeyRound, Plus, QrCode, ScanLine } from '@lucide/vue'
import { expect, within } from 'storybook/test'
import CardActions from './CardActions.vue'
import IslandCard from './IslandCard.vue'
import type { CardActionItem } from './cardActionItem'

const ITEMS: readonly CardActionItem[] = [
  { id: 'words', label: 'Restore from words', icon: KeyRound },
  { id: 'sheets', label: 'Restore from sheets', icon: ScanLine },
  { id: 'join', label: 'Join', icon: QrCode },
  { id: 'create', label: 'Create', icon: Plus, isPrimary: true },
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
    await expect(within(canvasElement).getByText('Restore from words')).toBeVisible()
  },
}

/** Narrow: the words give way to icons, every button keeps its name. */
export const Narrow: Story = {
  parameters: { cardWidth: '24rem' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.queryByText('Restore from words')).not.toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore from words' })).toBeVisible()
    await expect(canvas.getByRole('heading', { name: 'Identity' })).toBeVisible()
  },
}

/** Narrower still: the secondary actions move into the overflow menu. */
export const Narrowest: Story = {
  parameters: { cardWidth: '14rem' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.queryByRole('button', { name: 'Restore from words' })).toBeNull()
    await expect(canvas.getByRole('button', { name: 'More actions' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Create' })).toBeVisible()
  },
}
