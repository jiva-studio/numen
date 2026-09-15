/**
 * Every situation the plex has to survive. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`.
 *
 * The knobs are flat, so everything a reader might want to turn is a slider, a
 * toggle or a select; the object they build is assembled here.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref } from 'vue'
import Plex from './Plex.vue'
import {
  resolveOptions,
  rowsAndColumns,
  type Placement,
  type PlexOptionsInput,
} from '../lib/arrange'
import { optionsForType, useTypeSize } from '../model/sizing'
import { DWELL } from '../model/dwell'
import { type PlexPart } from '../lib/inside'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import { neighbourhoodOf, walkStart } from '../fixtures/walk'
import { build, buildAround, type TitledNode } from '../fixtures/build'
import { nameNow } from '../fixtures/names'
import { ring } from '../lib/arrange/ring'
import type { PlexEdge } from '../lib/edge'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlexNode, Position } from '../lib/node'
import type { PlexRelatedSeat } from '../lib/seat'
import type { PlexDestination } from '../model/showing'
import type { Clock } from '../model/transition'
import type { MenuOpening } from '@/shared/ui/menu'

interface Knobs {
  neighbourhood: PlexNeighbourhood
  placement: Placement
  showEdgeLabels: boolean
  duration: number
  dwell: number
  onActivate: (id: string) => void
  onShow: (id: string, showing: PlexDestination) => void
  onCreate: (from: string, seat: PlexRelatedSeat) => void
  onLink: (from: string, to: string, seat: PlexRelatedSeat) => void
  onBring: (dragged: readonly string[], seat: PlexRelatedSeat) => void
  onMenu: (id: string, at: Position, opening: MenuOpening) => void
  onDismiss: () => void
  parts: (id: string) => readonly PlexPart[]
  onEnter: (id: string, part: string) => void

  focusWidth: number
  focusHeight: number
  nodeWidth: number
  nodeHeight: number
  gap: number
  lineGap: number
  focusGap: number
  margin: number
  spread: number
  squeeze: number
  maxPerLine: number
  maxLines: number
  maxParts: number
  orientation: 'parents above' | 'parents below'

  curvature: number
  minReach: number
  arrowRoom: number

  arriveAfter: number
  leaveBefore: number

  /** Playground only: how many of each seat to invent. */
  parents: number
  children: number
  jumps: number
  siblings: number
  title: string

  /**
   * What a node made by a gesture is called. Handed in rather than settled
   * here, because naming is the application's business and the plex never
   * learns that a title is a name.
   */
  naming: () => string

  /**
   * What is being dragged over the picture from somewhere else, each of them
   * opaque, and empty while nothing is. The story plays the part of whoever is
   * dragging them, since a plex has no way to pick anything up.
   */
  dragged: readonly string[]

  /** What the shape under the pointer says while something is dragged in. */
  dropName: (seat: PlexRelatedSeat) => string

  /** Component props the panel has no business showing. */
  options?: PlexOptionsInput
  clock?: Clock
}

const getOptions = (a: Knobs): PlexOptionsInput => ({
  focusSize: { width: a.focusWidth, height: a.focusHeight },
  nodeSize: { width: a.nodeWidth, height: a.nodeHeight },
  gap: a.gap,
  lineGap: a.lineGap,
  focusGap: a.focusGap,
  margin: a.margin,
  squeeze: a.squeeze,
  spread: a.spread,
  maxPerLine: a.maxPerLine,
  maxLines: a.maxLines,
  maxParts: a.maxParts,
  routing: { curvature: a.curvature, minReach: a.minReach, arrowRoom: a.arrowRoom },
  motion: { arriveAfter: a.arriveAfter, leaveBefore: a.leaveBefore },
  direction:
    a.orientation === 'parents below'
      ? { parent: 'down', child: 'up', jump: 'left', sibling: 'right' }
      : { parent: 'up', child: 'down', jump: 'left', sibling: 'right' },
})

/**
 * The knobs, at the size the type in a node is being set at. The toolbar says
 * how large the interface is drawn, the label follows it, and the boxes and
 * the clearances are handed in to hold that label.
 */
const optionsFrom = (a: Knobs, type: number): PlexOptionsInput =>
  optionsForType(type, resolveOptions(getOptions(a)))

const range = (min: number, max: number, step = 1, category = 'Layout') => ({
  control: { type: 'range' as const, min, max, step },
  table: { category },
})

const countsFrom = (a: Knobs) => ({
  parent: a.parents,
  child: a.children,
  jump: a.jumps,
  sibling: a.siblings,
})

/**
 * Every story is walkable: choosing a node builds the next neighbourhood
 * around it. The first picture is whatever the story starts from; after that
 * the counts invent one, which is the application's part being played.
 *
 * The neighbourhood is computed: a fresh object on every render tells the plex
 * it has somewhere new to go, and it re-aims once a frame.
 */
