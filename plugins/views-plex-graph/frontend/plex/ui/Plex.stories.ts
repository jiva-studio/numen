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
import { optionsForType, useTypeSize } from '../model/sizing/plexScale'
import { EXPANSION_DELAY } from '../model/sizing/expansion'
import { type PlexHeading } from '../lib/inside'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import { createNeighbourhood, walkStart } from '../fixtures/walk'
import { build, buildAround, type TitledNode } from '../fixtures/build'
import { nameNow } from '../fixtures/names'
import { ring } from '../fixtures/ring'
import type { PlexEdge } from '../lib/edge'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlexNode, Position } from '../lib/node'
import type { PlexRelation } from '../lib/relation'
import type { PlexDestination } from '../model/interaction/destination'
import type { Clock } from '../model/transition'
import type { MenuTrigger } from '@numen/ui'

interface Knobs {
  neighbourhood: PlexNeighbourhood
  placement: Placement
  showEdgeLabels: boolean
  duration: number
  expansionDelay: number
  onActivate: (id: string) => void
  onShow: (id: string, showing: PlexDestination) => void
  onCreate: (from: string, relation: PlexRelation) => void
  onLink: (from: string, to: string, relation: PlexRelation) => void
  onBring: (dragged: readonly string[], relation: PlexRelation) => void
  onMenu: (id: string, point: Position, trigger: MenuTrigger) => void
  onDismiss: () => void
  headings: (id: string) => readonly PlexHeading[]
  onEnter: (id: string, heading: string) => void

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
  maxHeadings: number
  orientation: 'parents above' | 'parents below'

  curvature: number
  minReach: number
  arrowClearance: number

  arriveAfter: number
  leaveBefore: number

  /** Playground only: how many of each relation to invent. */
  parents: number
  children: number
  jumps: number
  siblings: number
  title: string

  /**
   * What a node made by a gesture is called. Handed in and not settled
   * here, because naming is the application's business and the plex never
   * learns that a title is a name.
   */
  naming: () => string

  /**
   * Identifiers of the opaque items dragged over the plex from outside it;
   * empty when nothing is dragged. The story plays the drag source, since a
   * plex cannot start a drag.
   */
  dragged: readonly string[]

  /** The label for the relation under the pointer while an item is dragged in. */
  dropName: (relation: PlexRelation) => string

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
  maxHeadings: a.maxHeadings,
  routing: { curvature: a.curvature, minReach: a.minReach, arrowClearance: a.arrowClearance },
  motion: { arriveAfter: a.arriveAfter, leaveBefore: a.leaveBefore },
  direction:
    a.orientation === 'parents below'
      ? { parent: 'down', child: 'up', jump: 'left', sibling: 'right' }
      : { parent: 'up', child: 'down', jump: 'left', sibling: 'right' },
})

/**
 * The knobs at the type size of a node. The toolbar sets the interface size,
 * the label follows it, and the box sizes and clearances are passed in to fit
 * that label.
 */
const createOptions = (a: Knobs, type: number): PlexOptionsInput =>
  optionsForType(type, resolveOptions(getOptions(a)))

const range = (min: number, max: number, step = 1, category = 'Layout') => ({
  control: { type: 'range' as const, min, max, step },
  table: { category },
})

const createCounts = (a: Knobs) => ({
  parent: a.parents,
  child: a.children,
  jump: a.jumps,
  sibling: a.siblings,
})

/**
 * Every story is walkable: choosing a node builds the next neighbourhood
 * around it. The first neighbourhood is the story's starting point; later ones
 * are generated from the counts, standing in for the application.
 *
 * The neighbourhood is computed: a new object on every render signals the plex
 * to re-target, and it re-aims once per frame.
 */
