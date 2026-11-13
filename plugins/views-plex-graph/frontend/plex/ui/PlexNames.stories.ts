/**
 * The names of the notes related to a related note: small text over and under its pill.
 * Every graph here is made up.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import Plex from './Plex.vue'
import type { PlexGraph, PlexHeading, PlexNames } from '../../types'

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Kitchen garden', relation: 'focus' },
    { id: 'beds', title: 'Raised beds', relation: 'child' },
    { id: 'tools', title: 'Hand tools', relation: 'child' },
    { id: 'garden', title: 'Garden', relation: 'parent' },
  ],
  edges: [
    { from: 'garden', to: 'focus' },
    { from: 'focus', to: 'beds' },
    { from: 'focus', to: 'tools' },
  ],
}

const name = (title: string) => ({ id: `name:${title}`, title })

const NAMES: Record<string, PlexNames> = {
  beds: {
    above: [name('Allotment')],
    below: [name('Compost'), name('Mulch'), name('Seedlings'), name('Watering')],
  },
  tools: { above: [], below: [name('Spade'), name('Fork')] },
  garden: { above: [], below: [name('Orchard')] },
}

const HEADINGS: Record<string, readonly PlexHeading[]> = {
  beds: [{ id: '1', text: 'Soil mix', level: 1 }],
}

interface Knobs {
  showNames: boolean
  onActivate: (id: string) => void
  onChooseName: (id: string) => void
}

const getNamesOf = (id: string): PlexNames => NAMES[id] ?? { above: [], below: [] }

const meta = {
  title: 'Plex/Names',
  component: Plex,
  parameters: { layout: 'fullscreen' },
  args: { showNames: true, onActivate: fn(), onChooseName: fn() },
  render: (args: Knobs) => ({
    components: { Plex },
    setup: () => ({ args, graph, getNamesOf, headings: (id: string) => HEADINGS[id] ?? [] }),
    template: `
      <div style="height:100vh">
        <Plex
          :graph="graph"
          :duration="0"
          :names="getNamesOf"
          :show-names="args.showNames"
          :headings="headings"
          :options="{ focusScale: 2 }"
          @activate="args.onActivate"
          @choose-name="args.onChooseName"
        />
      </div>`,
  }),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<Knobs>

const getName = (canvas: ReturnType<typeof within>, title: string) =>
  canvas.getByRole('button', { name: title })

const getCenterY = (element: Element) => {
  const box = element.getBoundingClientRect()
  return box.top + box.height / 2
}

/** Each related note shows its names as buttons, the names of its parents over it and the rest under it. */
export const WithNames: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const beds = canvas.getByLabelText('Raised beds, child')

    await expect(getCenterY(getName(canvas, 'Allotment'))).toBeLessThan(getCenterY(beds))
    for (const title of ['Compost', 'Mulch', 'Seedlings', 'Watering']) {
      await expect(getCenterY(getName(canvas, title))).toBeGreaterThan(getCenterY(beds))
    }
    await expect(getCenterY(getName(canvas, 'Spade'))).toBeGreaterThan(
      getCenterY(canvas.getByLabelText('Hand tools, child')),
    )
    await expect(canvas.queryByRole('button', { name: 'Kitchen garden' })).toBeNull()
  },
}

/** Choosing a name hands the id of its note to the caller, and the node of the note it names is not chosen. */
export const ChoosingAName: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(getName(canvas, 'Mulch'))

    await expect(args.onChooseName).toHaveBeenCalledTimes(1)
    await expect(args.onChooseName).toHaveBeenCalledWith('name:Mulch')
    await expect(args.onActivate).not.toHaveBeenCalled()
  },
}

/** The hover of a pill opens the heading list of its note. */
export const APillOpensHeadings: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.hover(canvas.getByLabelText('Raised beds, child'))

    await waitFor(
      async () => await expect(canvasElement.querySelector('.plex__inside')).not.toBeNull(),
    )
  },
}
