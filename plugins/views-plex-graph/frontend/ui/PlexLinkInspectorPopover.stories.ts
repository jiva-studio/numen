/**
 * The popover that edits the connection between two notes: the kind of link, and for each note
 * whether it writes a record, its label and its arrow.
 *
 * The widths of the rows and the pixels the controls land on are the browser's answer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'

const meta = {
  title: 'Plex/Link Inspector Popover',
  component: PlexLinkInspectorPopover,
  parameters: { layout: 'fullscreen' },
  args: {
    request: {
      nodeA: { id: 'a', title: 'Ontology', path: 'Ontology.md' },
      nodeB: { id: 'b', title: 'Being', path: 'Being.md' },
      connection: {
        first: 'Ontology.md',
        second: 'Being.md',
        role: 'child',
        type: '',
        firstSide: { label: 'studies', marker: 'arrow', reason: '' },
        secondSide: null,
      },
      roles: ['parent', 'child', 'jump'],
      at: { x: 300, y: 260 },
    },
  },
} satisfies Meta<typeof PlexLinkInspectorPopover>

export default meta
type Story = StoryObj<typeof meta>

/** One note writes a record with an arrow; the other writes none, so its fields are off. */
export const OneSide: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const fields = canvas.getAllByRole('textbox')

    await expect(fields[0]).toBeEnabled()
    await expect(fields[1]).toBeDisabled()
    await expect(canvas.getAllByRole('button', { name: 'Arrow' })[0]).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  },
}

/** Writing a record for the second note turns its fields on, and its arrow starts off. */
export const AddTheOtherSide: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const [, second] = canvas.getAllByRole('checkbox')

    await userEvent.click(second!)

    await expect(canvas.getAllByRole('textbox')[1]).toBeEnabled()
    await expect(canvas.getAllByRole('button', { name: 'Arrow' })[1]).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  },
}
