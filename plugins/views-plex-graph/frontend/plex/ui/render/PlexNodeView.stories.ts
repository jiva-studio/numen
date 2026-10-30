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
import { awkwardLabels } from '../../fixtures/neighbourhoods'
import { hoverOver, lightness } from '../../../fixtures/colour'
import { DARK, expectDark } from '../../../fixtures/theme'
import type { GestureRole, PlacedNode } from '../../lib/node'
import { RELATION_NAMES, type PlexNodeRelation } from '../../lib/relation'
import type { PlexDestination } from '../../model/interaction/destination'

interface Knobs {
  title: string
  relation: PlexNodeRelation
  width: number
  height: number
  /** Below one, a node is on its way in or out and cannot be chosen. */
  opacity: number
  /** What the node is to a gesture. One setting, because it is one of these. */
  gestureRole: GestureRole
  /** Fill the icon slot. What goes in it is the application's, not the plex's. */
  hasIcon: boolean

  onActivate: () => void
  onShow: (showing: PlexDestination) => void
  onReach: (pointer: PointerEvent) => void
  onAsk: () => void
}

const ROLES: readonly GestureRole[] = ['open', 'closed', 'source', 'target', 'ghost']

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

const createNode = (a: Knobs, overrides: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: a.title,
  relation: a.relation,
  x: 0,
  y: 0,
  width: a.width,
  height: a.height,
  order: 0,
  opacity: a.opacity,
  ...overrides,
})

/**
 * A window in plex units, one to the pixel, with the origin in its middle.
 *
 * Sized and not stretched: fitting it to the frame would scale every box
 * and letter, and the size of a node against the size of its text is most of
 * what there is to judge here.
 */
const DEFAULT_FRAME = { width: 480, height: 200 }