const navigable = (start: (args: Knobs) => PlexNeighbourhood) => (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focus = ref<TitledNode | null>(null)
    const cameFrom = ref<TitledNode | null>(null)

    /** What the application would keep: what has been made, and what is new. */
    const made = ref<PlexNode[]>([])
    const links = ref<PlexEdge[]>([])

    const neighbourhood = computed<PlexNeighbourhood>(() => {
      const base = focus.value
        ? buildAround(focus.value, cameFrom.value, countsFrom(args))
        : start(args)
      const here = focus.value?.id ?? base.nodes.find((n) => n.seat === 'focus')?.id
      const mine = made.value.filter((node) =>
        links.value.some(
          (edge) =>
            (edge.from === node.id && edge.to === here) ||
            (edge.to === node.id && edge.from === here),
        ),
      )
      return {
        nodes: [...base.nodes, ...mine],
        edges: [...base.edges, ...links.value.filter((e) => e.from === here || e.to === here)],
      }
    })

    const onActivate = (id: string) => {
      const chosen = neighbourhood.value.nodes.find((node) => node.id === id)
      if (!chosen) return
      const was = neighbourhood.value.nodes.find((node) => node.seat === 'focus')
      cameFrom.value = was ? { id: was.id, title: was.title } : null
      focus.value = { id: chosen.id, title: chosen.title }
    }

    /** A node seated beside another one, linked the way that seat is written. */
    const createSeated = (from: string, title: string, seat: PlexRelatedSeat) => {
      const id = `made/${made.value.length}`
      made.value = [...made.value, { id, title, seat }]
      links.value = [
        ...links.value,
        seat === 'parent' || seat === 'jump'
          ? { from: id, to: from, label: seat === 'jump' ? 'see also' : 'is a' }
          : { from, to: id, label: 'contains' },
      ]
    }

    /**
     * A new node arrives finished: seated, linked and already called
     * something. The gesture ends where the hand let go, and what the node is
     * really to be called is a later idea and somebody else's screen.
     */
    const create = (from: string, seat: PlexRelatedSeat) => {
      createSeated(from, args.naming(), seat)
    }

    /**
     * What was dragged in from outside, each seated beside the focus in the
     * one seat and called by the identifier it was dragged in as. What those
     * identifiers address is the story's to know, and the plex handed them
     * back untouched.
     */
    const dropOnto = (ids: readonly string[], seat: PlexRelatedSeat) => {
      const here = neighbourhood.value.nodes.find((node) => node.seat === 'focus')
      if (!here) return
      for (const one of ids) createSeated(here.id, one, seat)
    }

    const link = (from: string, to: string, seat: PlexRelatedSeat) => {
      links.value = [
        ...links.value,
        seat === 'parent' || seat === 'jump' ? { from: to, to: from } : { from, to },
      ]
    }

    return {
      args,
      onActivate,
      create,
      link,
      dropOnto,
      neighbourhood,
      options: computed(() => optionsFrom(args, type.value)),
    }
  },
  template: `
    <div style="height:100vh">
      <Plex
        :neighbourhood="neighbourhood"
        :placement="args.placement"
        :options="options"
        :show-edge-labels="args.showEdgeLabels"
        :duration="args.duration"
        :dwell="args.dwell"
        :dragged="args.dragged"
        :drop-name="args.dropName"
        :parts="args.parts"
        @enter="(id, part) => args.onEnter(id, part)"
        @activate="onActivate($event); args.onActivate($event)"
        @show="(id, showing) => args.onShow(id, showing)"
        @create="(from, seat) => { create(from, seat); args.onCreate(from, seat) }"
        @link="(from, to, seat) => { link(from, to, seat); args.onLink(from, to, seat) }"
        @bring="(dragged, seat) => { dropOnto(dragged, seat); args.onBring(dragged, seat) }"
        @menu="args.onMenu"
        @dismiss="args.onDismiss"
      />
    </div>
  `,
})

/**
 * Choosing a node moves the focus, which is what the story is for.
 *
 * The neighbourhood is computed: a fresh object on every render tells the plex
 * it has somewhere new to go, and it re-aims once a frame.
 */
const renderWalk = (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focused = ref(walkStart)
    return {
      args,
      focused,
      neighbourhood: computed(() => neighbourhoodOf(focused.value)),
      options: computed(() => optionsFrom(args, type.value)),
    }
  },
  template: `
    <div style="height:100vh">
      <Plex
        :neighbourhood="neighbourhood"
        :placement="args.placement"
        :options="options"
        :show-edge-labels="args.showEdgeLabels"
        :duration="args.duration"
        :dwell="args.dwell"
        @activate="focused = $event; args.onActivate($event)"
        @show="(id, showing) => args.onShow(id, showing)"
      />
    </div>
  `,
})

const meta = {
  title: 'Plex/Plex',
  component: Plex,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'One node in focus, everything else placed by its seat: parents ' +
          'above, children below, jumps to the left, siblings to the right. ' +
          'It takes a neighbourhood and emits an identifier when a node is ' +
          'chosen — it does not fetch anything and does not know what an ' +
          'identifier addresses.',
      },
    },
  },

  argTypes: {
    focusWidth: range(80, 320, 4),
    focusHeight: range(24, 96, 2),
    nodeWidth: range(64, 280, 4),
    nodeHeight: range(20, 80, 2),
    gap: range(0, 64, 2),
    lineGap: range(0, 64, 2),
    focusGap: range(8, 200, 4),
    margin: range(0, 160, 4),
    spread: range(1, 4, 0.1),
    squeeze: range(0.2, 1, 0.05),
    maxPerLine: range(1, 12),
    maxLines: range(1, 8),
    maxParts: range(1, 12),
    orientation: {
      control: 'inline-radio',
      options: ['parents above', 'parents below'],
      table: { category: 'Layout' },
    },
    placement: {
      control: 'inline-radio',
      options: ['rows and columns', 'ring'],
      mapping: { 'rows and columns': rowsAndColumns, ring },
      table: { category: 'Layout' },
    },

    curvature: { ...range(0, 1, 0.05, 'Edges'), table: { category: 'Edges' } },
    minReach: { ...range(0, 120, 2, 'Edges'), table: { category: 'Edges' } },
    arrowRoom: { ...range(0, 40, 1, 'Edges'), table: { category: 'Edges' } },
    showEdgeLabels: { control: 'boolean', table: { category: 'Edges' } },

    duration: {
      control: { type: 'range', min: 0, max: 3000, step: 20 },
      table: { category: 'Motion' },
    },
    dwell: {
      control: { type: 'range', min: 0, max: 2000, step: 20 },
      table: { category: 'Motion' },
    },
    arriveAfter: { ...range(0, 0.95, 0.05, 'Motion'), table: { category: 'Motion' } },
    leaveBefore: { ...range(0.05, 1, 0.05, 'Motion'), table: { category: 'Motion' } },

    parents: range(0, 9, 1, 'Playground'),
    children: range(0, 30, 1, 'Playground'),
    jumps: range(0, 12, 1, 'Playground'),
    siblings: range(0, 12, 1, 'Playground'),
    title: { control: 'text', table: { category: 'Playground' } },

    // Not knobs: the input the caller builds, the clock a test replaces, what
    // it names the nodes it makes, and what the component reports back.
    neighbourhood: { table: { disable: true } },
    options: { table: { disable: true } },
    clock: { table: { disable: true } },
    naming: { table: { disable: true } },
    dragged: { table: { disable: true } },
    dropName: { table: { disable: true } },
    onActivate: { table: { disable: true } },
    onShow: { table: { disable: true } },
    onCreate: { table: { disable: true } },
    onLink: { table: { disable: true } },
    onBring: { table: { disable: true } },
    onMenu: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
    parts: { table: { disable: true } },
    onEnter: { table: { disable: true } },
  },

  args: {
    placement: rowsAndColumns,
    showEdgeLabels: true,
    duration: 420,
    dwell: DWELL,
    onActivate: fn(),
    onShow: fn(),
    onCreate: fn(),
    onLink: fn(),
    onBring: fn(),
    onMenu: fn(),
    onDismiss: fn(),
    parts: () => [],
    onEnter: fn(),
    naming: nameNow,
    dragged: [],
    dropName: (seat: PlexRelatedSeat) => `as ${seat}`,

    focusWidth: 176,
    focusHeight: 44,
    nodeWidth: 144,
    nodeHeight: 36,
    gap: 16,
    lineGap: 20,
    focusGap: 56,
    margin: 16,
    spread: 2.5,
    squeeze: 0.35,
    maxPerLine: 5,
    maxLines: 4,
    maxParts: 6,
    orientation: 'parents above',

    curvature: 0.55,
    minReach: 22,
    arrowRoom: 14,
    arriveAfter: 0.35,
    leaveBefore: 0.45,

    parents: 2,
    children: 6,
    jumps: 3,
    siblings: 2,
    title: 'A node',
  },

  render: navigable((args) => args.neighbourhood),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

