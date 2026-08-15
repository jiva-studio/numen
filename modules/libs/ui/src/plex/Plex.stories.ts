/**
 * Every situation the plex has to survive. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`.
 *
 * The knobs are flat rather than one `options` object, because a JSON editor
 * is not a control. Everything a reader might want to turn is a slider, a
 * toggle or a select; the object is assembled here.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref } from 'vue'
import Plex from './Plex.vue'
import { rowsAndColumns, type Placement, type PlexOptionsInput } from './arrange'
import { neighbourhoods } from './fixtures/neighbourhoods'
import { neighbourhoodOf, walkStart } from './fixtures/walk'
import { around, build, type Thought } from './fixtures/build'
import { ring } from './fixtures/ring'
import type { PlexNeighbourhood } from './model'
import type { Environment } from './transition'

interface Knobs {
  neighbourhood: PlexNeighbourhood
  placement: Placement
  showEdgeLabels: boolean
  duration: number
  onActivate: (id: string) => void

  focusWidth: number
  focusHeight: number
  nodeWidth: number
  nodeHeight: number
  gap: number
  lineGap: number
  focusGap: number
  margin: number
  maxPerLine: number
  maxLines: number
  orientation: 'parents above' | 'parents below'

  curvature: number
  minReach: number

  arriveAfter: number
  leaveBefore: number

  /** Playground only: how many of each role to invent. */
  parents: number
  children: number
  jumps: number
  siblings: number
  label: string

  /** Component props the panel has no business showing. */
  options?: PlexOptionsInput
  environment?: Environment
}

const optionsFrom = (a: Knobs): PlexOptionsInput => ({
  focusSize: { width: a.focusWidth, height: a.focusHeight },
  nodeSize: { width: a.nodeWidth, height: a.nodeHeight },
  gap: a.gap,
  lineGap: a.lineGap,
  focusGap: a.focusGap,
  margin: a.margin,
  maxPerLine: a.maxPerLine,
  maxLines: a.maxLines,
  routing: { curvature: a.curvature, minReach: a.minReach },
  motion: { arriveAfter: a.arriveAfter, leaveBefore: a.leaveBefore },
  direction:
    a.orientation === 'parents below'
      ? { parent: 'down', child: 'up', jump: 'left', sibling: 'right' }
      : { parent: 'up', child: 'down', jump: 'left', sibling: 'right' },
})

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
 * The neighbourhood is computed rather than built in the template: a fresh
 * object on every render tells the plex it has somewhere new to go, and it
 * re-aims once a frame instead of arriving.
 */
const navigable = (start: (args: Knobs) => PlexNeighbourhood) => (args: Knobs) => ({
  components: { Plex },
  setup() {
    const focus = ref<Thought | null>(null)
    const cameFrom = ref<Thought | null>(null)

    const neighbourhood = computed(() =>
      focus.value ? around(focus.value, cameFrom.value, countsFrom(args)) : start(args),
    )

    const chose = (id: string) => {
      const chosen = neighbourhood.value.nodes.find((node) => node.id === id)
      if (!chosen) return
      const was = neighbourhood.value.nodes.find((node) => node.role === 'focus')
      cameFrom.value = was ? { id: was.id, label: was.label } : null
      focus.value = { id: chosen.id, label: chosen.label }
    }

    return { args, chose, neighbourhood, options: computed(() => optionsFrom(args)) }
  },
  template: `
    <div style="height:100vh">
      <Plex
        :neighbourhood="neighbourhood"
        :placement="args.placement"
        :options="options"
        :show-edge-labels="args.showEdgeLabels"
        :duration="args.duration"
        @activate="chose($event); args.onActivate($event)"
      />
    </div>
  `,
})

/**
 * Choosing a node moves the focus, which is what the story is for.
 *
 * The neighbourhood is computed rather than called in the template: a fresh
 * object on every render tells the plex it has somewhere new to go, and it
 * re-aims once a frame instead of arriving.
 */