const renderScene =
  (scene: (args: Knobs) => readonly PlacedNode[], frame = DEFAULT_FRAME) =>
  (args: Knobs) => ({
    components: { PlexNodeView },
    setup: () => ({ args, window: frame, glyph: GLYPH, scene: computed(() => scene(args)) }),
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
            :gesture-role="args.gestureRole"
            @activate="args.onActivate"
            @show="args.onShow"
            @reach="args.onReach"
            @ask="args.onAsk"
          >
            <template v-if="args.hasIcon" #icon><span v-html="glyph" /></template>
          </PlexNodeView>
        </svg>
      </div>
    `,
  })

const range = (min: number, max: number, step = 1) => ({
  control: { type: 'range' as const, min, max, step },
})

/**
 * Annotated and not `satisfies`, unlike the plex's own stories: the knobs
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
          'gesture, which arrives as its role in one.',
      },
    },
  },

  argTypes: {
    title: { control: 'text' },
    relation: { control: 'inline-radio', options: ['focus', ...RELATION_NAMES] },
    width: range(64, 320, 4),
    height: range(20, 96, 2),
    opacity: range(0, 1, 0.05),
    hasIcon: { control: 'boolean' },
    gestureRole: { control: 'select', options: ROLES },

    onActivate: { table: { disable: true } },
    onShow: { table: { disable: true } },
    onReach: { table: { disable: true } },
    onAsk: { table: { disable: true } },
  },

  args: {
    title: 'A node',
    relation: 'child',
    width: 144,
    height: 36,
    opacity: 1,
    hasIcon: false,
    gestureRole: 'open',
    onActivate: fn(),
    onShow: fn(),
    onReach: fn(),
    onAsk: fn(),
  },

  render: renderScene((args) => [createNode(args)]),
}

export default meta
type Story = StoryObj<Knobs>

/** Every setting a node has, live. */
export const Playground: Story = {}

/**
 * All five relations at once, each placed around a focus node.
 *
 * The check is whether the hues distinguish one relation from another and
 * whether the focus node is distinguishable as the current node. Neither can be
 * judged on a single node.
 */
export const EveryRelation: Story = {
  render: renderScene(
    (args) => [
      createNode(args, {
        id: 'focus',
        title: 'Where you are',
        relation: 'focus',
        width: 176,
        height: 44,
      }),
      createNode(args, { id: 'parent', title: 'Above it', relation: 'parent', y: -72 }),
      createNode(args, { id: 'child', title: 'Below it', relation: 'child', y: 72 }),
      createNode(args, { id: 'jump', title: 'Off to the left', relation: 'jump', x: -190 }),
      createNode(args, { id: 'sibling', title: 'Off to the right', relation: 'sibling', x: 190 }),
    ],
    { width: 560, height: 240 },
  ),
  args: { hasIcon: true },
  play: async ({ canvasElement }) => {
    const ink = (name: string) => {
      const node = canvasElement.querySelector(`[aria-label^="${name}"]`)
      if (!node) throw new Error(`no node ${name}`)
      const icon = node.querySelector('.plex__icon')
      const title = node.querySelector('.plex__title')
      if (!icon || !title) throw new Error('node not found')
      return [getComputedStyle(icon).color, getComputedStyle(title).color]
    }

    // A node carries its relation's hue before its title. The focused node is
    // painted from that hue, so its icon takes the title's own color.
    const [child, childTitle] = ink('Below it')
    await expect(child).not.toBe(childTitle)
    const [focus, focusTitle] = ink('Where you are')
    await expect(focus).toBe(focusTitle)
  },
}

/**
 * The handle: absent until the node is hovered or focused by keyboard.
 *
 * Pressing the handle navigates, so it must start a gesture and must not also
 * select the node underneath.
 *
 * Only a browser can verify this: where tab stops inside an SVG, and whether
 * preventing the pointer default keeps focus off the node.
 */
export const Reaching: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText('A node, child')
    const handle = () => canvasElement.querySelector('.plex__handle')

    // Present under the pointer, absent again when the pointer leaves.
    await expect(handle()).toBeNull()
    await userEvent.hover(node)
    await expect(handle()).not.toBeNull()

    // Its disc and its cross are sized by CSS geometry properties, so a token
    // that never arrives is a handle of no size and not a default one.
    const bounds = handle()!.getBoundingClientRect()
    await expect(bounds.width).toBeGreaterThan(0)
    await expect(
      canvasElement.querySelector('.plex__handle-mark')!.getBoundingClientRect().width,
    ).toBeGreaterThan(0)

    // Pressing it starts a gesture; the node is not selected and does not take
    // the focus a press gives by default.
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
 * compared and not arranged.
 *
 * Only a browser can answer it: jsdom lays out no text, so every title is the
 * same size to it and one that overflows looks identical to one that fits.
 */
export const AwkwardLabels: Story = {
  render: renderScene(
    (args) =>
      awkwardLabels.nodes.map((node, index) =>
        createNode(args, {
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

    // One line with an ellipsis, never two. The first title has space to wrap
    // and the second has none, and neither node grows taller.
    for (const tooLong of [/^Заметка/, /^Supercalifragilistic/]) {
      const title = canvas.getByLabelText(tooLong).querySelector('.plex__title-text')!
      await expect(title.scrollWidth).toBeGreaterThan(title.clientWidth)
      await expect(title.scrollHeight).toBe(title.clientHeight)
    }
  },
}

/**
 * The two corner radii of a node: its own, and the larger radius of the node in
 * front.
 *
 * Both are lengths in rem inside the node, and the computed value of a rem is a
 * browser result, so it is checked in a browser.
 */
export const TheCornersAreAsDesigned: Story = {
  render: renderScene(
    (args) => [
      createNode(args, { id: 'child', title: 'Below it', relation: 'child', x: -130 }),
      createNode(args, {
        id: 'focus',
        title: 'Where you are',
        relation: 'focus',
        x: 130,
        width: 176,
      }),
    ],
    { width: 560, height: 200 },
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const getBox = (title: string): SVGRectElement =>
      canvas.getByLabelText(new RegExp(`^${title},`)).querySelector('rect')!

    await expect(getComputedStyle(getBox('Below it')).rx).toBe('6px')
    await expect(getComputedStyle(getBox('Where you are')).rx).toBe('8px')
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
  render: renderScene((args) => [
    createNode(args, { id: 'staying', title: 'Staying', x: -84 }),
    createNode(args, { id: 'going', title: 'On its way out', x: 84, opacity: 0.35 }),
  ]),
}

/**
 * The dark token set, with the pointer over a plain node and over the focused
 * node.
 *
 * Hover mixes part of a node's title color into its background, and the focused
 * node uses a different color pair. Both must move toward their own title color
 * on a dark background as they do on a light one.
 */
export const Dark: Story = {
  globals: DARK,
  render: renderScene(
    (args) => [
      createNode(args, {
        id: 'focus',
        title: 'Where you are',
        relation: 'focus',
        x: -130,
        width: 176,
      }),
      createNode(args, { id: 'child', title: 'Below it', relation: 'child', x: 130 }),
    ],
    { width: 560, height: 200 },
  ),
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const canvas = within(canvasElement)

    const fills: number[] = []
    for (const title of ['Where you are', 'Below it']) {
      const node = canvas.getByLabelText(new RegExp(`^${title},`))
      const box = node.querySelector('rect')!
      const ink = lightness(getComputedStyle(canvas.getByText(title)).color)
      const restingLightness = lightness(getComputedStyle(box).fill)
      fills.push(restingLightness)

      // The title color is compared with the box background, which hover then
      // moves part of the way toward the title color. On the dark token set a
      // plain node is light on dark and the focused node is dark on light.
      await expect(Math.abs(ink - restingLightness)).toBeGreaterThan(40)
      await expect(Math.sign(ink - restingLightness)).toBe(title === 'Where you are' ? -1 : 1)

      await hoverOver(node)
      await waitFor(async () => {
        const moved = lightness(getComputedStyle(box).fill) - restingLightness
        await expect(Math.abs(moved)).toBeGreaterThan(2)
        await expect(Math.sign(moved)).toBe(Math.sign(ink - restingLightness))
      })
    }

    // The focused node uses its own color pair, so its background differs from
    // the adjacent node's.
    await expect(Math.abs(fills[0]! - fills[1]!)).toBeGreaterThan(2)
  },
}