/**
 * Every setting, live, and a plex to walk while turning them. The counts
 * invent the neighbourhood; choosing a node builds the next one around it.
 */
const invented = {
  args: { neighbourhood: neighbourhoods.typical, maxPerLine: 9 },
  render: navigable((args: Knobs) => build(args.title, countsFrom(args))),
}

export const Playground: Story = invented

/**
 * Reaching out of a node, and what comes back.
 *
 * Its own story rather than a check on the playground: a play function runs
 * the moment a story is opened, and a plex that has already made a node is
 * not the one anyone meant to look at.
 *
 * It hands in a naming of its own, which is both how the check knows what to
 * look for and the whole point being shown — the plex reports the gesture and
 * the caller decides what the node is called.
 */
export const MakingOne: Story = {
  args: { ...invented.args, naming: () => 'Reached for' },
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)
    const box = focus.getBoundingClientRect()

    // The handle is offered on hover and not before.
    await expect(canvasElement.querySelector('.plex__handle')).toBeNull()
    await userEvent.hover(focus)
    const surface = canvasElement.querySelector('svg')!
    const handle = canvasElement.querySelector('.plex__handle')
    await expect(handle).not.toBeNull()

    // Nothing the hand crosses on the way is text to select — the labels a
    // connection carries least of all, since a gesture passes right over them.
    // Read under the spelling WebKit answers for.
    await expect(
      getComputedStyle(canvasElement.querySelector('.plex__edge-label')!).getPropertyValue(
        '-webkit-user-select',
      ),
    ).toBe('none')

    // Reaching upwards, where the parents are. Only a browser can answer this:
    // pointer capture, a real matrix, and the drawing that follows the pointer.
    const above = { clientX: box.x + box.width / 2, clientY: box.y - 220 }
    await userEvent.pointer([
      {
        keys: '[MouseLeft>]',
        target: handle!,
        coords: { clientX: box.right, clientY: box.y + box.height / 2 },
      },
      { target: surface, coords: above },
    ])
    await expect(canvasElement.querySelector('.plex__thread')).not.toBeNull()
    await expect(
      canvasElement.querySelector('.plex__node--ghost .plex__title-text')?.textContent,
    ).toBe('parent')

    // Let go by hand: a second `userEvent.pointer` call does not know a button
    // is still down from the first, so its release is a no-op.
    surface.dispatchEvent(new PointerEvent('pointerup', { ...above, pointerId: 1, bubbles: true }))
    await expect(args.onCreate).toHaveBeenCalledWith(expect.any(String), 'parent')

    // No handle is left behind on the node it came from. Only a browser can
    // answer it: whether letting go of a captured pointer somewhere else says
    // so to the node it was captured from.
    await expect(canvasElement.querySelector('.plex__handle')).toBeNull()

    // The story answers by seating the node, already called what its own
    // naming said to call it. Nothing is asked for and nothing is typed.
    await waitFor(async () => {
      await expect(canvas.getByLabelText('Reached for, parent')).toBeInTheDocument()
    })
  },
}

/**
 * Something dragged over the picture from outside it.
 *
 * The gesture starts where the plex cannot see it, so all the plex is handed
 * is a list of identifiers and all it answers is a seat, measured from the
 * focus. Several are one line and one seat, and whoever is dragging them says
 * how many, at the pointer, in its own words.
 *
 * The line and the words are drawn while the pointer travels, so what letting
 * go would do is plain before it happens — which is what this is here to be
 * looked at for, and it is left mid-drag.
 *
 * Only a browser can answer any of it: a real matrix, a pointer the plex never
 * took hold of, and a drawing that follows it across.
 */