const walking = (args: Knobs) => ({
  components: { Plex },
  setup() {
    const focused = ref(walkStart)
    return {
      args,
      focused,
      neighbourhood: computed(() => neighbourhoodOf(focused.value)),
      options: computed(() => optionsFrom(args)),
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
        @activate="focused = $event; args.onActivate($event)"
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
          'One node in focus, everything else placed by its role: parents ' +
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
    maxPerLine: range(1, 12),
    maxLines: range(1, 8),
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
    showEdgeLabels: { control: 'boolean', table: { category: 'Edges' } },

    duration: {
      control: { type: 'range', min: 0, max: 3000, step: 20 },
      table: { category: 'Motion' },
    },
    arriveAfter: { ...range(0, 0.95, 0.05, 'Motion'), table: { category: 'Motion' } },
    leaveBefore: { ...range(0.05, 1, 0.05, 'Motion'), table: { category: 'Motion' } },

    parents: range(0, 9, 1, 'Playground'),
    children: range(0, 30, 1, 'Playground'),
    jumps: range(0, 12, 1, 'Playground'),
    siblings: range(0, 12, 1, 'Playground'),
    label: { control: 'text', table: { category: 'Playground' } },

    // Not knobs: the input the caller builds, the clock a test replaces, and
    // what the component reports back.
    neighbourhood: { table: { disable: true } },
    options: { table: { disable: true } },
    environment: { table: { disable: true } },
    onActivate: { table: { disable: true } },
  },

  args: {
    placement: rowsAndColumns,
    showEdgeLabels: true,
    duration: 420,
    onActivate: fn(),

    focusWidth: 176,
    focusHeight: 44,
    nodeWidth: 144,
    nodeHeight: 36,
    gap: 16,
    lineGap: 20,
    focusGap: 56,
    margin: 16,
    maxPerLine: 5,
    maxLines: 4,
    orientation: 'parents above',

    curvature: 0.55,
    minReach: 22,
    arriveAfter: 0.35,
    leaveBefore: 0.45,

    parents: 2,
    children: 6,
    jumps: 3,
    siblings: 2,
    label: 'A thought',
  },

  render: navigable((args) => args.neighbourhood),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

/**
 * Every setting, live, and a plex to walk while turning them. The counts
 * invent the neighbourhood; choosing a node builds the next one around it.
 */
export const Playground: Story = {
  args: { neighbourhood: neighbourhoods.typical, maxPerLine: 9 },
  render: navigable((args) => build(args.label, countsFrom(args))),
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
 * Not Latin, far too long, nothing to break at, and nothing at all.
 *
 * Only a browser can answer this one: jsdom lays out no text, so every label
 * is the same size to it and an overflowing one looks identical to a fitting
 * one.
 */
export const AwkwardLabels: Story = {
  args: { neighbourhood: neighbourhoods.awkwardLabels },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const long = 'Supercalifragilisticexpialidociousandthensomemore'

    const node = canvas.getByLabelText(`${long}, child`)
    const box = node.querySelector('rect')!
    const label = node.querySelector('.plex__label-text')!

    // Clamped rather than spilling: what is drawn is shorter than the text.
    await expect(label.scrollHeight).toBeGreaterThan(label.clientHeight)
    await expect(label.getBoundingClientRect().width).toBeLessThanOrEqual(
      box.getBoundingClientRect().width,
    )

    // Devanagari, Arabic and an empty label all stay inside their boxes.
    for (const name of ['भगवद्गीता, parent', 'الفهرس, parent', 'Untitled, child']) {
      const other = canvas.getByLabelText(name)
      const rect = other.querySelector('rect')!.getBoundingClientRect()
      const text = other.querySelector('.plex__label-text')!.getBoundingClientRect()
      await expect(text.width).toBeLessThanOrEqual(rect.width)
    }
  },
}

/**
 * Walk the graph. Go back the way you came: nodes in both pictures travel
 * rather than blinking out and in, because they are matched by identifier.
 */
export const Walk: Story = {
  args: {
    neighbourhood: neighbourhoodOf(walkStart),
    gap: 18
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const middle = canvasElement.getBoundingClientRect()

    // The keyboard, in a real browser: whether tab actually stops on an SVG
    // group, and that the focus — where the reader already is — is not a stop.
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveAttribute(
      'tabindex',
      '-1',
    )
    await userEvent.tab()
    await expect(canvas.getByLabelText('Architecture, parent')).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await expect(args.onActivate).toHaveBeenCalledWith('architecture')

    await userEvent.click(canvas.getByLabelText('Domain, child'))

    // Where it ends up is what a browser is needed for: that a click on an SVG
    // group lands, and that the plex settles with the chosen node centred.
    await waitFor(
      async () => {
        const focus = canvas.getByLabelText('Domain, focus').getBoundingClientRect()
        const centre = focus.x + focus.width / 2
        await expect(Math.abs(centre - (middle.x + middle.width / 2))).toBeLessThan(2)
      },
      { timeout: 3000 },
    )

    // The old focus took the seat above, and the plex is a plex again.
    await expect(canvas.getByLabelText('Hexagonal architecture, parent')).toBeInTheDocument()
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
  render: walking,
}

/** Too small a window: the plex is clipped rather than shrunk. */
export const SmallWindow: Story = {
  args: { neighbourhood: neighbourhoods.crowded },
  render: (args) => ({
    components: { Plex },
    setup: () => ({ args, options: computed(() => optionsFrom(args)) }),
    template: `
      <div style="height:100vh;display:grid;place-items:center;background:#8883">
        <div style="width:420px;height:320px;outline:1px solid var(--numen-node-border)">
          <Plex
            :neighbourhood="args.neighbourhood"
            :placement="args.placement"
            :options="options"
            :show-edge-labels="args.showEdgeLabels"
            :duration="args.duration"
            @activate="args.onActivate"
          />
        </div>
      </div>
    `,
  }),
}
