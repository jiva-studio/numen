/**
 * One node, on its own. Inside a plex most of what a node does is a fifth of
 * a second long — a handle is one hover away and gone again, a fading node is
 * halfway through a movement — so this is where those states are looked at.
 *
 * A node draws itself around its own coordinates, so every story here supplies
 * a window with the origin in the middle of it, and nothing else.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { computed } from 'vue'
import PlexNodeView from './PlexNodeView.vue'
import { awkwardLabels } from '../fixtures/neighbourhoods'
import {
  RELATED_SEATS,
  type NodeStanding,
  type PlacedNode,
  type PlexSeat,
  type PlexShowing,
} from '../model'

interface Knobs {
  title: string
  seat: PlexSeat
  width: number
  height: number
  /** Below one, a node is on its way in or out and cannot be chosen. */
  opacity: number
  /** What the node is to a gesture. One setting, because it is one of these. */
  standing: NodeStanding
  /** Fill the icon slot. What goes in it is the application's, not the plex's. */
  icon: boolean

  onActivate: () => void
  onShow: (showing: PlexShowing) => void
  onReach: (pointer: PointerEvent) => void
  onAsk: () => void
}

const STANDINGS: readonly NodeStanding[] = ['open', 'closed', 'source', 'target', 'ghost']

/**
 * Something to put in the slot. Any markup will do — a node has no idea what
 * it is being handed, which is the whole reason it is a slot.
 */
const GLYPH = `
  <svg viewBox="0 0 16 16" width="13" height="13" fill="none"
       stroke="currentColor" stroke-width="1.4" stroke-linejoin="round">
    <path d="M4 2h5l3 3v9H4z" />
    <path d="M9 2v3h3" />
  </svg>
`

