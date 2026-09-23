/**
 * The handle a gesture leaves from, on its own and blown up.
 *
 * Inside a plex it is a disc of nine pixels' radius, and only there while a
 * hand is over the node, which is no way to judge a disc, a cross and a focus
 * ring.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { computed } from 'vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import { RELATED_SEATS, type PlexSeat } from '../../lib/seat'

interface Knobs {
  /** Where it sits, in the coordinates of whatever draws it. */
  x: number
  y: number
  /** Whose hue it takes. The focus has none, and it falls back to the border. */
  seat: PlexSeat
  /** How far the drawing is scaled up, to look at nine pixels of it. */
  zoom: number

  onReach: (pointer: PointerEvent) => void
  onAsk: () => void
}

/** What a node puts on itself, and the handle inherits. */
const hue = (seat: PlexSeat) => ({ '--numen-seat-hue': `var(--numen-seat-${seat})` })

const renderScene =
  (scene: (args: Knobs) => readonly { id: string; x: number; y: number; seat: PlexSeat }[]) =>
  (args: Knobs) => ({
    components: { PlexNodeHandle },
    setup: () => ({ args, hue, scene: computed(() => scene(args)) }),
    template: `
      <div style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
        <svg width="480" height="200" :viewBox="[-240 / args.zoom, -100 / args.zoom, 480 / args.zoom, 200 / args.zoom].join(' ')">
          <g v-for="one in scene" :key="one.id" :style="hue(one.seat)">
            <PlexNodeHandle :at="one" @reach="args.onReach" @ask="args.onAsk" />
          </g>
        </svg>
      </div>
    `,
  })

const meta: Meta<Knobs> = {
  title: 'Plex/Node Handle',
  component: PlexNodeHandle,
  parameters: {
    layout: 'fullscreen',
    // The point is assembled from the two knobs below.
    controls: { exclude: ['at'] },
    docs: {
      description: {
        component:
          'A disc with a cross on it, drawn about its own origin and put ' +
          'where it belongs by the one point it takes. Every size in it is a ' +
          'token, and the hue is whatever it was hung on.',
      },
    },
  },

  argTypes: {
    x: { control: { type: 'range', min: -100, max: 100, step: 1 } },
    y: { control: { type: 'range', min: -60, max: 60, step: 1 } },
    seat: { control: 'inline-radio', options: ['focus', ...RELATED_SEATS] },
    zoom: { control: { type: 'range', min: 1, max: 12, step: 0.5 } },
    onReach: { table: { disable: true } },
    onAsk: { table: { disable: true } },
  },

  args: { x: 0, y: 0, seat: 'child', zoom: 6, onReach: fn(), onAsk: fn() },

  render: renderScene((args) => [{ id: 'one', x: args.x, y: args.y, seat: args.seat }]),
}

export default meta
type Story = StoryObj<Knobs>

export const Playground: Story = {}

/**
 * One of each, because the hue is not the handle's own: it comes from the node
 * it hangs off, and the focus supplies none at all.
 */
export const EveryHue: Story = {
  render: renderScene((args) =>
    ['focus', ...RELATED_SEATS].map((seat, index) => ({
      id: seat,
      x: (index - 2) * (60 / args.zoom),
      y: 0,
      seat: seat as PlexSeat,
    })),
  ),
  args: { zoom: 4 },
}

/**
 * The disc, the arms of the cross on it and the bar they are drawn with, in the
 * pixels they come to.
 *
 * Every one of them is a length in rem inside the handle itself, and what a rem
 * comes to is the browser's answer and no test's.
 */
export const EverySizeIsAsDesigned: Story = {
  args: { zoom: 1 },
  play: async ({ canvasElement }) => {
    const disc = canvasElement.querySelector('.plex__handle')!
    const across = canvasElement.querySelector('.plex__handle-mark--across')!
    const down = canvasElement.querySelector('.plex__handle-mark--down')!

    // Nine pixels of radius, arms four out from the middle either way, on a bar
    // of one and a half.
    await expect(getComputedStyle(disc).r).toBe('9px')
    await expect(getComputedStyle(across).width).toBe('8px')
    await expect(getComputedStyle(across).height).toBe('1.5px')
    await expect(getComputedStyle(down).width).toBe('1.5px')
    await expect(getComputedStyle(down).height).toBe('8px')
  },
}

/**
 * Pressed by hand and by keyboard, which are two different things: a pointer
 * has somewhere to be dragged to and the keyboard has not.
 *
 * Only a browser can answer it — what tab stops on inside an SVG, and whether
 * a token that never arrives leaves a disc of no size rather than a default.
 */
export const Pressing: Story = {
  play: async ({ args, canvasElement }) => {
    const disc = canvasElement.querySelector('.plex__handle')!
    const cross = canvasElement.querySelector('.plex__handle-mark')!
    await expect(disc.getBoundingClientRect().width).toBeGreaterThan(0)
    await expect(cross.getBoundingClientRect().width).toBeGreaterThan(0)

    await userEvent.pointer([{ keys: '[MouseLeft]', target: disc }])
    await expect(args.onReach).toHaveBeenCalledTimes(1)

    await userEvent.tab()
    await expect(disc).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await userEvent.keyboard(' ')
    await expect(args.onAsk).toHaveBeenCalledTimes(2)
  },
}