export const DraggingThemIn: Story = {
  args: {
    ...invented.args,
    dragged: ['physics/Entropy.md', 'physics/Kelvin.md', 'Heat.md'],
  },
  render: invented.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)
    const box = focus.getBoundingClientRect()
    const middle = box.x + box.width / 2

    const dragTo = (clientX: number, clientY: number) =>
      window.dispatchEvent(
        new PointerEvent('pointermove', { clientX, clientY, pointerId: 1, bubbles: true }),
      )
    const expectDraggedTitle = async (words: string | null) => {
      await waitFor(async () => {
        await expect(
          canvasElement.querySelector('.plex__dragged .plex__title-text')?.textContent ?? null,
        ).toBe(words)
      })
    }

    // Above the focus, where the parents are, and in the words the story gave.
    dragTo(middle, box.y - 220)
    await expectDraggedTitle('as parent')
    await expect(canvasElement.querySelector('.plex__dragged .plex__thread')).not.toBeNull()

    // Below it, where the children are: the words follow the pointer.
    dragTo(middle, box.bottom + 220)
    await expectDraggedTitle('as child')

    // Off the edge of the plex, where letting go would join nothing: nothing
    // is promised, and the line goes with the promise.
    dragTo(middle, -400)
    await expectDraggedTitle(null)
    await expect(canvasElement.querySelector('.plex__dragged')).toBeNull()

    // Back over the picture, and left there to be looked at.
    dragTo(middle, box.y - 220)
    await expectDraggedTitle('as parent')
  },
}

/** Whether a focus reads as the keyboard's. Browsers take it; the types do not. */
interface KeyboardFocus extends FocusOptions {
  focusVisible?: boolean
}

/**
 * Where the handle is, and how few of them there are.
 *
 * A handle belongs to the box under the hand, and to the box the keyboard is
 * visibly on. A click leaves a box holding a focus that is neither, so a box
 * clicked and then left alone wears nothing at all.
 *
 * Only a browser can answer any of it: a click and a tab leave the same focus
 * behind, and which of them the keyboard is visibly on is the browser's own
 * reckoning. A hand driven from a story is untrusted and does not reach that
 * reckoning, so the focus a click leaves behind is named here outright.
 */
export const OneHandleAtATime: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const getHandles = () => canvasElement.querySelectorAll('.plex__handle')
    const boxes = [
      canvas.getByLabelText(/, focus$/),
      ...canvas.getAllByLabelText(/, (parent|child|jump|sibling)$/).slice(0, 3),
    ]
    const first = boxes[0]!

    // A picture nobody is on offers nothing.
    await expect(getHandles()).toHaveLength(0)

    // The focus a click leaves is one the keyboard is not visibly on, and no
    // hand is anywhere near.
    first.focus({ focusVisible: false } as KeyboardFocus)
    await expect(first).toHaveFocus()
    await expect(getHandles()).toHaveLength(0)

    // The hand across the picture, box after box. One handle in the whole
    // plex, on the box the hand is on, and none left behind it.
    for (const box of boxes) {
      await userEvent.hover(box)
      await expect(getHandles()).toHaveLength(1)
      await expect(box.querySelector('.plex__handle')).not.toBeNull()
      await userEvent.unhover(box)
      await expect(getHandles()).toHaveLength(0)
    }

    // Arriving by tab is the keyboard being used, and the box it stops at
    // wears one.
    first.blur()
    await userEvent.tab()
    const stop = canvasElement.ownerDocument.activeElement!
    await expect(stop).toHaveClass('plex__node')
    await expect(getHandles()).toHaveLength(1)
    await expect(stop.querySelector('.plex__handle')).not.toBeNull()
  },
}

/**
 * Asking a node for a menu.
 *
 * The plex says which node was asked about, where, and from what. What the
 * menu holds and what choosing an item does never reach this far.
 *
 * Only a browser can answer any of it: whether a `contextmenu` lands on an SVG
 * group at all, whether refusing it takes the webview's own menu away, and
 * whether the right button over the handle reaches the node under it instead
 * of reaching out from it.
 */
export const AskingForAMenu: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    // Read after the node has had it: a listener above sees what the node did.
    let refused: boolean | null = null
    const watch = (event: Event) => {
      refused = event.defaultPrevented
    }
    canvasElement.addEventListener('contextmenu', watch)
    await userEvent.pointer({ keys: '[MouseRight]', target: focus })
    canvasElement.removeEventListener('contextmenu', watch)

    // The focus is the note being read, and the likeliest one to ask about.
    await expect(args.onMenu).toHaveBeenCalledWith(
      expect.any(String),
      expect.any(Object),
      'pointer',
    )
    await expect(refused).toBe(true)

    // The handle sits on the trailing edge, which is where a person aims. The
    // right button over it reaches the node, and makes nothing.
    await userEvent.hover(focus)
    const handle = canvasElement.querySelector('.plex__handle')!
    await userEvent.pointer({ keys: '[MouseRight]', target: handle })
    await expect(args.onCreate).not.toHaveBeenCalled()
    await expect(canvasElement.querySelector('.plex__thread')).toBeNull()
    await expect(args.onMenu).toHaveBeenCalledTimes(2)
  },
}

/**
 * Asking for a node itself, rather than travelling to it.
 *
 * Asked on the focus, which is the one node a click moves nothing on, so what
 * a double click leaves behind is the asking on its own.
 *
 * Only a browser can answer any of it: whether a double click reaches an SVG
 * group at all, and whether a modifier held over it arrives with it.
 */
export const AskingForANode: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    // One session throughout, so that a modifier held down is still held when
    // the hand arrives.
    const hand = userEvent.setup()

    await hand.dblClick(focus)
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'here')
    await expect(args.onActivate).not.toHaveBeenCalled()

    // The same gesture with the modifier held asks for it beside where the
    // reader is. What that means is the application's, and never reaches here.
    await hand.keyboard('{Alt>}')
    await hand.dblClick(focus)
    await hand.keyboard('{/Alt}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'beside')

    // The keyboard route: a press with Shift, and the same modifier for where.
    focus.focus()
    await hand.keyboard('{Shift>}{Enter}{/Shift}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'here')
    await expect(args.onShow).toHaveBeenCalledTimes(3)
  },
}

/** Two parents, one a parent of the other: the only edge inside a row. */
export const SeveralParents: Story = {
  args: { neighbourhood: neighbourhoods.diamond },
}

/** Past every limit. What the overflow line at the bottom is for. */
export const Overcrowded: Story = {
  args: { neighbourhood: neighbourhoods.overcrowded },
}

/**
 * Titles on every shape of line a plex draws: across the page, down it, and
 * round out of a column. One of them is far longer than the curve it is set
 * on, and is cut to it.
 */
