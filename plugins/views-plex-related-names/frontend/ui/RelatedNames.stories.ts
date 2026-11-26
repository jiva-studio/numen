/**
 * The names of the notes related to a note of the plex, as the slot of a node draws them. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import type { RelatedName } from '../types'
import RelatedNames from './RelatedNames.vue'

const toName = (title: string): RelatedName => ({
  path: `notes/${title}.md`,
  title,
  relation: 'child',
  label: `${title}, child of Pear rows`,
})

const meta = {
  title: 'Plex/Related names',
  component: RelatedNames,
  args: {
    names: ['Rootstock list', 'Grafting', 'Pollination'].map(toName),
    onChoose: fn(),
  },
  render: (args) => ({
    components: { RelatedNames },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;justify-content:center;inline-size:12rem;padding:1rem;background:var(--numen-surface)">
        <RelatedNames v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof RelatedNames>

export default meta
type Story = StoryObj<typeof meta>

/** Three names under a note; choosing one reports the path of its note. */
export const ChoosesAName: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'names', width: 400, height: 240, drawn: 1_000 }],
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(
      await canvas.findByRole('button', { name: 'Grafting, child of Pear rows' }),
    )
    await expect(args.onChoose).toHaveBeenCalledWith('notes/Grafting.md')
    await expect(args.onChoose).toHaveBeenCalledTimes(1)
  },
}

/** Titles too long for the width wrap onto more lines, each name whole. */
export const LongTitles: Story = {
  args: {
    names: [
      'A very long note about grafting pear scions onto quince rootstock in early spring',
      'Pollination',
    ].map(toName),
  },
}

/** A note with no names draws nothing. */
export const NoNames: Story = {
  args: { names: [] },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).queryByRole('button')).toBeNull()
  },
}
