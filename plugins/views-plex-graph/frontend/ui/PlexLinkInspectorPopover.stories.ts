/**
 * The popover that edits the connection between two notes: one row per record, each with its
 * direction, its description and a button that removes it.
 *
 * The widths of the rows and the pixels the controls land on are the browser's answer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { Connection } from '@numen/plugin-sdk'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'
import { WORDS as words } from '../words'

const createConnection = (more: Partial<Connection>): Connection => ({
  first: 'Harbour.md',
  second: 'Lighthouse.md',
  role: 'jump',
  type: '',
  firstSide: null,
  secondSide: null,
  ...more,
})

const createRequest = (connection: Connection) => ({
  nodeA: { id: 'a', title: 'Harbour', path: 'Harbour.md' },
  nodeB: { id: 'b', title: 'Lighthouse', path: 'Lighthouse.md' },
  connection,
  at: { x: 300, y: 260 },
})

const meta = {
  title: 'Plex/Link Inspector Popover',
  component: PlexLinkInspectorPopover,
  parameters: { layout: 'fullscreen' },
  args: {
    request: createRequest(
      createConnection({ firstSide: { label: 'guides ships in', marker: null, reason: '' } }),
    ),
  },
} satisfies Meta<typeof PlexLinkInspectorPopover>

export default meta
type Story = StoryObj<typeof meta>

const getDirections = (canvasElement: HTMLElement) =>
  [...canvasElement.querySelectorAll('.plex-link-row__dir-btn')].map((button) =>
    button.getAttribute('data-direction'),
  )

/** One record without a marker: one row, no kind of link and no checkbox. */
export const NoDirection: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(getDirections(canvasElement)).toStrictEqual(['none'])
    await expect(canvas.getByRole('textbox')).toHaveValue('guides ships in')
    await expect(canvas.getByRole('button', { name: words.removeLink })).toBeVisible()
    await expect(canvas.queryByRole('combobox')).toBeNull()
    await expect(canvas.queryByRole('checkbox')).toBeNull()
  },
}

/** The record of the first note, pointing at the second. */
export const Forward: Story = {
  args: {
    request: createRequest(
      createConnection({ firstSide: { label: 'guides ships in', marker: 'arrow', reason: '' } }),
    ),
  },
  play: async ({ canvasElement }) => {
    await expect(getDirections(canvasElement)).toStrictEqual(['forward'])
  },
}

/** The record of the second note, pointing at the first. */
export const Backward: Story = {
  args: {
    request: createRequest(
      createConnection({ secondSide: { label: 'shelters', marker: 'arrow', reason: '' } }),
    ),
  },
  play: async ({ canvasElement }) => {
    await expect(getDirections(canvasElement)).toStrictEqual(['backward'])
  },
}

/** A record each way: two rows with arrows, each with its own description. */
export const BothDirections: Story = {
  args: {
    request: createRequest(
      createConnection({
        firstSide: { label: 'guides ships in', marker: 'arrow', reason: '' },
        secondSide: { label: 'shelters', marker: 'arrow', reason: '' },
      }),
    ),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(getDirections(canvasElement)).toStrictEqual(['forward', 'backward'])
    for (const button of canvasElement.querySelectorAll('.plex-link-row__dir-btn')) {
      await expect(button).toBeEnabled()
    }
    await expect(canvas.queryByText(words.addReverseDirection, { exact: false })).toBeNull()
  },
}

/** The direction button turns through none, forward, backward and none again. */
export const CycleDirection: Story = {
  play: async ({ canvasElement }) => {
    const button = canvasElement.querySelector<HTMLButtonElement>('.plex-link-row__dir-btn')!
    const directions = [button.getAttribute('data-direction')]
    for (let step = 0; step < 3; step += 1) {
      await userEvent.click(button)
      directions.push(button.getAttribute('data-direction'))
    }

    await expect(directions).toStrictEqual(['none', 'forward', 'backward', 'none'])
  },
}

/** A record of each note without an arrow: each button gives its own record an arrow and takes it away. */
export const TurnArrowsOfTwoRecords: Story = {
  args: {
    request: createRequest(
      createConnection({
        firstSide: { label: 'guides ships in', marker: null, reason: '' },
        secondSide: { label: 'shelters', marker: null, reason: '' },
      }),
    ),
  },
  play: async ({ canvasElement }) => {
    const [first, second] = [
      ...canvasElement.querySelectorAll<HTMLButtonElement>('.plex-link-row__dir-btn'),
    ]
    await expect(getDirections(canvasElement)).toStrictEqual(['none', 'none'])

    await userEvent.click(second!)
    await expect(getDirections(canvasElement)).toStrictEqual(['none', 'backward'])
    await userEvent.click(first!)
    await expect(getDirections(canvasElement)).toStrictEqual(['forward', 'backward'])
    await expect(first).toHaveAccessibleName('Arrow to Lighthouse. Click for: No arrow')
    await userEvent.click(second!)
    await expect(getDirections(canvasElement)).toStrictEqual(['forward', 'none'])
  },
}

/** Adding the reverse direction gives a second row with an empty description. */
export const AddReverseDirection: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(
      canvas.getByRole('button', { name: new RegExp(words.addReverseDirection) }),
    )

    await expect(getDirections(canvasElement)).toStrictEqual(['forward', 'backward'])
    await expect(
      canvas.getAllByRole('textbox').map((field) => (field as HTMLInputElement).value),
    ).toStrictEqual(['guides ships in', ''])
  },
}

/** With the last record removed, the popover says the link goes and offers to add one. */
export const NoRecordLeft: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(canvas.getByRole('button', { name: words.removeLink }))

    await expect(canvas.getByText(words.linkWillBeRemoved)).toBeVisible()
    await expect(canvas.getByRole('button', { name: new RegExp(words.addLink) })).toBeVisible()
  },
}
