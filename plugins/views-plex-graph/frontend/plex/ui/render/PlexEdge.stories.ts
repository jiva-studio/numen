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
import { createTitle, getLabelPosition } from '../../lib/arrange'
import { getArrow, type EdgeArrow, type EdgeCurve, type PlacedEdge } from '../../lib/edge'
import { lightness } from '../../../fixtures/colour'
import { DARK, expectDark } from '../../../fixtures/theme'

/**
 * Which records the edge draws: one without a marker, one pointing at the `to`
 * end, one pointing at the `from` end, or one of each.
 */
type Direction = 'none' | 'forward' | 'backward' | 'both'

interface Knobs {
  /** The words of the record that points at the `to` end, or of the only record. */
  words: string
  /** The words of the record that points at the `from` end, drawn where both are. */
  reverseWords: string
  direction: Direction
  /** Below one, the edge is on its way in or out. */
  opacity: number
}

const CURVE: EdgeCurve = {
  fromPoint: { x: -200, y: -40 },
  toPoint: { x: 200, y: 40 },
  control1: { x: -60, y: -40 },
  control2: { x: 60, y: 40 },
}

/** Each record the edge draws: its words and the end it points at, none for no marker. */
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

/** The edge as the arrangement places it, every title at the position of its end. */
const createEdge = (args: Knobs): PlacedEdge => {
  const records = getRecords(args)
  const ends = records.flatMap((record) => (record.end ? [record.end] : []))
  const labelled = records.filter((record) => record.words)
  return {
    from: 'one',
    to: 'two',
    ...CURVE,
    labels: labelled.map((record) =>
      record.end ? { text: record.words, end: record.end } : { text: record.words },
    ),
    arrows: ends,
    opacity: args.opacity,
    titles: labelled.map((record) =>
      createTitle(CURVE, getLabelPosition(record.end), record.words, record.end),
    ),
    arrowheads: ends.map((end) => getArrow(CURVE, end)),
  }
}

/**
 * The pair drawn twice over one window: at rest, and lifted. Each is wrapped in
 * a group of the story's own, because neither drawing carries a name.
 */
const createBoth = (args: Knobs) => ({
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
          'it over a halo, each near the head of its record or in the middle ' +
          'for a record without one. Lifted, it is drawn over the boxes, in ' +
          'the colours and the weight that stand out against them.',
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
  render: createBoth,
}

export default meta
type Story = StoryObj<Knobs>

/** The pair at rest above, and lifted below. */
export const Playground: Story = {}

/** The resting drawing: the arrowheads by where they are put, and the titles by their place on the path. */
const getResting = (canvas: HTMLElement) => {
  const group = canvas.querySelector('[data-edge="resting"]')!
  const heads = [...group.querySelectorAll('.plex__edge-arrow')].map((head) =>
    head.getAttribute('transform'),
  )
  const titles = [...group.querySelectorAll('.plex__edge-label--letters textPath')].map((path) => [
    path.textContent?.trim(),
    path.getAttribute('startOffset'),
  ])
  return { heads, titles }
}

const AT_TO = `translate(${CURVE.toPoint.x} ${CURVE.toPoint.y})`
const AT_FROM = `translate(${CURVE.fromPoint.x} ${CURVE.fromPoint.y})`

/** A record without a marker: no head, and its title in the middle of the line. */
export const NoDirection: Story = {
  args: { direction: 'none' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getResting(canvasElement)
    await expect(heads).toStrictEqual([])
    await expect(titles).toStrictEqual([['refers to', '50%']])
  },
}

/** A record pointing at the `to` end: the head there, and its title near it. */
export const Forward: Story = {
  args: { direction: 'forward' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getResting(canvasElement)
    await expect(heads).toHaveLength(1)
    await expect(heads[0]).toContain(AT_TO)
    await expect(titles).toStrictEqual([['refers to', '80%']])
  },
}

/** A record pointing at the `from` end: the head there, and its title near it. */
export const Backward: Story = {
  args: { direction: 'backward' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getResting(canvasElement)
    await expect(heads).toHaveLength(1)
    await expect(heads[0]).toContain(AT_FROM)
    await expect(titles).toStrictEqual([['refers to', '20%']])
  },
}

/** A record each way: a head at each end, and each title near its own head. */
export const BothDirections: Story = {
  args: { direction: 'both' },
  play: async ({ canvasElement }) => {
    const { heads, titles } = getResting(canvasElement)
    await expect(heads).toHaveLength(2)
    await expect(heads[0]).toContain(AT_TO)
    await expect(heads[1]).toContain(AT_FROM)
    await expect(titles).toStrictEqual([
      ['refers to', '80%'],
      ['cited by', '20%'],
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

/** On its way in or out, where the whole edge fades and the title with it. */
export const PartWayThere: Story = {
  args: { opacity: 0.3 },
  play: async ({ canvasElement }) => {
    const viewElement = canvasElement.querySelectorAll(
      '[data-edge="resting"] path, [data-edge="resting"] text',
    )
    await expect(viewElement).toHaveLength(4)
    for (const one of viewElement) await expect(one.getAttribute('opacity')).toBe('0.3')
  },
}

/** The parts of one edge, in the order they are drawn under a group. */
const getParts = (canvas: HTMLElement, edge: string) => {
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

    const restParts = getParts(canvasElement, 'resting')
    const lifted = getParts(canvasElement, 'lifted')

    // A resting line is still lighter than the surface in the dark set, where
    // lines are drawn lighter than the background.
    await expect(lightness(getComputedStyle(restParts.curve).stroke, ground)).toBeGreaterThan(
      surface + 2,
    )

    // The line and its head form one drawing, and lifting increases the contrast
    // of both against the background. The curve is stroked and the head is
    // filled, so each is read from the property it is painted with.
    await expect(getDistanceFrom(getComputedStyle(lifted.curve).stroke)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(restParts.curve).stroke) + 2,
    )
    await expect(getDistanceFrom(getComputedStyle(lifted.head).fill)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(restParts.head).fill) + 2,
    )

    // The letters likewise, and the halo under them stays the surface, at a
    // weight that covers the line running through the words.
    await expect(getDistanceFrom(getComputedStyle(lifted.letters).fill)).toBeGreaterThan(
      getDistanceFrom(getComputedStyle(restParts.letters).fill) + 2,
    )
    for (const halo of [restParts.halo, lifted.halo]) {
      const renderedStyle = getComputedStyle(halo)
      await expect(getDistanceFrom(renderedStyle.stroke)).toBeLessThan(2)
      await expect(Number.parseFloat(renderedStyle.strokeWidth)).toBeGreaterThan(0)
    }
    await expect(Number.parseFloat(getComputedStyle(lifted.halo).strokeWidth)).toBeGreaterThan(
      Number.parseFloat(getComputedStyle(restParts.halo).strokeWidth),
    )
  },
}
