/**
 * The handle a gesture starts from, on its own and enlarged.
 *
 * Inside a plex it is a disc of nine pixels' radius, visible only while the
 * pointer is over the node, which is not enough to judge a disc, a cross and a
 * focus ring.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { computed } from 'vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import { RELATION_NAMES } from '../../lib/relation'
import type { PlexNodeRelation } from '../../../types'

interface Knobs {
  x: number
  y: number
  relation: PlexNodeRelation
  zoom: number

  onReach: (pointer: PointerEvent) => void
  onAsk: () => void
}

const hue = (relation: PlexNodeRelation) => ({
  '--relation-hue': `var(--numen-relation-${relation})`,
})

const renderScene =
  (
    scene: (
      args: Knobs,
    ) => readonly { id: string; x: number; y: number; relation: PlexNodeRelation }[],
  ) =>
  (args: Knobs) => ({
    components: { PlexNodeHandle },
    setup: () => ({ args, hue, scene: computed(() => scene(args)) }),
    template: `
      <div style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
        <svg width="480" height="200" :viewBox="[-240 / args.zoom, -100 / args.zoom, 480 / args.zoom, 200 / args.zoom].join(' ')">
          <g v-for="one in scene" :key="one.id" :style="hue(one.relation)">
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
    relation: { control: 'inline-radio', options: ['focus', ...RELATION_NAMES] },
    zoom: { control: { type: 'range', min: 1, max: 12, step: 0.5 } },
    onReach: { table: { disable: true } },
    onAsk: { table: { disable: true } },
  },

  args: { x: 0, y: 0, relation: 'child', zoom: 6, onReach: fn(), onAsk: fn() },

  render: renderScene((args) => [{ id: 'one', x: args.x, y: args.y, relation: args.relation }]),
}

export default meta
type Story = StoryObj<Knobs>

export const Playground: Story = {}

/**
 * Pressed by pointer and by keyboard, which differ: a pointer can be dragged to
 * a target and the keyboard cannot.
 *
 * Only a browser can answer it — what tab stops on inside an SVG.
 */
export const Pressing: Story = {
  play: async ({ args, canvasElement }) => {
    const disc = canvasElement.querySelector('.plex__handle')!
    const cross = canvasElement.querySelector('.plex__handle-mark')!
    await expect(cross).not.toBeNull()

    await userEvent.pointer([{ keys: '[MouseLeft]', target: disc }])
    await expect(args.onReach).toHaveBeenCalledTimes(1)

    await userEvent.tab()
    await expect(disc).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await userEvent.keyboard(' ')
    await expect(args.onAsk).toHaveBeenCalledTimes(2)
  },
}