const titledLines: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', title: 'The meeting', seat: 'focus' },
    { id: 'parent', title: 'Minutes', seat: 'parent' },
    { id: 'earlier', title: 'The morning', seat: 'jump' },
    { id: 'across', title: 'Elsewhere', seat: 'jump' },
    { id: 'later', title: 'The evening', seat: 'jump' },
    { id: 'sibling', title: 'The hedge', seat: 'sibling' },
  ],
  edges: [
    { from: 'parent', to: 'focus', label: 'is a' },
    { from: 'parent', to: 'sibling', label: 'contains' },
    { from: 'earlier', to: 'focus', label: 'see also' },
    { from: 'across', to: 'focus', label: 'the minutes of the meeting' },
    { from: 'later', to: 'focus', label: 'and also' },
  ],
}

/**
 * What a line does with the title it carries: it is set along it, and cut to
 * it where the words are longer.
 *
 * Only a browser can answer any of it: how wide the words are in the type they
 * are set in, how long the curve is that they are set on, and whether the two
 * layers a title is painted in land on one another.
 *
 * The picture arrives at once, since a title is a property of a line that has
 * come to rest, and a line on its way somewhere cannot be pointed at.
 */
export const TitledLines: Story = {
  // At the settings, so the story is about the cutting and not about how far
  // a roomy window opens the gaps a title is set along.
  args: { neighbourhood: titledLines, duration: 0, spread: 1 },
  play: async ({ canvasElement }) => {
    /** Both layers of one title: the halo first, the letters over it. */
    const title = (words: string) =>
      [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')].filter(
        (text) => text.textContent === words,
      )

    /** The line a title is set along, by the path it names. */
    const lineOf = (text: SVGTextElement) => {
      const href = text.querySelector('textPath')!.getAttribute('href')!
      return canvasElement.querySelector<SVGPathElement>(`defs path[id="${href.slice(1)}"]`)!
    }

    // The window is measured after the first drawing, so the plex settles on
    // its second.
    await waitFor(async () => {
      await expect(title('contains')).toHaveLength(2)
    })

    // Every title is set along the line it belongs to, and none of them is
    // drawn wider than the curve it is set on.
    const titles = [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')]
    await expect(titles.length).toBeGreaterThan(0)
    for (const text of titles) {
      await expect(text.querySelector('textPath')).not.toBeNull()
      await expect(text.getComputedTextLength()).toBeLessThanOrEqual(lineOf(text).getTotalLength())
    }

    // Words longer than their curve are cut to it and end in an ellipsis.
    await expect(title('the minutes of the meeting')).toHaveLength(0)
    const cut = titles.filter((text) => text.textContent!.endsWith('…'))
    await expect(cut).toHaveLength(2)
    await expect('the minutes of the meeting'.startsWith(cut[0]!.textContent!.slice(0, -1))).toBe(
      true,
    )

    // A title is painted in two layers that land on one another.
    const [halo, letters] = title('contains')
    await expect(halo!.querySelector('textPath')).not.toBeNull()
    await expect(getComputedStyle(halo!).fill).toBe('none')
    await expect(getComputedStyle(letters!).stroke).toBe('none')
    await expect(halo!.getComputedTextLength()).toBe(letters!.getComputedTextLength())
    const haloAtRest = parseFloat(getComputedStyle(halo!).strokeWidth)

    // The hand on that line lifts it, and the lifted title is painted the same
    // way on a heavier halo.
    const along = halo!.querySelector('textPath')!.getAttribute('href')!
    const drawn = canvasElement
      .querySelector(`defs path[id="${along.slice(1)}"]`)!
      .getAttribute('d')
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === drawn,
    )
    await userEvent.hover(band!)

    const lifted = canvasElement.querySelectorAll<SVGTextElement>('.plex__lift .plex__edge-label')
    await expect(lifted).toHaveLength(2)
    await expect(getComputedStyle(lifted[0]!).fill).toBe('none')
    await expect(getComputedStyle(lifted[1]!).stroke).toBe('none')
    await expect(parseFloat(getComputedStyle(lifted[0]!).strokeWidth)).toBeGreaterThan(haloAtRest)
    await expect(lifted[0]!.getComputedTextLength()).toBe(lifted[1]!.getComputedTextLength())
    await userEvent.unhover(band!)

    // A jump that leaves its column and comes round is set along its line like
    // any other, and its words are short enough to stay whole.
    const loop = title('see also')[0]!
    await expect(loop.querySelector('textPath')).not.toBeNull()
    await expect(loop.getComputedTextLength()).toBeLessThan(lineOf(loop).getTotalLength())
  },
}

/** What two things on the screen are held apart by. */
interface Edges {
  left: number
  right: number
  top: number
  bottom: number
}

/**
 * Every title that is drawn stands clear: none of its letters lies over a box,
 * and none of them lies over another title's.
 *
 * Two counts, and the assertion is the count itself, so a picture that gets
 * worse says by how much.
 */
