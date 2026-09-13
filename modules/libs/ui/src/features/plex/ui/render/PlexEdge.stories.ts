/**
 * One edge and the title along it, at rest and lifted over the boxes.
 *
 * Inside a plex an edge is a hairline under everything else, which is no way to
 * judge whether the lifted pair really stands out from the resting one. Both
 * are drawn here side by side, in a window with the origin in the middle.
 *
 * A title is set along a path the drawing has to name, so every story lays that
 * path down itself, as the plex does.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import type { EdgeLine } from './lines'
import { arrowTransformOf, pathOf, readingPathOf } from '../../lib/arrange'
import type { PlacedEdge } from '../../lib/edge'
import { lightness } from '@/shared/fixtures/colour'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Knobs {
  /** The words set along the line. Nothing draws no title at all. */
  words: string
  /** How far along the line the middle of those words stands. */
  wordsAt: number
  /** Below one, the edge is on its way in or out. */
  opacity: number
  /** Whether the line ends in a head. */
  arrow: boolean
}

const edgeFrom = (args: Knobs): PlacedEdge => ({
  from: 'one',
  to: 'two',
  label: args.words,
  ...(args.arrow ? { arrow: 'to' as const } : {}),
  fromPoint: { x: -200, y: -40 },
  toPoint: { x: 200, y: 40 },
  control1: { x: -60, y: -40 },
  control2: { x: 60, y: 40 },
  opacity: args.opacity,
  heading: 'along',
  words: args.words || undefined,
  wordsAt: args.wordsAt,
  ...(args.arrow ? { arrowhead: { at: { x: 200, y: 40 }, angle: 12 } } : {}),
})

/** The edge with what the drawing asks of it, as the plex works it out. */
const lineFrom = (edge: PlacedEdge): EdgeLine => ({
  edge,
  key: `${edge.from}->${edge.to}`,
  pair: `${edge.from}|${edge.to}`,
  d: pathOf(edge),
  arrow: edge.arrowhead ? arrowTransformOf(edge.arrowhead) : null,
  titlePath: edge.words ? 'edge-title' : null,
  titleLine: readingPathOf(edge),
  titleAt: `${100 * edge.wordsAt}%`,
})

/**
 * The pair drawn twice over one window: at rest, and lifted. Each is wrapped in
 * a group of the story's own, because neither drawing carries a name.
 */
const both = (args: Knobs) => ({
  components: { PlexEdgeLine, PlexEdgeTitle },
  setup: () => ({ line: lineFrom(edgeFrom(args)) }),
  template: `
    <div data-ground style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
      <svg class="numen" width="480" height="180" viewBox="-240 -90 480 180" style="font-family:var(--numen-font-sans)">
        <defs><path :id="line.titlePath" :d="line.titleLine" /></defs>
        <g data-edge="resting">
          <PlexEdgeLine :line="line" />
          <PlexEdgeTitle :line="line" />
        </g>
        <g data-edge="lifted" transform="translate(0 48)">
          <PlexEdgeLine :line="line" lifted />
          <PlexEdgeTitle :line="line" lifted />
        </g>
      </svg>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Plex/Edge',
  component: PlexEdgeLine,
  parameters: {
    layout: 'fullscreen',
    // The line is assembled from the knobs below.
    controls: { exclude: ['line', 'lifted'] },
    docs: {
      description: {
        component:
          'One edge: the curve, the head it ends in, and the title set along ' +
          'it over a halo. Lifted, it is drawn over the boxes, in the ' +
          'colours and the weight that stand out against them.',
      },
    },
  },
  argTypes: {
    words: { control: 'text' },
    wordsAt: { control: { type: 'range', min: 0.1, max: 0.9, step: 0.05 } },
    opacity: { control: { type: 'range', min: 0, max: 1, step: 0.05 } },
    arrow: { control: 'boolean' },
  },
  args: { words: 'refers to', wordsAt: 0.5, opacity: 1, arrow: true },
  render: both,
}

export default meta
type Story = StoryObj<Knobs>

/** The pair at rest above, and lifted below. */
export const Playground: Story = {}

/** A line carrying no words, which draws no title and no path to set one on. */
export const NoTitle: Story = {
  args: { words: '' },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelectorAll('text')).toHaveLength(0)
    expect(canvasElement.querySelectorAll('[data-edge] path')).toHaveLength(4)
  },
}

/** On its way in or out, where the whole edge fades and the title with it. */
export const PartWayThere: Story = {
  args: { opacity: 0.3 },
  play: async ({ canvasElement }) => {
    const drawn = canvasElement.querySelectorAll('[data-edge="resting"] path, [data-edge="resting"] text')
    expect(drawn).toHaveLength(4)
    for (const one of drawn) expect(one.getAttribute('opacity')).toBe('0.3')
  },
}

/** The parts of one edge, in the order they are drawn under a group. */
const partsOf = (canvas: HTMLElement, edge: string) => {
  const group = canvas.querySelector(`[data-edge="${edge}"]`)
  if (!group) throw new Error(`no ${edge} edge`)
  const [curve, head] = group.querySelectorAll('path')
  const [halo, letters] = group.querySelectorAll('text')
  if (!curve || !head || !halo || !letters) throw new Error(`the ${edge} edge is short a part`)
  return { curve, head, halo, letters }
}

/**
 * On the dark set of tokens, at rest and lifted over one another.
 *
 * Lifted means drawn over the boxes, so both the line and its letters have to
 * come further from the surface than the resting pair does, and the halo has
 * to stay the surface itself or it stops hiding what runs under the words.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const ground = getComputedStyle(canvasElement.querySelector('[data-ground]')!).backgroundColor
    const surface = lightness(ground)
    const getDistanceFrom = (colour: string) => Math.abs(lightness(colour, ground) - surface)

    const resting = partsOf(canvasElement, 'resting')
    const lifted = partsOf(canvasElement, 'lifted')

    // A line lying quietly on the surface still stands above it on the dark
    // set, where a line is drawn lighter than what it is drawn on.
    await expect(lightness(getComputedStyle(resting.curve).stroke, ground)).toBeGreaterThan(
      surface + 2,
    )

    // The line and its head are one drawing, and lifting moves both further
    // from the ground they are read on. The curve is stroked and the head is
    // filled, so each is read off the property it is painted with.
    await expect(getDistanceFrom(getComputedStyle(lifted.curve).stroke)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(resting.curve).stroke) + 2,
    )
    await expect(getDistanceFrom(getComputedStyle(lifted.head).fill)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(resting.head).fill) + 2,
    )

    // The letters likewise, and the halo under them stays the surface, at a
    // weight that covers the line running through the words.
    await expect(getDistanceFrom(getComputedStyle(lifted.letters).fill)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(resting.letters).fill) + 2,
    )
    for (const halo of [resting.halo, lifted.halo]) {
      const drawn = getComputedStyle(halo)
      await expect(getDistanceFrom(drawn.stroke)).toBeLessThan(2)
      await expect(Number.parseFloat(drawn.strokeWidth)).toBeGreaterThan(0)
    }
    await expect(Number.parseFloat(getComputedStyle(lifted.halo).strokeWidth)).toBeGreaterThan(
      Number.parseFloat(getComputedStyle(resting.halo).strokeWidth),
    )
  },
}
