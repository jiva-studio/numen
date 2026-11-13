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

const names = (id: string): PlexNames => NAMES[id] ?? { above: [], below: [] }

const meta = {
  title: 'Plex/Names',
  component: Plex,
  parameters: { layout: 'fullscreen' },
  args: { showNames: true, onActivate: fn(), onChooseName: fn() },
  render: (args: Knobs) => ({
    components: { Plex },
    setup: () => ({ args, graph, names, headings: (id: string) => HEADINGS[id] ?? [] }),
    template: `
      <div style="height:100vh">
        <Plex
          :graph="graph"
          :duration="0"
          :names="names"
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
type Story = StoryObj<typeof meta>

const getName = (canvas: ReturnType<typeof within>, title: string) =>
  canvas.getByRole('button', { name: title })

const getCenterY = (element: Element) => {
  const box = element.getBoundingClientRect()
  return box.top + box.height / 2
}

const getFontSize = (element: Element) => parseFloat(getComputedStyle(element).fontSize)

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

/** A name is smaller than a title, and the size of the focus changes the pill of the focus only. */
export const NamesStaySmall: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const title = canvas.getByLabelText('Raised beds, child').querySelector('.plex__title-text')!

    await expect(getFontSize(getName(canvas, 'Compost'))).toBeLessThan(getFontSize(title))
    await expect(getFontSize(getName(canvas, 'Compost'))).toBe(getFontSize(getName(canvas, 'Spade')))
    const focusBox = canvas.getByLabelText(/, focus$/).querySelector('.plex__box')!
    const childBox = canvas.getByLabelText('Raised beds, child').querySelector('.plex__box')!
    await expect(focusBox.getBoundingClientRect().height).toBeGreaterThan(
      childBox.getBoundingClientRect().height,
    )
  },
}

/** Choosing a name hands the id of its note to the caller, and the node of the note it names is not chosen. */
export const ChoosingAName: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(getName(canvas, 'Mulch'))

    await expect(args.onChooseName).toHaveBeenCalledExactlyOnceWith('name:Mulch')
    await expect(args.onActivate).not.toHaveBeenCalled()
  },
}

/** The pointer over a note or over one of its names brings the names of that note forward, and no other. */
export const HoverBringsNamesForward: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const getOpacity = (title: string) => parseFloat(getComputedStyle(getName(canvas, title)).opacity)
    const beds = canvas.getByLabelText('Raised beds, child')
    const atRest = getOpacity('Compost')

    await userEvent.hover(beds)
    await waitFor(async () => await expect(getOpacity('Compost')).toBeGreaterThan(atRest))
    await expect(getOpacity('Compost')).toBe(1)
    await expect(getOpacity('Spade')).toBe(atRest)

    await userEvent.unhover(beds)
    await waitFor(async () => await expect(getOpacity('Compost')).toBe(atRest))

    await userEvent.hover(getName(canvas, 'Mulch'))
    await waitFor(async () => await expect(getOpacity('Compost')).toBe(1))
  },
}

/** The hover of a pill opens its heading list; the hover of a name opens none. */
export const OnlyAPillOpensHeadings: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.hover(getName(canvas, 'Compost'))
    await new Promise((isDone) => setTimeout(isDone, 400))
    await expect(canvasElement.querySelector('.plex__inside')).toBeNull()

    await userEvent.unhover(getName(canvas, 'Compost'))
    await userEvent.hover(canvas.getByLabelText('Raised beds, child'))
    await waitFor(async () => await expect(canvasElement.querySelector('.plex__inside')).not.toBeNull())
  },
}

/** With the names off no name is drawn. */
export const NamesOff: Story = {
  args: { showNames: false },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(canvas.queryByRole('button', { name: 'Compost' })).toBeNull()
    await expect(canvas.queryByRole('button', { name: 'Orchard' })).toBeNull()
    await expect(canvas.getByLabelText('Raised beds, child')).toBeInTheDocument()
  },
}