const createNavigable = (start: (args: Knobs) => PlexNeighbourhood) => (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focus = ref<TitledNode | null>(null)
    const cameFrom = ref<TitledNode | null>(null)

    /** What the application would keep: what has been made, and what is new. */
    const createdItem = ref<PlexNode[]>([])
    const links = ref<PlexEdge[]>([])

    const neighbourhood = computed<PlexNeighbourhood>(() => {
      const base = focus.value
        ? buildAround(focus.value, cameFrom.value, createCounts(args))
        : start(args)
      const here = focus.value?.id ?? base.nodes.find((n) => n.relation === 'focus')?.id
      const mine = createdItem.value.filter((node) =>
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
      const selected = neighbourhood.value.nodes.find((node) => node.id === id)
      if (!selected) return
      const was = neighbourhood.value.nodes.find((node) => node.relation === 'focus')
      cameFrom.value = was ? { id: was.id, title: was.title } : null
      focus.value = { id: selected.id, title: selected.title }
    }

    /** A node related beside another one, linked the way that relation is written. */
    const createRelated = (from: string, title: string, relation: PlexRelation) => {
      const id = `made/${createdItem.value.length}`
      createdItem.value = [...createdItem.value, { id, title, relation }]
      links.value = [
        ...links.value,
        relation === 'parent' || relation === 'jump'
          ? { from: id, to: from, label: relation === 'jump' ? 'see also' : 'is a' }
          : { from, to: id, label: 'contains' },
      ]
    }

    /**
     * A new node arrives complete: related, linked and already named. The
     * gesture ends at the pointer release; the final name is chosen later by the
     * application.
     */
    const create = (from: string, relation: PlexRelation) => {
      createRelated(from, args.naming(), relation)
    }

    /**
     * What was dragged in from outside, each related beside the focus in the
     * one relation and called by the identifier it was dragged in as. What those
     * identifiers address is the story's to know, and the plex handed them
     * back untouched.
     */
    const dropOnto = (ids: readonly string[], relation: PlexRelation) => {
      const here = neighbourhood.value.nodes.find((node) => node.relation === 'focus')
      if (!here) return
      for (const one of ids) createRelated(here.id, one, relation)
    }

    const link = (from: string, to: string, relation: PlexRelation) => {
      links.value = [
        ...links.value,
        relation === 'parent' || relation === 'jump' ? { from: to, to: from } : { from, to },
      ]
    }

    return {
      args,
      onActivate,
      create,
      link,
      dropOnto,
      neighbourhood,
      options: computed(() => createOptions(args, type.value)),
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
        :expansion-delay="args.expansionDelay"
        :dragged="args.dragged"
        :drop-name="args.dropName"
        :headings="args.headings"
        @enter="(id, heading) => args.onEnter(id, heading)"
        @activate="onActivate($event); args.onActivate($event)"
        @show="(id, showing) => args.onShow(id, showing)"
        @create="(from, relation) => { create(from, relation); args.onCreate(from, relation) }"
        @link="(from, to, relation) => { link(from, to, relation); args.onLink(from, to, relation) }"
        @bring="(dragged, relation) => { dropOnto(dragged, relation); args.onBring(dragged, relation) }"
        @menu="args.onMenu"
        @dismiss="args.onDismiss"
      />
    </div>
  `,
})

/**
 * Choosing a node moves the focus, which is what the story verifies.
 *
 * The neighbourhood is computed: a new object on every render signals the plex
 * to re-target, and it re-aims once per frame.
 */
const renderWalk = (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focused = ref(walkStart)
    return {
      args,
      focused,
      neighbourhood: computed(() => createNeighbourhood(focused.value)),
      options: computed(() => createOptions(args, type.value)),
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
        :expansion-delay="args.expansionDelay"
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
          'One node in focus, everything else placed by its relation: parents ' +
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
    maxHeadings: range(1, 12),
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
    arrowClearance: { ...range(0, 40, 1, 'Edges'), table: { category: 'Edges' } },
    showEdgeLabels: { control: 'boolean', table: { category: 'Edges' } },

    duration: {
      control: { type: 'range', min: 0, max: 3000, step: 20 },
      table: { category: 'Motion' },
    },
    expansionDelay: {
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
    headings: { table: { disable: true } },
    onEnter: { table: { disable: true } },
  },

  args: {
    placement: rowsAndColumns,
    showEdgeLabels: true,
    duration: 420,
    expansionDelay: EXPANSION_DELAY,
    onActivate: fn(),
    onShow: fn(),
    onCreate: fn(),
    onLink: fn(),
    onBring: fn(),
    onMenu: fn(),
    onDismiss: fn(),
    headings: () => [],
    onEnter: fn(),
    naming: nameNow,
    dragged: [],
    dropName: (relation: PlexRelation) => `as ${relation}`,

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
    maxHeadings: 6,
    orientation: 'parents above',

    curvature: 0.55,
    minReach: 22,
    arrowClearance: 14,
    arriveAfter: 0.35,
    leaveBefore: 0.45,

    parents: 2,
    children: 6,
    jumps: 3,
    siblings: 2,
    title: 'A node',
  },

  render: createNavigable((args) => args.neighbourhood),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

/**
 * Every setting, live, and a plex to walk while turning them. The counts
 * invent the neighbourhood; choosing a node builds the next one around it.
 */
const invented = {
  args: { neighbourhood: neighbourhoods.typical, maxPerLine: 9 },
  render: createNavigable((args: Knobs) => build(args.title, createCounts(args))),
}

export const Playground: Story = invented

/**
 * Dragging out of a node and the result.
 *
 * A dedicated story, not a check on the playground: a play function runs when
 * a story opens, and a plex that has already created a node is not the state
 * under test.
 *
 * It passes its own naming, which the check reads and which shows the contract:
 * the plex reports the gesture and the caller names the node.
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

    // Labels on a connection are not text-selectable during the gesture, since
    // the pointer passes over them. Read under the WebKit property spelling.
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

    // Release the pointer by dispatching the event: a second `userEvent.pointer`
    // call does not track the button state of the first, so its release is a no-op.
    surface.dispatchEvent(new PointerEvent('pointerup', { ...above, pointerId: 1, bubbles: true }))
    await expect(args.onCreate).toHaveBeenCalledWith(expect.any(String), 'parent')

    // Verifies that releasing a captured pointer elsewhere notifies the node it
    // was captured from.
    await expect(canvasElement.querySelector('.plex__handle')).toBeNull()

    // The story relates the node and applies the name returned by its own
    // naming. No prompt is shown and no text is typed.
    await waitFor(async () => {
      await expect(canvas.getByLabelText('Reached for, parent')).toBeInTheDocument()
    })
  },
}

/**
 * Items dragged over the plex from outside it.
 *
 * The gesture starts outside the plex, so the plex receives only a list of
 * identifiers and returns a relation, measured from the focus. Several items
 * form one line and one relation; the drag source provides the count label,
 * drawn at the pointer.
 *
 * The line and the label are drawn while the pointer moves, so the outcome of a
 * drop is visible before it happens. The story stops mid-drag.
 *
 * Only a browser can verify it: a real matrix, a pointer the plex never
 * captured, and a drawing that follows the pointer across.
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

    // Return over the plex and stay there for inspection.
    dragTo(middle, box.y - 220)
    await expectDraggedTitle('as parent')
  },
}

/** Whether a focus reads as the keyboard's. Browsers take it; the types do not. */
interface KeyboardFocus extends FocusOptions {
  focusVisible?: boolean
}

/**
 * Handle placement and the one-handle limit.
 *
 * A handle belongs to the box under the pointer and to the box with visible
 * keyboard focus. A click leaves a box with a focus that is neither, so a box
 * that was clicked and then released shows no handle.
 *
 * Only a browser can verify it: a click and a tab leave the same focus, and
 * which of them counts as visible keyboard focus is decided by the browser. A
 * pointer driven from a story is untrusted and does not trigger that state.
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

    // A plex with no pointer or keyboard focus on a node shows no handle.
    await expect(getHandles()).toHaveLength(0)

    // The focus left by a click is not visible keyboard focus, and the pointer
    // is not over the box.
    first.focus({ focusVisible: false } as KeyboardFocus)
    await expect(first).toHaveFocus()
    await expect(getHandles()).toHaveLength(0)

    // Hover each box in turn; exactly one handle exists at a time.
    for (const box of boxes) {
      await userEvent.hover(box)
      await expect(getHandles()).toHaveLength(1)
      await expect(box.querySelector('.plex__handle')).not.toBeNull()
      await userEvent.unhover(box)
      await expect(getHandles()).toHaveLength(0)
    }

    // Arriving by tab is keyboard use, and the box it stops at shows a handle.
    first.blur()
    await userEvent.tab()
    const stop = canvasElement.ownerDocument.activeElement!
    await expect(stop).toHaveClass('plex__node')
    await expect(getHandles()).toHaveLength(1)
    await expect(stop.querySelector('.plex__handle')).not.toBeNull()
  },
}

/**
 * Requesting a context menu on a node.
 *
 * The plex reports which node was targeted, the position, and the input
 * source. The menu content and the item actions are outside the plex.
 *
 * Only a browser can verify it: whether a `contextmenu` event reaches an SVG
 * group, whether preventing it suppresses the webview's own menu, and whether
 * the right button over the handle targets the node under it and not the
 * handle.
 */
export const AskingForAMenu: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    // Read after the node has had it: a listener above sees what the node did.
    let isPrevented: boolean | null = null
    const watch = (event: Event) => {
      isPrevented = event.defaultPrevented
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
    await expect(isPrevented).toBe(true)

    // The right button over the handle targets the node and creates nothing.
    await userEvent.hover(focus)
    const handle = canvasElement.querySelector('.plex__handle')!
    await userEvent.pointer({ keys: '[MouseRight]', target: handle })
    await expect(args.onCreate).not.toHaveBeenCalled()
    await expect(canvasElement.querySelector('.plex__thread')).toBeNull()
    await expect(args.onMenu).toHaveBeenCalledTimes(2)
  },
}

/**
 * Requesting a node itself without navigating to it.
 *
 * Triggered on the focus, the one node where a click does not move the focus.
 *
 * Only a browser can verify it: whether a double click reaches an SVG group,
 * and whether a held modifier arrives with it.
 */
export const AskingForANode: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    // One user session throughout, so a modifier held down is still held at the
    // later pointer event.
    const hand = userEvent.setup()

    await hand.dblClick(focus)
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'current')
    await expect(args.onActivate).not.toHaveBeenCalled()

    // The same gesture with the modifier held requests the node next to the
    // reader's position. Its meaning belongs to the application and never
    // reaches the plex.
    await hand.keyboard('{Alt>}')
    await hand.dblClick(focus)
    await hand.keyboard('{/Alt}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'split')

    // The keyboard route: a press with Shift, and the same modifier for where.
    focus.focus()
    await hand.keyboard('{Shift>}{Enter}{/Shift}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'current')
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
    { id: 'focus', title: 'The meeting', relation: 'focus' },
    { id: 'parent', title: 'Minutes', relation: 'parent' },
    { id: 'earlier', title: 'The morning', relation: 'jump' },
    { id: 'across', title: 'Elsewhere', relation: 'jump' },
    { id: 'later', title: 'The evening', relation: 'jump' },
    { id: 'sibling', title: 'The hedge', relation: 'sibling' },
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
 * How a line renders the title it carries: the title is set along the line and
 * truncated where the words are longer.
 *
 * Only a browser can verify it: the width of the words in the chosen type, the
 * length of the curve they are set on, and whether the two paint layers of a
 * title coincide.
 *
 * The layout is complete on load: a title belongs to a line at rest, and a line
 * in motion cannot be targeted.
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
    const getTextLine = (text: SVGTextElement) => {
      const href = text.querySelector('textPath')!.getAttribute('href')!
      return canvasElement.querySelector<SVGPathElement>(`defs path[id="${href.slice(1)}"]`)!
    }

    // The window is measured after the first drawing, so the plex settles on
    // its second.
    await waitFor(async () => {
      await expect(title('contains')).toHaveLength(2)
    })

    // Every title is set along the line it belongs to, and none of them is
    const titles = [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')]
    await expect(titles.length).toBeGreaterThan(0)
    for (const text of titles) {
      await expect(text.querySelector('textPath')).not.toBeNull()
      await expect(text.getComputedTextLength()).toBeLessThanOrEqual(
        getTextLine(text).getTotalLength(),
      )
    }

    // Words longer than their curve are cut to it and end in an ellipsis.
    await expect(title('the minutes of the meeting')).toHaveLength(0)
    const truncatedTitles = titles.filter((text) => text.textContent!.endsWith('…'))
    await expect(truncatedTitles).toHaveLength(2)
    await expect(
      'the minutes of the meeting'.startsWith(truncatedTitles[0]!.textContent!.slice(0, -1)),
    ).toBe(true)

    // A title is painted in two layers that land on one another.
    const [halo, letters] = title('contains')
    await expect(halo!.querySelector('textPath')).not.toBeNull()
    await expect(getComputedStyle(halo!).fill).toBe('none')
    await expect(getComputedStyle(letters!).stroke).toBe('none')
    await expect(halo!.getComputedTextLength()).toBe(letters!.getComputedTextLength())
    const haloAtRest = Number.parseFloat(getComputedStyle(halo!).strokeWidth)

    // Hovering the line raises it, and the raised title is painted the same way
    // on a heavier halo.
    const axisHref = halo!.querySelector('textPath')!.getAttribute('href')!
    const viewElement = canvasElement
      .querySelector(`defs path[id="${axisHref.slice(1)}"]`)!
      .getAttribute('d')
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === viewElement,
    )
    await userEvent.hover(band!)

    const lifted = canvasElement.querySelectorAll<SVGTextElement>('.plex__lift .plex__edge-label')
    await expect(lifted).toHaveLength(2)
    await expect(getComputedStyle(lifted[0]!).fill).toBe('none')
    await expect(getComputedStyle(lifted[1]!).stroke).toBe('none')
    await expect(Number.parseFloat(getComputedStyle(lifted[0]!).strokeWidth)).toBeGreaterThan(
      haloAtRest,
    )
    await expect(lifted[0]!.getComputedTextLength()).toBe(lifted[1]!.getComputedTextLength())
    await userEvent.unhover(band!)

    // A jump that leaves its column and comes round is set along its line like
    // any other, and its words are short enough to stay whole.
    const loop = title('see also')[0]!
    await expect(loop.querySelector('textPath')).not.toBeNull()
    await expect(loop.getComputedTextLength()).toBeLessThan(getTextLine(loop).getTotalLength())
  },
}

interface Edges {
  left: number
  right: number
  top: number
  bottom: number
}

/**
 * No drawn title overlaps a box or another title: none of its letters lies over
 * a box, and none lies over another title's letters.
 *
 * Two counts are asserted directly, so a regression reports its magnitude.
 */
const expectTitlesClear = async ({ canvasElement }: { canvasElement: HTMLElement }) => {
  /** Whether two boxes on the screen share any of it. */
  const isOverlapping = (one: Edges, other: Edges) =>
    one.left < other.right &&
    other.left < one.right &&
    one.top < other.bottom &&
    other.top < one.bottom

  /**
   * The screen rectangle of each letter of a title. A title set along a curve
   * is a ribbon of letters: the box around a diagonal title covers an area
   * that the letters do not occupy.
   */
  const getLetters = (text: SVGTextElement): Edges[] => {
    const onScreen = text.getScreenCTM()!
    const letters: Edges[] = []

    for (let charIndex = 0; charIndex < text.getNumberOfChars(); charIndex += 1) {
      const letter = text.getExtentOfChar(charIndex)
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

  const renderedTitles = titles().map((text) => ({
    words: text.textContent,
    letters: getLetters(text),
  }))
  const boxes = [...canvasElement.querySelectorAll<SVGRectElement>('.plex__box')].map((box) =>
    box.getBoundingClientRect(),
  )

  const piled: string[] = []
  for (const [at, title] of renderedTitles.entries()) {
    for (const other of renderedTitles.slice(at + 1)) {
      const touching = title.letters.some((letter) =>
        other.letters.some((mine) => isOverlapping(letter, mine)),
      )
      if (touching) piled.push(`${title.words} × ${other.words}`)
    }
  }

  const overBoxes = renderedTitles
    .filter((title) =>
      title.letters.some((letter) => boxes.some((box) => isOverlapping(letter, box))),
    )
    .map((title) => title.words)

  await expect(renderedTitles.length).toBeGreaterThan(0)
  await expect(piled).toHaveLength(0)
  await expect(overBoxes).toHaveLength(0)
}

/**
 * Overlap between titles. Uses the playground's neighbourhood, which is dense:
 * a fan of lines out of every node, many with the same word.
 *
 * Two counts are asserted directly, so a regression reports its magnitude. Only
 * a browser can verify either: the width of the words in the chosen type, and
 * the glyph positions after they are bent along their curves.
 *
 * It uses its own neighbourhood, since the playground exists for adjusting
 * knobs.
 */
export const TitlesFindSpace: Story = {
  args: { ...invented.args, duration: 0 },
  render: invented.render,
  play: expectTitlesClear,
}

/**
 * The lines of one note, every one of them named, and more children than a row
 * holds.
 *
 * The lines to the far row cross the near row, so their titles compete for
 * space in a band that the near row's titles already occupy. A title without a
 * clear span is truncated to the space available, and a title with space for
 * less than half of its words is not drawn.
 */
export const TitlesAcrossRows: Story = {
  args: { neighbourhood: neighbourhoods.labelledRows, duration: 0 },
  play: expectTitlesClear,
}

/**
 * One note's lines: two lines with an arrow, one without, and a sibling's line
 * attached to the parent that the two share.
 */
const arrowedLines: PlexNeighbourhood = {
  nodes: [
    { id: 'alice', title: 'Alice Fenn, The Chair', relation: 'focus' },
    { id: 'allotments', title: 'Marrowfield allotments', relation: 'parent' },
    { id: 'bram', title: 'Bram Doyle', relation: 'jump' },
    { id: 'cora', title: 'Cora Hale', relation: 'jump' },
    { id: 'rota', title: 'The question of the rota', relation: 'sibling' },
  ],
  edges: [
    { from: 'allotments', to: 'alice', arrow: 'from' },
    { from: 'bram', to: 'alice', label: 'the oldest tenant', arrow: 'from' },
    { from: 'cora', to: 'alice', label: 'keys, hoses, the gate' },
    { from: 'allotments', to: 'rota', label: 'the minutes of the meeting' },
  ],
}

/**
 * The arrow a line carries, drawn at either of the two ends it was given.
 *
 * Only a browser can verify it: whether the style produces a head, where the
 * head lands once its rotation is applied, and which side of the box it is on.
 *
 * The layout is complete on load: a line in motion has not reached the end its
 * arrow belongs at.
 */
export const ArrowedLines: Story = {
  args: { neighbourhood: arrowedLines, duration: 0 },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const svg = canvasElement.querySelector('svg')!

    /** Whether a box holds a point, give or take a pixel. */
    const hasPoint = (box: DOMRect, point: DOMPoint) =>
      point.x >= box.left - 1 &&
      point.x <= box.right + 1 &&
      point.y >= box.top - 1 &&
      point.y <= box.bottom + 1

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
    const getLine = (name: string) => {
      const box = canvas.getByLabelText(name).getBoundingClientRect()
      for (const { line, ends } of lines()) {
        const end = ends.find((point) => hasPoint(box, point))
        if (end) return { line, end }
      }
      throw new Error(`no line touches ${name}`)
    }

    const getEnd = (name: string) => getLine(name).end

    const heads = () => [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-arrow')]

    /** Whether an arrowhead was drawn on a given end of a line. */
    const getHead = (end: DOMPoint) =>
      heads().some((head) => hasPoint(head.getBoundingClientRect(), end))

    /** The positions of every head and every line end, as one snapshot. */
    const readView = () =>
      JSON.stringify([
        heads().map((head) => {
          const box = head.getBoundingClientRect()
          return [box.x, box.y, box.width, box.height]
        }),
        lines().map(({ ends }) => ends.map((end) => [end.x, end.y])),
      ])

    /** A frame, after which what was arranged on the last one can be read. */
    const frame = () => new Promise((isDone) => requestAnimationFrame(() => isDone(null)))

    // The viewport is measured after the first drawing, so the plex settles on
    // its second drawing and the layout is stable afterwards.
    await waitFor(async () => {
      await expect(heads()).toHaveLength(2)
      const was = readView()
      await frame()
      await expect(readView()).toBe(was)
    })

    // A head is drawn and is not an empty path.
    for (const head of heads()) {
      const box = head.getBoundingClientRect()
      await expect(box.width).toBeGreaterThan(0)
      await expect(box.height).toBeGreaterThan(0)
    }

    // Each of them at the far end of its own line, away from the focus.
    await expect(getHead(getEnd('Marrowfield allotments, parent'))).toBe(true)
    await expect(getHead(getEnd('Bram Doyle, jump'))).toBe(true)

    // The lines given none carry none, the sibling's line among them.
    await expect(getHead(getEnd('Cora Hale, jump'))).toBe(false)
    await expect(getHead(getEnd('The question of the rota, sibling'))).toBe(false)

    // The head lies outside the box it points into: it is rotated along the
    // line, and its tip touches the border.
    const jump = getLine('Bram Doyle, jump')
    const box = canvas.getByLabelText('Bram Doyle, jump').getBoundingClientRect()
    const head = heads().find((one) => hasPoint(one.getBoundingClientRect(), jump.end))!
    const bounds = head.getBoundingClientRect()
    await expect(
      hasPoint(box, new DOMPoint(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2)),
    ).toBe(false)

    // The title on that line is cut short of the head, at either end of the
    // words: neither the letters nor the halo under them reach it.
    const clear = Math.max(bounds.width, bounds.height)
    const title = [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')].find(
      (text) => text.textContent!.startsWith('the oldest'),
    )!
    const letters = title.getNumberOfChars()
    const onScreen = svg.getScreenCTM()!
    for (const glyph of [
      title.getStartPositionOfChar(0),
      title.getEndPositionOfChar(letters - 1),
    ]) {
      const point = new DOMPoint(glyph.x, glyph.y).matrixTransform(onScreen)
      await expect(Math.hypot(point.x - jump.end.x, point.y - jump.end.y)).toBeGreaterThan(clear)
    }

    // Hovering the line raises it, and the head is brightened with it.
    const restingLightness = getComputedStyle(head).fill
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === jump.line.getAttribute('d'),
    )!
    await userEvent.hover(band)

    const lifted = canvasElement.querySelector<SVGPathElement>('.plex__lift .plex__edge-arrow')!
    await expect(lifted).not.toBeNull()
    await expect(getComputedStyle(lifted).fill).not.toBe(restingLightness)
  },
}

/**
 * Not Latin, far too long, nothing to break at, and nothing at all — the whole
 * arrangement made of them, to see that text nothing fits into still leaves a
 * plex and not a pile.
 *
 * Whether a title stays inside its own box is the node's affair and is checked
 * on the node, against the same corpus.
 */
export const AwkwardLabels: Story = {
  args: { neighbourhood: neighbourhoods.awkwardLabels },
}

/**
 * A title too long for its box, read by hovering the box.
 *
 * Only a browser can verify it: the width of a title is the evaluated width in
 * the chosen type, whether the box contains it is decided by the browser, and
 * overlap with neighbouring boxes depends on paint order.
 */
export const RestingOnATitle: Story = {
  args: { neighbourhood: neighbourhoods.awkwardLabels },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText(/^Supercalifragilistic/)
    const adjacentNode = canvas.getByLabelText('日本語のノート, child')

    const getWidth = (box: Element) =>
      box.querySelector('.plex__box')!.getBoundingClientRect().width
    const nodeWidth = getWidth(node)
    const adjacentX = adjacentNode.getBoundingClientRect().x

    // Truncated in the box the arrangement drew, and shown in full while the
    // pointer stays on it.
    const words = node.querySelector<HTMLElement>('.plex__title-text')!
    await expect(words.scrollWidth).toBeGreaterThan(words.clientWidth)

    await userEvent.hover(node)
    await waitFor(async () => await expect(getWidth(node)).toBeGreaterThan(nodeWidth), {
      timeout: 3000,
    })
    await waitFor(
      async () => await expect(words.scrollWidth).toBeLessThanOrEqual(words.clientWidth),
    )

    await expect(adjacentNode.getBoundingClientRect().x).toBe(adjacentX)
    const boxes = canvasElement.querySelectorAll('.plex__node')
    await expect(boxes[boxes.length - 1]).toBe(node)

    // After the pointer leaves, the title returns to its original state.
    await userEvent.unhover(node)
    await waitFor(async () => await expect(getWidth(node)).toBe(nodeWidth))
  },
}

/**
 * Walk the graph. Go back the way you came: nodes in both pictures travel
 * and not blinking out and in, because they are matched by identifier.
 */
export const Walk: Story = {
  render: renderWalk,
  args: {
    neighbourhood: createNeighbourhood(walkStart),
    gap: 18,
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const middle = canvasElement.getBoundingClientRect()

    /**
     * Measured on the box, not on the surrounding group: a node that was just
     * clicked holds the focus and shows a handle at its trailing edge, which
     * would add to the group's width.
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

    // The keyboard, in a real browser: whether tab stops on an SVG group. The
    // focus is also a tab stop although it cannot be chosen; a menu is requested
    // from the keyboard position.
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveAttribute(
      'tabindex',
      '0',
    )
    await userEvent.tab()
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveFocus()

    // Past the handle the node under the keyboard now offers, onto the relation
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

    // The node walked away from took the relation above, and the plex is a plex
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
}

/**
 * Everything hung headings have to survive, one node each: a nested set, far more
 * than fit, one alone, one that runs on past the box, headings not written in
 * Latin, a heading with no words at all, and nodes with none.
 */
const INSIDE: Record<string, readonly PlexHeading[]> = {
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
  // The one node there is exactly one of, so the play can find it by its relation.
  'focus/jump-0': Array.from({ length: 30 }, (_, index) => ({
    id: `${index}`,
    text: `Section ${index + 1}`,
    level: 1,
  })),
}

export const HeadingsInside: Story = {
  args: {
    ...invented.args,
    parents: 1,
    children: 4,
    jumps: 1,
    siblings: 0,
    headings: (id: string) => INSIDE[id] ?? [],
  },
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)
    const getHeadings = (node: Element) => [...node.querySelectorAll('.plex__heading')]

    /** The maximum number of headings shown at once, set by the knob. */
    const most = args.maxHeadings

    // No headings appear until the pointer has stayed on the node for a while.
    await expect(getHeadings(focus)).toHaveLength(0)

    await userEvent.hover(focus)
    await waitFor(async () => await expect(getHeadings(focus)).toHaveLength(most), {
      timeout: 3000,
    })

    // Nesting is drawn by indenting a heading, so a deeper heading is indented
    // further than its parent.
    const getIndent = (heading: Element) =>
      Number.parseFloat(getComputedStyle(heading).paddingInlineStart)
    const [first, second, third] = getHeadings(focus)
    await expect(getIndent(second!)).toBeGreaterThan(getIndent(first!))
    await expect(getIndent(third!)).toBeGreaterThan(getIndent(second!))

    // After the pointer leaves, the headings are hidden again.
    await userEvent.unhover(focus)
    await waitFor(async () => await expect(getHeadings(focus)).toHaveLength(0))

    // Choosing a heading reports which heading it was, on the node it belongs to.
    await userEvent.hover(focus)
    await waitFor(async () => await expect(getHeadings(focus)).toHaveLength(most), {
      timeout: 3000,
    })
    await userEvent.click(getHeadings(focus)[1]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus', '4')

    // A note with more headings than stand at once is scrolled through them.
    const many = canvas.getByLabelText(/, jump$/)
    await userEvent.hover(many)
    await waitFor(async () => await expect(getHeadings(many)).toHaveLength(most), { timeout: 3000 })

    // A window on more than it holds carries an arrow at the edge it may be
    // scrolled towards, and scrolling it moves it by whole headings.
    const arrows = () => [...many.querySelectorAll('.plex__more')]
    await expect(arrows()).toHaveLength(1)

    // in pixels and scrolls when they come to a heading's height.
    const wheel = (deltaY: number) =>
      many
        .querySelector('.plex__inside')!
        .dispatchEvent(
          new WheelEvent('wheel', { deltaY, deltaMode: 1, bubbles: true, cancelable: true }),
        )
    wheel(1)
    await waitFor(
      async () => await expect(getHeadings(many)[0]!.textContent?.trim()).toBe('Section 2'),
    )
    await expect(arrows()).toHaveLength(2)

    wheel(-1)
    await waitFor(
      async () => await expect(getHeadings(many)[0]!.textContent?.trim()).toBe('Section 1'),
    )

    // Choosing a heading is not choosing the node it hangs from: the plex stays
    // where it is standing.
    await userEvent.click(getHeadings(many)[0]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus/jump-0', '0')
    await expect(args.onActivate).not.toHaveBeenCalled()
  },
}

/** Too small a window: the plex is clipped and not shrunk. */
export const SmallWindow: Story = {
  args: { neighbourhood: neighbourhoods.crowded },
  render: (args) => ({
    components: { Plex },
    setup: () => {
      const type = useTypeSize()
      return { args, options: computed(() => createOptions(args, type.value)) }
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
            :expansion-delay="args.expansionDelay"
            @activate="args.onActivate"
          />
        </div>
      </div>
    `,
  }),
}