const expectTitlesClear = async ({ canvasElement }: { canvasElement: HTMLElement }) => {
  /** Whether two boxes on the screen share any of it. */
  const isOverlapping = (one: Edges, other: Edges) =>
    one.left < other.right &&
    other.left < one.right &&
    one.top < other.bottom &&
    other.top < one.bottom

  /**
   * Where each letter of a title stands, on the screen. A title set along a
   * curve is a ribbon of letters: the box around the whole of a diagonal one
   * covers a quarter of the picture the letters are nowhere near.
   */
  const lettersOf = (text: SVGTextElement): Edges[] => {
    const onScreen = text.getScreenCTM()!
    const letters: Edges[] = []

    for (let at = 0; at < text.getNumberOfChars(); at += 1) {
      const letter = text.getExtentOfChar(at)
      const corners = [
        new DOMPoint(letter.x, letter.y),
        new DOMPoint(letter.x + letter.width, letter.y),
        new DOMPoint(letter.x, letter.y + letter.height),
        new DOMPoint(letter.x + letter.width, letter.y + letter.height),
      ].map((corner) => corner.matrixTransform(onScreen))

      letters.push({
        left: Math.min(...corners.map((corner) => corner.x)),
        right: Math.max(...corners.map((corner) => corner.x)),
        top: Math.min(...corners.map((corner) => corner.y)),
        bottom: Math.max(...corners.map((corner) => corner.y)),
      })
    }

    return letters
  }

  /** The letters of each title, which is the layer the halo is painted for. */
  const titles = () => [
    ...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label--letters'),
  ]

  // The plex draws on a fallback window until the canvas has been measured,
  // and the counts below belong to the arrangement the measured one settles.
  await waitFor(async () => {
    const svg = canvasElement.querySelector('svg')!
    const drawnFor = Number(svg.getAttribute('viewBox')!.split(' ')[2])
    await expect(Math.abs(drawnFor - svg.getBoundingClientRect().width)).toBeLessThan(1)
  })

  const drawn = titles().map((text) => ({
    words: text.textContent,
    letters: lettersOf(text),
  }))
  const boxes = [...canvasElement.querySelectorAll<SVGRectElement>('.plex__box')].map((box) =>
    box.getBoundingClientRect(),
  )

  const piled: string[] = []
  for (const [at, title] of drawn.entries()) {
    for (const other of drawn.slice(at + 1)) {
      const touching = title.letters.some((letter) =>
        other.letters.some((mine) => isOverlapping(letter, mine)),
      )
      if (touching) piled.push(`${title.words} × ${other.words}`)
    }
  }

  const overBoxes = drawn
    .filter((title) =>
      title.letters.some((letter) => boxes.some((box) => isOverlapping(letter, box))),
    )
    .map((title) => title.words)

  await expect(drawn.length).toBeGreaterThan(0)
  await expect(piled).toHaveLength(0)
  await expect(overBoxes).toHaveLength(0)
}

/**
 * How much of the picture the titles cover each other in. The playground's own
 * neighbourhood, which is the crowded one: a fan of lines out of every node,
 * many of them saying the same word.
 *
 * Two counts, and the assertion is the count itself, so a picture that gets
 * worse says by how much. Only a browser can answer either: how wide the words
 * are in the type they are set in, and where the glyphs land once they are
 * bent along their curves.
 *
 * It stands on its own, since the playground is a plex to turn knobs on.
 */
export const TitlesFindRoom: Story = {
  args: { ...invented.args, duration: 0 },
  render: invented.render,
  play: expectTitlesClear,
}

/**
 * The lines of one note, every one of them named, and more children than a row
 * holds.
 *
 * The lines to the far row cross the near one, so their titles are looking for
 * room in a band the near row's titles already stand in. A title with nowhere
 * clear is cut to the room it has, and one with room for less than half of its
 * words is not written at all.
 */
export const TitlesAcrossRows: Story = {
  args: { neighbourhood: neighbourhoods.labelledRows, duration: 0 },
  play: expectTitlesClear,
}

/**
 * One note's lines: two the plex is asked to draw an arrow on, one it is not,
 * and a sibling's line hanging off the parent the two of them share.
 */
const arrowedLines: PlexNeighbourhood = {
  nodes: [
    { id: 'alice', title: 'Alice Fenn, The Chair', seat: 'focus' },
    { id: 'allotments', title: 'Marrowfield allotments', seat: 'parent' },
    { id: 'bram', title: 'Bram Doyle', seat: 'jump' },
    { id: 'cora', title: 'Cora Hale', seat: 'jump' },
    { id: 'rota', title: 'The question of the rota', seat: 'sibling' },
  ],
  edges: [
    { from: 'allotments', to: 'alice', arrow: 'from' },
    { from: 'bram', to: 'alice', label: 'the oldest tenant', arrow: 'from' },
    { from: 'cora', to: 'alice', label: 'keys, hoses, the gate' },
    { from: 'allotments', to: 'rota', label: 'the minutes of the meeting' },
  ],
}

/**
 * The arrow a line carries, drawn at whichever of its two ends it was given.
 *
 * Only a browser can answer any of it: whether a head comes out of the style
 * at all, where it lands once the turn it is given is applied, and which side
 * of the box it ends up on.
 *
 * The picture arrives at once: a line on its way somewhere has not reached the
 * end its arrow belongs at.
 */