const nodeFrom = (a: Knobs, over: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: a.title,
  seat: a.seat,
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
    setup: () => ({ args, window, glyph: GLYPH, scene: computed(() => scene(args)) }),
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
            :standing="args.standing"
            @activate="args.onActivate"
            @show="args.onShow"
            @reach="args.onReach"
            @ask="args.onAsk"
          >
            <template v-if="args.icon" #icon><span v-html="glyph" /></template>
          </PlexNodeView>
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
    // The node is assembled from the knobs below, so the panel's own editor
    // for it would be a second answer to the same question.
    controls: { exclude: ['node'] },
    docs: {
      description: {
        component:
          'A box, a title and the handle to reach out from. Everything it ' +
          'draws with is already on the node it was handed, and whether a ' +
          'pointer is over it is its own affair — hover it and the handle ' +
          'appears. The one thing it cannot work out is what it is to a ' +
          'gesture, which arrives as its standing.',
      },
    },
  },

  argTypes: {
    title: { control: 'text' },
    seat: { control: 'inline-radio', options: ['focus', ...RELATED_SEATS] },
    width: range(64, 320, 4),
    height: range(20, 96, 2),
    opacity: range(0, 1, 0.05),
    icon: { control: 'boolean' },
    standing: { control: 'select', options: STANDINGS },

    onActivate: { table: { disable: true } },
    onShow: { table: { disable: true } },
    onReach: { table: { disable: true } },
    onAsk: { table: { disable: true } },
  },

  args: {
    title: 'A node',
    seat: 'child',
    width: 144,
    height: 36,
    opacity: 1,
    icon: false,
    standing: 'open',
    onActivate: fn(),
    onShow: fn(),
    onReach: fn(),
    onAsk: fn(),
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
 * one seat from another and whether the focus reads as somewhere the reader
 * already is, and neither question can be asked of one node at a time.
 */
export const EverySeat: Story = {
  render: on(
    (args) => [
      nodeFrom(args, { id: 'focus', title: 'Where you are', seat: 'focus', width: 176, height: 44 }),
      nodeFrom(args, { id: 'parent', title: 'Above it', seat: 'parent', y: -72 }),
      nodeFrom(args, { id: 'child', title: 'Below it', seat: 'child', y: 72 }),
      nodeFrom(args, { id: 'jump', title: 'Off to the left', seat: 'jump', x: -190 }),
      nodeFrom(args, { id: 'sibling', title: 'Off to the right', seat: 'sibling', x: 190 }),
    ],
    { width: 560, height: 240 },
  ),
  args: { icon: true },
  play: async ({ canvasElement }) => {
    const ink = (name: string) => {
      const node = canvasElement.querySelector(`[aria-label^="${name}"]`)
      if (!node) throw new Error(`no node ${name}`)
      const icon = node.querySelector('.plex__icon')
      const title = node.querySelector('.plex__title')
      if (!icon || !title) throw new Error(`nothing drawn before ${name}`)
      return [getComputedStyle(icon).color, getComputedStyle(title).color]
    }

    // A node carries its seat's hue before its title. The focused one is
    // painted from that hue, so there the icon takes the title's own ink.
    const [child, childTitle] = ink('Below it')
    expect(child).not.toBe(childTitle)
    const [focus, focusTitle] = ink('Where you are')
    expect(focus).toBe(focusTitle)
  },
}

/**
 * The handle: absent until the node has a hand or the keyboard on it.
 *
 * The thing about it that is easy to get wrong is that it sits inside a node
 * which navigates when pressed, so pressing the handle must start a gesture
 * and must not also choose the node underneath.
 *
 * Only a browser can answer any of it: what tab actually stops on inside an
 * SVG, and whether preventing a pointer's default really keeps focus off.
 */
export const Reaching: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText('A node, child')
    const handle = () => canvasElement.querySelector('.plex__handle')

    // Under the hand, and gone again when it leaves.
    await expect(handle()).toBeNull()
    await userEvent.hover(node)
    await expect(handle()).not.toBeNull()

    // Its disc and its cross are sized by CSS geometry properties, so a token
    // that never arrives is a handle of no size rather than a default one.
    const drawn = handle()!.getBoundingClientRect()
    await expect(drawn.width).toBeGreaterThan(0)
    await expect(canvasElement.querySelector('.plex__handle-mark')!.getBoundingClientRect()
      .width).toBeGreaterThan(0)

    // Pressing it: a gesture begins, and the node is neither chosen nor left
    // holding the focus a press would otherwise give it.
    await userEvent.pointer([{ keys: '[MouseLeft]', target: handle()! }])
    await expect(args.onReach).toHaveBeenCalledTimes(1)
    await expect(args.onActivate).not.toHaveBeenCalled()
    await userEvent.unhover(node)
    await expect(handle()).toBeNull()

    // Under the keyboard as well, or there would be no way to reach out
    // without a pointer at all. Tab stops at the node, then at its handle.
    await userEvent.tab()
    await expect(node).toHaveFocus()
    await waitFor(async () => await expect(handle()).not.toBeNull())

    await userEvent.tab()
    await expect(handle()).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await expect(args.onAsk).toHaveBeenCalledTimes(1)
    await expect(args.onActivate).not.toHaveBeenCalled()

    // Pressing the box itself still chooses it.
    await userEvent.click(node)
    await expect(args.onActivate).toHaveBeenCalledTimes(1)

    // A drag across a title is a drag that meant to reach somewhere, so the
    // text must not come away highlighted under it. Read under the spelling
    // WebKit answers for.
    const title = node.querySelector('.plex__title')!
    await expect(getComputedStyle(title).getPropertyValue('-webkit-user-select')).toBe('none')
  },
}

/**
 * Not Latin, far too long, nothing to break at, and nothing at all — the same
 * corpus the plex is shown, laid out in a grid because these are being
 * compared rather than arranged.
 *
 * Only a browser can answer it: jsdom lays out no text, so every title is the
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
      const text = node.querySelector('.plex__title-text')!.getBoundingClientRect()
      await expect(text.width).toBeLessThanOrEqual(box.width)
    }

    // One line and an ellipsis, never two. The first of these has room to
    // break and the second has none, and neither is any taller for it.
    for (const tooLong of [/^Заметка/, /^Supercalifragilistic/]) {
      const title = canvas.getByLabelText(tooLong).querySelector('.plex__title-text')!
      await expect(title.scrollWidth).toBeGreaterThan(title.clientWidth)
      await expect(title.scrollHeight).toBe(title.clientHeight)
    }
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
    nodeFrom(args, { id: 'staying', title: 'Staying', x: -84 }),
    nodeFrom(args, { id: 'going', title: 'On its way out', x: 84, opacity: 0.35 }),
  ]),
}
