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

const escapeRegExp = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const getName = (canvas: ReturnType<typeof within>, title: string) =>
  canvas.getByRole('button', { name: new RegExp(`^${escapeRegExp(title)}, name under `) })

// The y comes from the translate the element is drawn at.
const getY = (element: Element) => {
  const match = /translate\(\S+ (\S+)\)/.exec(element.getAttribute('transform') ?? '')
  if (!match) throw new Error('the element is not placed yet')
  return Number(match[1])
}

/** Each related note shows its names as buttons, the names of its parents over it and the rest under it. */
export const WithNames: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const beds = canvas.getByLabelText('Raised beds, child')

    await waitFor(async () => await expect(getY(beds)).toBeGreaterThan(0))
    await expect(getY(getName(canvas, 'Allotment'))).toBeLessThan(getY(beds))
    for (const title of ['Compost', 'Mulch', 'Seedlings', 'Watering']) {
      await expect(getY(getName(canvas, title))).toBeGreaterThan(getY(beds))
    }
    await expect(getY(getName(canvas, 'Spade'))).toBeGreaterThan(
      getY(canvas.getByLabelText('Hand tools, child')),
    )
    await expect(canvas.queryByRole('button', { name: 'Kitchen garden' })).toBeNull()
  },
}

/** Choosing a name hands the id of its note to the caller. */
export const ChoosingAName: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(getName(canvas, 'Mulch'))

    await expect(args.onChooseName).toHaveBeenCalledTimes(1)
    await expect(args.onChooseName).toHaveBeenCalledWith('name:Mulch')
  },
}