export const ArrowedLines: Story = {
  args: { neighbourhood: arrowedLines, duration: 0 },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const svg = canvasElement.querySelector('svg')!

    /** Whether a box holds a point, give or take a pixel. */
    const hasPoint = (box: DOMRect, at: DOMPoint) =>
      at.x >= box.left - 1 && at.x <= box.right + 1 && at.y >= box.top - 1 && at.y <= box.bottom + 1

    /** Every line drawn, with both its ends where they land on the screen. */
    const lines = () => {
      const onScreen = svg.getScreenCTM()!
      return [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge')].map((line) => ({
        line,
        ends: [line.getPointAtLength(0), line.getPointAtLength(line.getTotalLength())].map(
          (point) => point.matrixTransform(onScreen),
        ),
      }))
    }

    /** The line that touches a node's box, and the end of it that does. */
    const lineAt = (name: string) => {
      const box = canvas.getByLabelText(name).getBoundingClientRect()
      for (const { line, ends } of lines()) {
        const end = ends.find((point) => hasPoint(box, point))
        if (end) return { line, end }
      }
      throw new Error(`no line touches ${name}`)
    }

    const endAt = (name: string) => lineAt(name).end

    const heads = () => [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-arrow')]

    /** Whether an arrowhead was drawn on a given end of a line. */
    const headAt = (end: DOMPoint) =>
      heads().some((head) => hasPoint(head.getBoundingClientRect(), end))

    /** Where every head and every end of every line stands, as one reading. */
    const reading = () =>
      JSON.stringify([
        heads().map((head) => {
          const box = head.getBoundingClientRect()
          return [box.x, box.y, box.width, box.height]
        }),
        lines().map(({ ends }) => ends.map((end) => [end.x, end.y])),
      ])

    /** A frame, after which what was arranged on the last one can be read. */
    const frame = () => new Promise((done) => requestAnimationFrame(() => done(null)))

    // The window is measured after the first drawing, so the plex settles on
    // its second, and the picture stands still once it has.
    await waitFor(async () => {
      await expect(heads()).toHaveLength(2)
      const was = reading()
      await frame()
      await expect(reading()).toBe(was)
    })

    // A head is really drawn, and is not an empty path.
    for (const head of heads()) {
      const box = head.getBoundingClientRect()
      await expect(box.width).toBeGreaterThan(0)
      await expect(box.height).toBeGreaterThan(0)
    }

    // Each of them at the far end of its own line, away from the focus.
    await expect(headAt(endAt('Marrowfield allotments, parent'))).toBe(true)
    await expect(headAt(endAt('Bram Doyle, jump'))).toBe(true)

    // The lines given none carry none, the sibling's line among them.
    await expect(headAt(endAt('Cora Hale, jump'))).toBe(false)
    await expect(headAt(endAt('The question of the rota, sibling'))).toBe(false)

    // The head stands outside the box it points into: it is turned along the
    // line, and its tip is what touches the border.
    const jump = lineAt('Bram Doyle, jump')
    const box = canvas.getByLabelText('Bram Doyle, jump').getBoundingClientRect()
    const head = heads().find((one) => hasPoint(one.getBoundingClientRect(), jump.end))!
    const drawn = head.getBoundingClientRect()
    await expect(
      hasPoint(box, new DOMPoint(drawn.x + drawn.width / 2, drawn.y + drawn.height / 2)),
    ).toBe(false)

    // The title on that line is cut short of the head, at either end of the
    // words: neither the letters nor the halo under them reach it.
    const clear = Math.max(drawn.width, drawn.height)
    const title = [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')].find(
      (text) => text.textContent!.startsWith('the oldest'),
    )!
    const letters = title.getNumberOfChars()
    const onScreen = svg.getScreenCTM()!
    for (const glyph of [
      title.getStartPositionOfChar(0),
      title.getEndPositionOfChar(letters - 1),
    ]) {
      const at = new DOMPoint(glyph.x, glyph.y).matrixTransform(onScreen)
      await expect(Math.hypot(at.x - jump.end.x, at.y - jump.end.y)).toBeGreaterThan(clear)
    }

    // The hand on that line lifts it, and the head is brightened with it.
    const resting = getComputedStyle(head).fill
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === jump.line.getAttribute('d'),
    )!
    await userEvent.hover(band)

    const lifted = canvasElement.querySelector<SVGPathElement>('.plex__lift .plex__edge-arrow')!
    await expect(lifted).not.toBeNull()
    await expect(getComputedStyle(lifted).fill).not.toBe(resting)
  },
}

/**
 * Not Latin, far too long, nothing to break at, and nothing at all — the whole
 * arrangement made of them, to see that text nothing fits into still leaves a
 * plex rather than a pile.
 *
 * Whether a title stays inside its own box is the node's affair and is checked
 * on the node, against the same corpus.
 */
export const AwkwardLabels: Story = {
  args: { neighbourhood: neighbourhoods.awkwardLabels },
}

/**
 * A title too long for its box, read by leaving a hand on the box.
 *
 * Only a browser can answer it: how wide a title runs is what the type it is
 * set in comes to, whether the box holds the whole of it is the browser's own
 * reckoning, and a box drawn over its neighbours is a matter of what was
 * painted last.
 */
export const RestingOnATitle: Story = {
  args: { neighbourhood: neighbourhoods.awkwardLabels },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText(/^Supercalifragilistic/)
    const beside = canvas.getByLabelText('日本語のノート, child')

    const widthOf = (box: Element) => box.querySelector('.plex__box')!.getBoundingClientRect().width
    const placed = widthOf(node)
    const nextDoor = beside.getBoundingClientRect().x

    // Cut short in the box the arrangement drew, and the whole of it under a
    // hand that stays.
    const words = node.querySelector<HTMLElement>('.plex__title-text')!
    await expect(words.scrollWidth).toBeGreaterThan(words.clientWidth)

    await userEvent.hover(node)
    await waitFor(async () => await expect(widthOf(node)).toBeGreaterThan(placed), {
      timeout: 3000,
    })
    await waitFor(
      async () => await expect(words.scrollWidth).toBeLessThanOrEqual(words.clientWidth),
    )

    // Nothing else moved for it, and it stands over what it now covers.
    await expect(beside.getBoundingClientRect().x).toBe(nextDoor)
    const boxes = canvasElement.querySelectorAll('.plex__node')
    await expect(boxes[boxes.length - 1]).toBe(node)

    // The hand leaves, and the picture is the one it was.
    await userEvent.unhover(node)
    await waitFor(async () => await expect(widthOf(node)).toBe(placed))
  },
}

/**
 * Walk the graph. Go back the way you came: nodes in both pictures travel
 * rather than blinking out and in, because they are matched by identifier.
 */
export const Walk: Story = {
  args: {
    neighbourhood: neighbourhoodOf(walkStart),
    gap: 18,
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const middle = canvasElement.getBoundingClientRect()

    /**
     * The plex has arrived when the node named stands in the middle of it.
     *
     * Measured on the box rather than the group around it: a node that has
     * just been clicked holds the focus, and so wears a handle that hangs off
     * its trailing edge and would count towards the group's width.
     */
    const waitForNode = async (label: string) =>
      await waitFor(
        async () => {
          const box = canvas
            .getByLabelText(label)
            .querySelector('.plex__box')!
            .getBoundingClientRect()
          const centre = box.x + box.width / 2
          await expect(Math.abs(centre - (middle.x + middle.width / 2))).toBeLessThan(2)
        },
        { timeout: 3000 },
      )

    // The keyboard, in a real browser: whether tab actually stops on an SVG
    // group. The focus is a stop too, although it cannot be chosen — a menu is
    // asked for from wherever the keyboard is.
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveAttribute(
      'tabindex',
      '0',
    )
    await userEvent.tab()
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveFocus()

    // Past the handle the node under the keyboard now offers, onto the seat
    // above it.
    await userEvent.tab()
    await userEvent.tab()
    await expect(canvas.getByLabelText('Architecture, parent')).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await expect(args.onActivate).toHaveBeenCalledWith('architecture')

    await waitForNode('Architecture, focus')

    // Go back the way you came. Where it ends up is what a browser is needed
    // for: that a click on an SVG group lands, and that the plex settles with
    // the chosen node centred.
    await userEvent.click(canvas.getByLabelText('Hexagonal architecture, child'))
    await waitForNode('Hexagonal architecture, focus')

    // The node walked away from took the seat above, and the plex is a plex
    // again.
    await expect(canvas.getByLabelText('Architecture, parent')).toBeInTheDocument()
  },
  parameters: {
    docs: {
      description: {
        story:
          'The graph, the parent/child/jump rules and the sibling derivation ' +
          'all live in the story, because that is exactly the work the ' +
          'application does and exactly the work the plex must not.',
      },
    },
  },
  render: renderWalk,
}

