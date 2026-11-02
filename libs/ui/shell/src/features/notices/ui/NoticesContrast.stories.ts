/**
 * The contrast of the text of every card of the corner, whatever its severity, in the light set and
 * in the dark one. Also the test corpus: a browser resolves each colour as drawn, and the story fails
 * when a card's text falls below 4.5:1 on the card.
 *
 * Each reading is printed as a `CONTRAST|kind|name|ratio` line.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import Notices from './Notices.vue'
import type { Notice } from '../lib/notice'
import { DARK, expectDark } from '@/shared/fixtures/theme'
import { expectAllText } from '../../../../.storybook/contrast'

const meta: Meta = {
  title: 'Application/Notices contrast',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const NOTICES: readonly Notice[] = [
  {
    id: 'work',
    text: 'Preparing search by meaning',
    about: 'notes/entropy.md',
    done: 3,
    total: 8,
    isWorking: true,
  },
  { id: 'info', text: 'Renamed', about: 'notes/entropy.md', stay: 'kept' },
  {
    id: 'warning',
    text: 'A note of that name is filed there already',
    about: 'notes/entropy.md',
    severity: 'warning',
    stay: 'kept',
  },
  {
    id: 'error',
    text: 'The model could not be fetched',
    about: 'intfloat/multilingual-e5-small',
    severity: 'error',
    stay: 'kept',
  },
]

const corner = () => ({
  components: { Notices },
  setup: () => ({ notices: NOTICES, clock: () => Date.now(), hidden: () => false }),
  template: `
    <div class="numen" style="height:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
      <Notices :notices="notices" name="Background work" dismiss="Dismiss" more="more"
        :wait="0" :room="4" :clock="clock" :hidden="hidden" />
    </div>`,
})

async function playCards(): Promise<void> {
  await waitFor(() => expect(document.body.querySelectorAll('.notice').length).toBe(4))
  for (const card of document.body.querySelectorAll('.notice'))
    await expectAllText(card, `notice ${card.getAttribute('data-severity')}`)
}

export const Light: Story = { render: corner, play: playCards }

export const Dark: Story = {
  render: corner,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playCards()
  },
}
