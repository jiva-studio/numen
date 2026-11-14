/**
 * One edge and the titles along it, at rest and lifted over the boxes.
 *
 * Inside a plex an edge is a hairline below all other elements, which makes it
 * hard to judge whether the lifted edge differs from the resting one. Both
 * are drawn side by side here, in a viewport with the origin in the middle.
 *
 * A title is set along a path that the drawing must define, so every story
 * creates that path itself, as the plex does.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import { getLines } from './lines'
import { createTitle, getLabelPlaces, getRise, getShift } from '../../lib/arrange'
import { getArrow, type EdgeCurve, type PlacedEdge } from '../../lib/edge'
import type { EdgeArrow } from '../../../types'

/**
 * Which records the edge draws: one without a marker, one pointing at the `to`
 * end, one pointing at the `from` end, or one of each.
 */
type Direction = 'none' | 'forward' | 'backward' | 'both'

interface Knobs {
  words: string
  reverseWords: string
  direction: Direction
  opacity: number
}

const CURVE: EdgeCurve = {
  fromPoint: { x: -200, y: -40 },
  toPoint: { x: 200, y: 40 },
  control1: { x: -60, y: -40 },
  control2: { x: 60, y: 40 },
}

const getRecords = (args: Knobs): { words: string; end?: EdgeArrow }[] => {
  switch (args.direction) {
    case 'none':
      return [{ words: args.words }]
    case 'forward':
      return [{ words: args.words, end: 'to' }]
    case 'backward':
      return [{ words: args.words, end: 'from' }]
    case 'both':
      return [
        { words: args.words, end: 'to' },
        { words: args.reverseWords, end: 'from' },
      ]
  }
}

/** The ink of the label font, as the arrangement measures it. */
const LABEL_TYPE = { ascent: 9, descent: 3, xHeight: 6 }

const createEdge = (args: Knobs): PlacedEdge => {
  const records = getRecords(args)
  const ends = records.flatMap((record) => (record.end ? [record.end] : []))
  const labelled = records.filter((record) => record.words)
  const labels = labelled.map((record) => ({
    text: record.words,
    ...(record.end ? { end: record.end } : {}),
  }))
  const places = getLabelPlaces(labels)
  return {
    from: 'one',
    to: 'two',
    ...CURVE,
    labels,
    arrows: ends,
    opacity: args.opacity,
    titles: labelled.map((record, index) => {
      const { position, side, heading } = places[index]!
      return createTitle(CURVE, position, record.words, {
        side,
        shift: getShift(side, LABEL_TYPE),
        rise: getRise(side, LABEL_TYPE),
        ...(heading ? { heading } : {}),
      })
    }),
    arrowheads: ends.map((end) => getArrow(CURVE, end)),
  }
}

const createEdgeStory = (args: Knobs) => ({
  components: { PlexEdgeLine, PlexEdgeTitle },
  setup: () => ({ line: getLines([createEdge(args)], 'edge')[0]! }),
  template: `
    <div data-ground style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
      <svg class="numen" width="480" height="180" viewBox="-240 -90 480 180" style="font-family:var(--numen-font-sans)">
        <defs><path v-for="title in line.titles" :key="title.path" :id="title.path" :d="title.line" /></defs>
        <g data-edge="resting">
          <PlexEdgeLine :line="line" />
          <PlexEdgeTitle :line="line" />
        </g>
        <g data-edge="lifted" transform="translate(0 48)">
          <PlexEdgeLine :line="line" is-raised />
          <PlexEdgeTitle :line="line" is-raised />
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
    controls: { exclude: ['line', 'isRaised'] },
    docs: {
      description: {
        component:
          'One edge: the curve, the heads it ends in, and the titles set along ' +
          'it over a halo, one centered on the line, or two set at opposite ends ' +
          'whatever heads the line has. Lifted, it is drawn over the boxes, in ' +
          'the colors and the weight that stand out against them.',
      },
    },
  },
  argTypes: {
    words: { control: 'text' },
    reverseWords: { control: 'text' },
    direction: { control: 'inline-radio', options: ['none', 'forward', 'backward', 'both'] },
    opacity: { control: { type: 'range', min: 0, max: 1, step: 0.05 } },
  },
  args: { words: 'refers to', reverseWords: 'cited by', direction: 'forward', opacity: 1 },
  render: createEdgeStory,
}

export default meta
type Story = StoryObj<Knobs>

/** The pair at rest above, and lifted below. */
export const Playground: Story = {}

const getRestingEdge = (canvas: HTMLElement) => {
  const group = canvas.querySelector('[data-edge="resting"]')!
  const heads = [...group.querySelectorAll('.plex__edge-arrow')].map((head) =>
    head.getAttribute('transform'),
  )
  const titles = [...group.querySelectorAll('.plex__edge-label--letters textPath')].map((path) => [
    path.textContent?.trim(),
    `${Math.round(Number.parseFloat(path.getAttribute('startOffset')!))}%`,
  ])
  return { heads, titles }
}

const AT_TO = `translate(${CURVE.toPoint.x} ${CURVE.toPoint.y})`
const AT_FROM = `translate(${CURVE.fromPoint.x} ${CURVE.fromPoint.y})`

/** A record without a marker: no head, and its title in the middle of the line. */
export const NoDirection: Story = {
  args: { direction: 'none' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getRestingEdge(canvasElement)
    await expect(heads).toStrictEqual([])
    await expect(titles).toStrictEqual([['refers to', '50%']])
  },
}

/** A record pointing at the `to` end: the head there, and its one title in the middle. */
export const Forward: Story = {
  args: { direction: 'forward' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getRestingEdge(canvasElement)
    await expect(heads).toHaveLength(1)
    await expect(heads[0]).toContain(AT_TO)
    await expect(titles).toStrictEqual([['refers to', '50%']])
  },
}

/** A record pointing at the `from` end: the head there, and its one title in the middle. */
export const Backward: Story = {
  args: { direction: 'backward' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getRestingEdge(canvasElement)
    await expect(heads).toHaveLength(1)
    await expect(heads[0]).toContain(AT_FROM)
    await expect(titles).toStrictEqual([['refers to', '50%']])
  },
}

/**
 * A record each way: a head at each end, and the two titles at opposite ends,
 * one on each side of the line and each read in the direction of its record.
 */
export const BothDirections: Story = {
  args: { direction: 'both' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getRestingEdge(canvasElement)
    await expect(heads).toHaveLength(2)
    await expect(heads[0]).toContain(AT_TO)
    await expect(heads[1]).toContain(AT_FROM)
    // The title of the record pointing at the `to` end is near that end, and the other is read from
    // the `to` end back toward the `from` end, so its place on its own path is counted from there.
    await expect(titles).toStrictEqual([
      ['refers to', '80%'],
      ['cited by', '80%'],
    ])
  },
}

/** A line carrying no words, which draws no title and no path to set one on. */
export const NoTitle: Story = {
  args: { words: '' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('text')).toHaveLength(0)
    await expect(canvasElement.querySelectorAll('[data-edge] path')).toHaveLength(4)
    await expect(canvasElement.querySelectorAll('defs path')).toHaveLength(0)
  },
}