/**
 * Everything hung parts have to survive, one node each: a nested set, far more
 * than fit, one alone, one that runs on past the box, parts not written in
 * Latin, a part with no words at all, and nodes with none.
 */
const INSIDE: Record<string, readonly PlexPart[]> = {
  focus: [
    { id: '0', text: 'What it is', level: 1 },
    { id: '4', text: 'Where it came from', level: 2 },
    { id: '9', text: 'The first account', level: 3 },
    { id: '14', text: 'The second', level: 3 },
    { id: '20', text: 'What follows from it', level: 2 },
    { id: '31', text: 'Notes', level: 1 },
  ],
  'focus/child-0': [{ id: '2', text: 'The only heading in it', level: 1 }],
  'focus/child-1': [
    {
      id: '3',
      text: 'A heading that runs on well past anything the box it hangs from could hold',
      level: 1,
    },
  ],
  'focus/child-2': [{ id: '5', text: '', level: 1 }],
  'focus/parent-0': [
    { id: '1', text: 'Что в сарае', level: 1 },
    { id: '6', text: '日本語の見出し', level: 2 },
    { id: '11', text: 'مدخل بالعربية', level: 2 },
  ],
  // The one node there is exactly one of, so the play can find it by its seat.
  'focus/jump-0': Array.from({ length: 30 }, (_, at) => ({
    id: `${at}`,
    text: `Section ${at + 1}`,
    level: 1,
  })),
}

export const PartsInside: Story = {
  args: {
    ...invented.args,
    parents: 1,
    children: 4,
    jumps: 1,
    siblings: 0,
    parts: (id: string) => INSIDE[id] ?? [],
  },
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)
    const partsOf = (node: Element) => [...node.querySelectorAll('.plex__part')]

    /** How many stand in the window at once, which the knob says. */
    const most = args.maxParts

    // Nothing hangs until the hand has been on it a while.
    await expect(partsOf(focus)).toHaveLength(0)

    await userEvent.hover(focus)
    await waitFor(async () => await expect(partsOf(focus)).toHaveLength(most), { timeout: 3000 })

    // The nesting is drawn by setting a part in, and the deeper of them stands
    // further in than the one it sits under.
    const setIn = (part: Element) => Number.parseFloat(getComputedStyle(part).paddingInlineStart)
    const [first, second, third] = partsOf(focus)
    await expect(setIn(second!)).toBeGreaterThan(setIn(first!))
    await expect(setIn(third!)).toBeGreaterThan(setIn(second!))

    // The hand leaves, and they go back under the box.
    await userEvent.unhover(focus)
    await waitFor(async () => await expect(partsOf(focus)).toHaveLength(0))

    // Choosing one says which part it was, on the node it hangs from.
    await userEvent.hover(focus)
    await waitFor(async () => await expect(partsOf(focus)).toHaveLength(most), { timeout: 3000 })
    await userEvent.click(partsOf(focus)[1]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus', '4')

    // A note with more parts than stand at once is scrolled through them.
    const many = canvas.getByLabelText(/, jump$/)
    await userEvent.hover(many)
    await waitFor(async () => await expect(partsOf(many)).toHaveLength(most), { timeout: 3000 })

    // A window on more than it holds carries an arrow at the edge it may be
    // scrolled towards, and scrolling it moves it by whole parts.
    const arrows = () => [...many.querySelectorAll('.plex__more')]
    await expect(arrows()).toHaveLength(1)

    // Said in lines, which is one part the line. A hand on a trackpad speaks
    // in pixels and scrolls when they come to a part's height.
    const wheel = (deltaY: number) =>
      many
        .querySelector('.plex__inside')!
        .dispatchEvent(
          new WheelEvent('wheel', { deltaY, deltaMode: 1, bubbles: true, cancelable: true }),
        )
    wheel(1)
    await waitFor(async () => await expect(partsOf(many)[0]!.textContent?.trim()).toBe('Section 2'))
    await expect(arrows()).toHaveLength(2)

    wheel(-1)
    await waitFor(async () => await expect(partsOf(many)[0]!.textContent?.trim()).toBe('Section 1'))

    // Choosing a part is not choosing the node it hangs from: the plex stays
    // where it is standing.
    await userEvent.click(partsOf(many)[0]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus/jump-0', '0')
    await expect(args.onActivate).not.toHaveBeenCalled()
  },
}

/** Too small a window: the plex is clipped rather than shrunk. */
export const SmallWindow: Story = {
  args: { neighbourhood: neighbourhoods.crowded },
  render: (args) => ({
    components: { Plex },
    setup: () => {
      const type = useTypeSize()
      return { args, options: computed(() => optionsFrom(args, type.value)) }
    },
    template: `
      <div style="height:100vh;display:grid;place-items:center;background:#8883">
        <div style="width:420px;height:320px;outline:1px solid var(--numen-rule)">
          <Plex
            :neighbourhood="args.neighbourhood"
            :placement="args.placement"
            :options="options"
            :show-edge-labels="args.showEdgeLabels"
            :duration="args.duration"
            :dwell="args.dwell"
            @activate="args.onActivate"
          />
        </div>
      </div>
    `,
  }),
}
