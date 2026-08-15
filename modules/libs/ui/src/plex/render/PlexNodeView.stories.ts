/**
 * One node, on its own. Inside a plex most of what a node does is a fifth of
 * a second long — a handle is one hover away and gone again, a fading node is
 * halfway through a movement — so this is where those states are looked at.
 *
 * A node draws itself around its own coordinates, so every story here supplies
 * a window with the origin in the middle of it, and nothing else.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { computed } from 'vue'
import PlexNodeView from './PlexNodeView.vue'
import { awkwardLabels } from '../fixtures/neighbourhoods'
import { RELATED_ROLES, type PlacedNode, type PlexRole } from '../model'

interface Knobs {
  label: string
  role: PlexRole
  width: number
  height: number
  /** Below one, a node is on its way in or out and cannot be chosen. */
  opacity: number
  offering: boolean
  aimed: boolean

  onActivate: () => void
  onReach: (pointer: PointerEvent) => void
  onHover: (over: boolean) => void
}

const nodeFrom = (a: Knobs, over: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  label: a.label,
  role: a.role,
  x: 0,
  y: 0,
  width: a.width,
  height: a.height,
  order: 0,
  opacity: a.opacity,
  ...over,
})

/**
 * A window in plex units, one to the pixel, with the origin in its middle.
 *
 * Sized rather than stretched: fitting it to the frame would scale every box
 * and letter, and the size of a node against the size of its text is most of
 * what there is to judge here.
 */
const on =
  (scene: (args: Knobs) => readonly PlacedNode[], window = { width: 480, height: 200 }) =>
  (args: Knobs) => ({
    components: { PlexNodeView },
    setup: () => ({ args, window, scene: computed(() => scene(args)) }),
    template: `
      <div style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
        <svg
          :width="window.width"
          :height="window.height"
          :viewBox="[-window.width / 2, -window.height / 2, window.width, window.height].join(' ')"
        >
          <PlexNodeView
            v-for="node in scene"
            :key="node.id"
            :node="node"
            :offering="args.offering"
            :aimed="args.aimed"
            @activate="args.onActivate"
            @reach="args.onReach"
            @hover="args.onHover"
          />
        </svg>
      </div>
    `,
  })

const range = (min: number, max: number, step = 1) => ({
  control: { type: 'range' as const, min, max, step },
})

/**
 * Annotated rather than `satisfies`, unlike the plex's own stories: the knobs
 * are the node's fields, and the node is assembled from them. Tying the args
 * to the component would demand a whole `node` in every story alongside the
 * knobs that make it.
 */
const meta: Meta<Knobs> = {
  title: 'Plex/Node',
  component: PlexNodeView,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A box, a label and the handle to reach out from. Everything it ' +
          'draws with is already on the node it was handed; the two things it ' +
          'cannot know — whether a handle is worth offering here, and whether ' +
          'a gesture would land a link on it — arrive as answers.',
      },
    },
  },

  argTypes: {
    label: { control: 'text' },
    role: { control: 'inline-radio', options: ['focus', ...RELATED_ROLES] },
    width: range(64, 320, 4),
    height: range(20, 96, 2),
    opacity: range(0, 1, 0.05),
    offering: { control: 'boolean' },
    aimed: { control: 'boolean' },

    onActivate: { table: { disable: true } },
    onReach: { table: { disable: true } },
    onHover: { table: { disable: true } },
  },

  args: {
    label: 'A thought',
    role: 'child',
    width: 144,
    height: 36,
    opacity: 1,
    offering: false,
    aimed: false,
    onActivate: fn(),
    onReach: fn(),
    onHover: fn(),
  },

  render: on((args) => [nodeFrom(args)]),
}

export default meta
type Story = StoryObj<Knobs>

/** Every setting a node has, live. */
export const Playground: Story = {}

/**
 * All five seats at once, each where it would sit around a focus.
 *
 * Not a knob turned five times: what is being judged is whether the hues tell
 * one role from another and whether the focus reads as somewhere the reader
 * already is, and neither question can be asked of one node at a time.
 */
export const EverySeat: Story = {
  render: on(
    (args) => [
      nodeFrom(args, { id: 'focus', label: 'Where you are', role: 'focus', width: 176, height: 44 }),
      nodeFrom(args, { id: 'parent', label: 'Above it', role: 'parent', y: -72 }),
      nodeFrom(args, { id: 'child', label: 'Below it', role: 'child', y: 72 }),
      nodeFrom(args, { id: 'jump', label: 'Off to the left', role: 'jump', x: -190 }),
      nodeFrom(args, { id: 'sibling', label: 'Off to the right', role: 'sibling', x: 190 }),
    ],
    { width: 560, height: 240 },
  ),
}

/**
 * The handle, and the one thing about it that is easy to get wrong: it sits
 * inside a node that navigates when clicked, so pressing it must start a
 * gesture and must not also choose the node underneath.
 */
export const Reaching: Story = {
  args: { offering: true },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText('A thought, child')

    await userEvent.hover(node)
    await expect(args.onHover).toHaveBeenLastCalledWith(true)

    // Pressing the handle: a gesture begins and the node is left where it is.
    await userEvent.pointer([{ keys: '[MouseLeft]', target: canvas.getByLabelText(/^Reach out/) }])
    await expect(args.onReach).toHaveBeenCalledTimes(1)
    await expect(args.onActivate).not.toHaveBeenCalled()

    // Pressing the box itself still chooses it.
    await userEvent.click(node)
    await expect(args.onActivate).toHaveBeenCalledTimes(1)

    await userEvent.unhover(node)
    await expect(args.onHover).toHaveBeenLastCalledWith(false)
  },
}

/**
 * Not Latin, far too long, nothing to break at, and nothing at all — the same
 * corpus the plex is shown, laid out in a grid because these are being
 * compared rather than arranged.
 *
 * Only a browser can answer it: jsdom lays out no text, so every label is the
 * same size to it and one that overflows looks identical to one that fits.
 */
export const AwkwardLabels: Story = {
  render: on(
    (args) =>
      awkwardLabels.nodes.map((node, index) =>
        nodeFrom(args, {
          ...node,
          x: ((index % 4) - 1.5) * (args.width + 24),
          y: (Math.floor(index / 4) - 0.5) * (args.height + 32),
        }),
      ),
    { width: 720, height: 200 },
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    // Nothing spills out of the box it was given, whatever the script.
    for (const node of Array.from(canvasElement.querySelectorAll('.plex__node'))) {
      const box = node.querySelector('rect')!.getBoundingClientRect()
      const text = node.querySelector('.plex__label-text')!.getBoundingClientRect()
      await expect(text.width).toBeLessThanOrEqual(box.width)
    }

    // The one with nothing to break at is clamped rather than shrunk to fit.
    const label = canvas
      .getByLabelText(/^Supercalifragilistic/)
      .querySelector('.plex__label-text')!
    await expect(label.scrollHeight).toBeGreaterThan(label.clientHeight)
  },
}

/**
 * On its way in or out, beside one that has arrived — because faintness only
 * means anything against something that is fully there.
 *
 * Such a node cannot be chosen, is not a tab stop and is not announced. Those
 * are negatives, and negatives are asserted next door in jsdom: they are what
 * fails silently and looks right in every screenshot.
 */
export const PartWayThere: Story = {
  render: on((args) => [
    nodeFrom(args, { id: 'staying', label: 'Staying', x: -84 }),
    nodeFrom(args, { id: 'going', label: 'On its way out', x: 84, opacity: 0.35 }),
  ]),
}
