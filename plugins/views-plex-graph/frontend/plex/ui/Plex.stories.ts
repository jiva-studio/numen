import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref } from 'vue'
import Plex from './Plex.vue'
import { resolveOptions, clusters, type Placement, type PlexOptionsInput } from '../lib/arrange'
import { optionsForType, useTypeSize } from '../model/sizing/plexScale'
import { EXPANSION_DELAY } from '../model/sizing/expansion'
import { graphs } from '../fixtures/graphs'
import { createGraph, walkStart } from '../fixtures/walk'
import { build, buildAround, type TitledNode } from '../fixtures/build'
import { nameNow } from '../fixtures/names'
import { ring } from '../fixtures/ring'
import type { Position } from '../lib/node'
import type { Clock } from '../model/transition'
import type { MenuTrigger } from '@numen/ui'
import type {
  PlexHeading,
  PlexEdge,
  PlexGraph,
  PlexNode,
  PlexRelation,
  PlexDestination,
} from '../../types'

interface Knobs {
  graph: PlexGraph
  placement: Placement
  showEdgeLabels: boolean
  highlightLinks: boolean
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
  maxPerCluster: number
  maxHeadings: number
  orientation: 'parents above' | 'parents below'

  curvature: number
  minReach: number
  arrowClearance: number

  arriveAfter: number
  leaveBefore: number

  parents: number
  children: number
  jumps: number
  siblings: number
  title: string

  naming: () => string

  dragged: readonly string[]

  dropName: (relation: PlexRelation) => string

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
  maxPerCluster: a.maxPerCluster,
  maxHeadings: a.maxHeadings,
  routing: {
    curvature: a.curvature,
    minReach: a.minReach,
    arrowClearance: a.arrowClearance,
  },
  motion: { arriveAfter: a.arriveAfter, leaveBefore: a.leaveBefore },
  direction:
    a.orientation === 'parents below'
      ? { parent: 'down', child: 'up', jump: 'left', reference: 'left', sibling: 'right' }
      : { parent: 'up', child: 'down', jump: 'left', reference: 'left', sibling: 'right' },
})

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

const createNavigableStory = (start: (args: Knobs) => PlexGraph) => (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focus = ref<TitledNode | null>(null)
    const cameFrom = ref<TitledNode | null>(null)

    const createdItem = ref<PlexNode[]>([])
    const links = ref<PlexEdge[]>([])

    const graph = computed<PlexGraph>(() => {
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
      const selected = graph.value.nodes.find((node) => node.id === id)
      if (!selected) return
      const was = graph.value.nodes.find((node) => node.relation === 'focus')
      cameFrom.value = was ? { id: was.id, title: was.title } : null
      focus.value = { id: selected.id, title: selected.title }
    }

    const createRelatedNode = (from: string, title: string, relation: PlexRelation) => {
      const id = `made/${createdItem.value.length}`
      createdItem.value = [...createdItem.value, { id, title, relation }]
      links.value = [
        ...links.value,
        relation === 'parent' || relation === 'jump'
          ? { from: id, to: from, labels: [{ text: relation === 'jump' ? 'see also' : 'is a' }] }
          : { from, to: id, labels: [{ text: 'contains' }] },
      ]
    }

    const create = (from: string, relation: PlexRelation) => {
      createRelatedNode(from, args.naming(), relation)
    }

    const dropOnto = (ids: readonly string[], relation: PlexRelation) => {
      const here = graph.value.nodes.find((node) => node.relation === 'focus')
      if (!here) return
      for (const one of ids) createRelatedNode(here.id, one, relation)
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
      graph,
      options: computed(() => createOptions(args, type.value)),
    }
  },
  template: `
    <div style="height:100vh">
      <Plex
        :graph="graph"
        :placement="args.placement"
        :options="options"
        :show-edge-labels="args.showEdgeLabels"
        :highlight-links="args.highlightLinks"
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

const renderWalk = (args: Knobs) => ({
  components: { Plex },
  setup() {
    const type = useTypeSize()
    const focused = ref(walkStart)
    return {
      args,
      focused,
      graph: computed(() => createGraph(focused.value)),
      options: computed(() => createOptions(args, type.value)),
    }
  },
  template: `
    <div style="height:100vh">
      <Plex
        :graph="graph"
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
        component: 'Interactive plex graph component.',
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
    maxPerCluster: range(1, 40),
    maxHeadings: range(1, 12),
    orientation: {
      control: 'inline-radio',
      options: ['parents above', 'parents below'],
      table: { category: 'Layout' },
    },
    placement: {
      control: 'inline-radio',
      options: ['clusters', 'ring'],
      mapping: { clusters, ring },
      table: { category: 'Layout' },
    },

    curvature: { ...range(0, 1, 0.05, 'Edges'), table: { category: 'Edges' } },
    minReach: { ...range(0, 120, 2, 'Edges'), table: { category: 'Edges' } },
    arrowClearance: { ...range(0, 40, 1, 'Edges'), table: { category: 'Edges' } },
    showEdgeLabels: { control: 'boolean', table: { category: 'Edges' } },
    highlightLinks: { control: 'boolean', table: { category: 'Edges' } },

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

    graph: { table: { disable: true } },
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
    placement: clusters,
    showEdgeLabels: true,
    highlightLinks: false,
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
    maxPerCluster: 20,
    maxHeadings: 6,
    orientation: 'parents above',

    curvature: 0.55,
    minReach: 22,
    arrowClearance: 20,
    arriveAfter: 0.35,
    leaveBefore: 0.45,

    parents: 2,
    children: 6,
    jumps: 3,
    siblings: 2,
    title: 'A node',
  },

  render: createNavigableStory((args) => args.graph),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const invented = {
  args: { graph: graphs.typical, maxPerCluster: 9 },
  render: createNavigableStory((args: Knobs) => build(args.title, createCounts(args))),
}

/** Interactive playground story. */
export const Playground: Story = invented

/** Creates a new parent node by dragging from the focus handle. */
export const MakingOne: Story = {
  args: { ...invented.args, naming: () => 'Reached for' },
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)
    const box = focus.getBoundingClientRect()

    await expect(canvasElement.querySelector('.plex__handle')).toBeNull()
    await userEvent.hover(focus)
    const surface = canvasElement.querySelector('svg')!
    const handle = canvasElement.querySelector('.plex__handle')
    await expect(handle).not.toBeNull()

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

    surface.dispatchEvent(new PointerEvent('pointerup', { ...above, pointerId: 1, bubbles: true }))
    await expect(args.onCreate).toHaveBeenCalledWith(expect.any(String), 'parent')

    await expect(canvasElement.querySelector('.plex__handle')).toBeNull()

    await waitFor(async () => {
      await expect(canvas.getByLabelText('Reached for, parent')).toBeInTheDocument()
    })
  },
}

/** Displays external drag preview over the plex. */
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

    dragTo(middle, box.y - 220)
    await expectDraggedTitle('as parent')
    await expect(canvasElement.querySelector('.plex__dragged .plex__thread')).not.toBeNull()

    dragTo(middle, box.bottom + 220)
    await expectDraggedTitle('as child')

    dragTo(middle, -400)
    await expectDraggedTitle(null)
    await expect(canvasElement.querySelector('.plex__dragged')).toBeNull()

    dragTo(middle, box.y - 220)
    await expectDraggedTitle('as parent')
  },
}

interface KeyboardFocus extends FocusOptions {
  focusVisible?: boolean
}

/** Shows at most one handle at a time for pointer hover or keyboard focus. */
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

    await expect(getHandles()).toHaveLength(0)

    first.focus({ focusVisible: false } as KeyboardFocus)
    await expect(first).toHaveFocus()
    await expect(getHandles()).toHaveLength(0)

    for (const box of boxes) {
      await userEvent.hover(box)
      await expect(getHandles()).toHaveLength(1)
      await expect(box.querySelector('.plex__handle')).not.toBeNull()
      await userEvent.unhover(box)
      await expect(getHandles()).toHaveLength(0)
    }

    first.blur()
    await userEvent.tab()
    const stop = canvasElement.ownerDocument.activeElement!
    await expect(stop).toHaveClass('plex__node')
    await expect(getHandles()).toHaveLength(1)
    await expect(stop.querySelector('.plex__handle')).not.toBeNull()
  },
}

/** Opens the context menu on right-click. */
export const AskingForAMenu: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    let isPrevented: boolean | null = null
    const watch = (event: Event) => {
      isPrevented = event.defaultPrevented
    }
    canvasElement.addEventListener('contextmenu', watch)
    await userEvent.pointer({ keys: '[MouseRight]', target: focus })
    canvasElement.removeEventListener('contextmenu', watch)

    await expect(args.onMenu).toHaveBeenCalledWith(
      expect.any(String),
      expect.any(Object),
      'pointer',
    )
    await expect(isPrevented).toBe(true)

    await userEvent.hover(focus)
    const handle = canvasElement.querySelector('.plex__handle')!
    await userEvent.pointer({ keys: '[MouseRight]', target: handle })
    await expect(args.onCreate).not.toHaveBeenCalled()
    await expect(canvasElement.querySelector('.plex__thread')).toBeNull()
    await expect(args.onMenu).toHaveBeenCalledTimes(2)
  },
}

/** Emits show on double-click or Shift+Enter. */
export const AskingForANode: Story = {
  args: invented.args,
  render: invented.render,
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    const hand = userEvent.setup()

    await hand.dblClick(focus)
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'current')
    await expect(args.onActivate).not.toHaveBeenCalled()

    await hand.keyboard('{Alt>}')
    await hand.dblClick(focus)
    await hand.keyboard('{/Alt}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'split')

    focus.focus()
    await hand.keyboard('{Shift>}{Enter}{/Shift}')
    await expect(args.onShow).toHaveBeenLastCalledWith(expect.any(String), 'current')
    await expect(args.onShow).toHaveBeenCalledTimes(3)
  },
}

/** Renders a graph with multiple connected parents. */
export const SeveralParents: Story = {
  args: { graph: graphs.diamond },
}

/** Renders an overcrowded graph with overflow indicators. */
export const Overcrowded: Story = {
  args: { graph: graphs.overcrowded },
  play: async ({ canvasElement }) => {
    const status = await waitFor(() => {
      const found = canvasElement.querySelector('[role="status"]')
      if (!found) throw new Error('nothing says what is not shown')
      return found
    })
    await expect(status.textContent).toMatch(/\d+ (children|child)\b.* not shown/)

    const boxes = [...canvasElement.querySelectorAll<HTMLElement>('.plex__box')].map((box) =>
      box.getBoundingClientRect(),
    )
    await expect(boxes.length).toBeGreaterThan(1)
    const overlaps = boxes.flatMap((a, index) =>
      boxes
        .slice(index + 1)
        .filter(
          (b) =>
            a.left < b.right - 0.5 &&
            b.left < a.right - 0.5 &&
            a.top < b.bottom - 0.5 &&
            b.top < a.bottom - 0.5,
        ),
    )
    await expect(overlaps).toHaveLength(0)
  },
}

const titledLines: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'The meeting', relation: 'focus' },
    { id: 'parent', title: 'Minutes', relation: 'parent' },
    { id: 'earlier', title: 'The morning', relation: 'jump' },
    { id: 'across', title: 'Elsewhere', relation: 'jump' },
    { id: 'later', title: 'The evening', relation: 'jump' },
    { id: 'sibling', title: 'The hedge', relation: 'sibling' },
  ],
  edges: [
    { from: 'parent', to: 'focus', labels: [{ text: 'is a' }] },
    { from: 'parent', to: 'sibling', labels: [{ text: 'contains' }] },
    { from: 'earlier', to: 'focus', labels: [{ text: 'see also' }] },
    { from: 'across', to: 'focus', labels: [{ text: 'the minutes of the meeting' }] },
    { from: 'later', to: 'focus', labels: [{ text: 'and also' }] },
  ],
}

/** Renders edge labels along curved paths, truncated where they do not fit, and raises hovered lines. */
export const TitledLines: Story = {
  args: { graph: titledLines, duration: 0, spread: 1 },
  play: async ({ canvasElement }) => {
    const title = (words: string) =>
      [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')].filter(
        (text) => text.textContent === words,
      )

    await waitFor(async () => {
      await expect(title('contains')).toHaveLength(2)
    })

    const titles = [...canvasElement.querySelectorAll<SVGTextElement>('.plex__edge-label')]
    await expect(titles.length).toBeGreaterThan(0)
    for (const text of titles) {
      await expect(text.querySelector('textPath')).not.toBeNull()
    }

    await expect(title('the minutes of the meeting')).toHaveLength(0)
    const truncatedTitles = titles.filter((text) => text.textContent!.endsWith('…'))
    await expect(truncatedTitles).toHaveLength(2)
    await expect(
      'the minutes of the meeting'.startsWith(truncatedTitles[0]!.textContent!.slice(0, -1)),
    ).toBe(true)

    const [halo] = title('contains')
    await expect(halo!.querySelector('textPath')).not.toBeNull()
    const axisHref = halo!.querySelector('textPath')!.getAttribute('href')!
    const viewElement = canvasElement
      .querySelector(`defs path[id="${axisHref.slice(1)}"]`)!
      .getAttribute('d')
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === viewElement,
    )

    const raised = await waitFor(async () => {
      await userEvent.unhover(band!)
      await userEvent.hover(band!)
      const found = canvasElement.querySelectorAll<SVGTextElement>(
        '.plex__raised .plex__edge-label',
      )
      if (found.length !== 2) throw new Error(`${found.length} raised titles, not 2`)
      return found
    })
    await expect(raised[0]!.textContent).toBe(raised[1]!.textContent)
    await userEvent.unhover(band!)

    const loop = title('see also')[0]!
    await expect(loop.querySelector('textPath')).not.toBeNull()
  },
}

const arrowedLines: PlexGraph = {
  nodes: [
    { id: 'alice', title: 'Alice Fenn, The Chair', relation: 'focus' },
    { id: 'allotments', title: 'Marrowfield allotments', relation: 'parent' },
    { id: 'bram', title: 'Bram Doyle', relation: 'jump' },
    { id: 'cora', title: 'Cora Hale', relation: 'jump' },
    { id: 'rota', title: 'The question of the rota', relation: 'sibling' },
  ],
  edges: [
    { from: 'allotments', to: 'alice', arrows: ['from'] },
    { from: 'bram', to: 'alice', labels: [{ text: 'the oldest tenant' }], arrows: ['from'] },
    { from: 'cora', to: 'alice', labels: [{ text: 'keys, hoses, the gate' }] },
    { from: 'allotments', to: 'rota', labels: [{ text: 'the minutes of the meeting' }] },
  ],
}

/** Renders directional arrowheads on edges and raises them on hover. */
export const ArrowedLines: Story = {
  args: { graph: arrowedLines, duration: 0 },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const svg = canvasElement.querySelector('svg')!

    const hasPoint = (box: DOMRect, point: DOMPoint) =>
      point.x >= box.left - 1 &&
      point.x <= box.right + 1 &&
      point.y >= box.top - 1 &&
      point.y <= box.bottom + 1

    const lines = () => {
      const onScreen = svg.getScreenCTM()!
      return [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge')].map((line) => ({
        line,
        ends: [line.getPointAtLength(0), line.getPointAtLength(line.getTotalLength())].map(
          (point) => point.matrixTransform(onScreen),
        ),
      }))
    }

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

    const getHead = (end: DOMPoint) =>
      heads().some((head) => hasPoint(head.getBoundingClientRect(), end))

    const readView = () =>
      JSON.stringify([
        heads().map((head) => {
          const box = head.getBoundingClientRect()
          return [box.x, box.y, box.width, box.height]
        }),
        lines().map(({ ends }) => ends.map((end) => [end.x, end.y])),
      ])

    const frame = () => new Promise((isDone) => requestAnimationFrame(() => isDone(null)))

    await waitFor(async () => {
      await expect(heads()).toHaveLength(2)
      const was = readView()
      await frame()
      await expect(readView()).toBe(was)
    })

    await expect(getHead(getEnd('Marrowfield allotments, parent'))).toBe(true)
    await expect(getHead(getEnd('Bram Doyle, jump'))).toBe(true)

    await expect(getHead(getEnd('Cora Hale, jump'))).toBe(false)
    await expect(getHead(getEnd('The question of the rota, sibling'))).toBe(false)

    const jump = getLine('Bram Doyle, jump')
    const band = [...canvasElement.querySelectorAll<SVGPathElement>('.plex__edge-hit')].find(
      (line) => line.getAttribute('d') === jump.line.getAttribute('d'),
    )!
    await userEvent.hover(band)

    const raised = canvasElement.querySelector<SVGPathElement>('.plex__raised .plex__edge-arrow')!
    await expect(raised).not.toBeNull()
  },
}

/** Renders nodes with long, non-Latin, and empty labels. */
export const AwkwardLabels: Story = {
  args: { graph: graphs.awkwardLabels },
}

/** Expands a truncated node box on hover to show the full title. */
export const RestingOnATitle: Story = {
  args: { graph: graphs.awkwardLabels },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const node = canvas.getByLabelText(/^Supercalifragilistic/)
    const adjacentNode = canvas.getByLabelText('日本語のノート, child')

    const getBox = () =>
      canvasElement.querySelector('.plex__lifted .plex__box') ?? node.querySelector('.plex__box')!
    const getWidth = () => getBox().getBoundingClientRect().width
    const getWords = () =>
      (canvasElement.querySelector('.plex__lifted') ?? node).querySelector<HTMLElement>(
        '.plex__title-text',
      )!
    const nodeWidth = getWidth()
    const adjacentX = adjacentNode.getBoundingClientRect().x

    const words = getWords()
    await expect(words.scrollWidth).toBeGreaterThan(words.clientWidth)

    await userEvent.hover(node)
    await waitFor(async () => await expect(getWidth()).toBeGreaterThan(nodeWidth), {
      timeout: 3000,
    })
    await waitFor(
      async () => await expect(getWords().scrollWidth).toBeLessThanOrEqual(getWords().clientWidth),
    )

    await expect(adjacentNode.getBoundingClientRect().x).toBe(adjacentX)
    const lifted = canvasElement.querySelector('.plex__lifted')!
    await expect(node.compareDocumentPosition(lifted) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )

    await userEvent.unhover(node)
    await waitFor(async () => await expect(getWidth()).toBe(nodeWidth))
  },
}

/** Navigates between nodes using keyboard and pointer. */
export const Walk: Story = {
  render: renderWalk,
  args: {
    graph: createGraph(walkStart),
    gap: 18,
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)

    /** The node is in the focus and has stopped moving. */
    const waitForNode = async (label: string) =>
      await waitFor(
        async () => {
          const read = () =>
            JSON.stringify(
              canvas.getByLabelText(label).querySelector('.plex__box')!.getBoundingClientRect(),
            )
          const was = read()
          await new Promise((isDone) => requestAnimationFrame(() => isDone(null)))
          await expect(read()).toBe(was)
        },
        { timeout: 3000 },
      )

    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveAttribute(
      'tabindex',
      '0',
    )
    await userEvent.tab()
    await expect(canvas.getByLabelText('Hexagonal architecture, focus')).toHaveFocus()

    await userEvent.tab()
    await userEvent.tab()
    await expect(canvas.getByLabelText('Architecture, parent')).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    await expect(args.onActivate).toHaveBeenCalledWith('architecture')

    await waitForNode('Architecture, focus')

    await userEvent.click(canvas.getByLabelText('Hexagonal architecture, child'))
    await waitForNode('Hexagonal architecture, focus')

    await expect(canvas.getByLabelText('Architecture, parent')).toBeInTheDocument()
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigates across connected nodes in a sample graph.',
      },
    },
  },
}

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
  'focus/jump-0': Array.from({ length: 30 }, (_, index) => ({
    id: `${index}`,
    text: `Section ${index + 1}`,
    level: 1,
  })),
}

/** Shows and scrolls note headings on hover. */
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
    const getOpenBox = (id: string) =>
      canvasElement.querySelector(`.plex__lifted[data-node="${id}"]`)
    const getHeadings = (id: string) => [
      ...(getOpenBox(id)?.querySelectorAll('.plex__heading') ?? []),
    ]

    const most = args.maxHeadings

    await expect(getHeadings('focus')).toHaveLength(0)

    await userEvent.hover(focus)
    await waitFor(async () => await expect(getHeadings('focus')).toHaveLength(most), {
      timeout: 3000,
    })

    await userEvent.unhover(focus)
    await waitFor(async () => await expect(getHeadings('focus')).toHaveLength(0))

    await userEvent.hover(focus)
    await waitFor(async () => await expect(getHeadings('focus')).toHaveLength(most), {
      timeout: 3000,
    })
    await userEvent.click(getHeadings('focus')[1]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus', '4')

    const many = canvas.getByLabelText(/, jump$/)
    await userEvent.hover(many)
    await waitFor(async () => await expect(getHeadings('focus/jump-0')).toHaveLength(most), {
      timeout: 3000,
    })

    const arrows = () => [...(getOpenBox('focus/jump-0')?.querySelectorAll('.plex__more') ?? [])]
    await expect(arrows()).toHaveLength(1)

    const wheel = (deltaY: number) =>
      getOpenBox('focus/jump-0')!
        .querySelector('.plex__inside')!
        .dispatchEvent(
          new WheelEvent('wheel', { deltaY, deltaMode: 1, bubbles: true, cancelable: true }),
        )
    wheel(1)
    await waitFor(
      async () =>
        await expect(getHeadings('focus/jump-0')[0]!.textContent?.trim()).toBe('Section 2'),
    )
    await expect(arrows()).toHaveLength(2)

    wheel(-1)
    await waitFor(
      async () =>
        await expect(getHeadings('focus/jump-0')[0]!.textContent?.trim()).toBe('Section 1'),
    )

    await userEvent.click(getHeadings('focus/jump-0')[0]!)
    await expect(args.onEnter).toHaveBeenCalledWith('focus/jump-0', '0')
    await expect(args.onActivate).not.toHaveBeenCalled()
  },
}

/** Draws the lines lifted by the highlight setting under the headings that open under a note. */
export const HeadingsOverLiftedLines: Story = {
  args: {
    ...invented.args,
    parents: 1,
    children: 4,
    jumps: 1,
    siblings: 0,
    highlightLinks: true,
    headings: (id: string) => INSIDE[id] ?? [],
  },
  render: invented.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = canvas.getByLabelText(/, focus$/)

    await userEvent.hover(focus)
    const backdrop = await waitFor(async () => {
      const found = canvasElement.querySelector('.plex__backdrop')
      if (found === null) throw new Error('the headings are not open')
      return found
    })
    const raised = await waitFor(async () => {
      const found = canvasElement.querySelector('.plex__raised')
      if (found === null) throw new Error('no line is lifted')
      return found
    })

    await expect(raised.querySelectorAll('.plex__edge-line').length).toBeGreaterThan(0)
    await expect(raised.compareDocumentPosition(backdrop) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
  },
}

/** Closes the note that was attended to as soon as another note is, whichever pointer events came. */
export const OneOpenNote: Story = {
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
    const other = canvas.getByLabelText(/, jump$/)
    const getOpenBox = (id: string) =>
      canvasElement.querySelector(`.plex__lifted[data-node="${id}"]`)
    const getHeadings = (id: string) => [
      ...(getOpenBox(id)?.querySelectorAll('.plex__heading') ?? []),
    ]
    const getHandles = () => canvasElement.querySelectorAll('.plex__handle')

    await userEvent.hover(focus)
    await waitFor(async () => await expect(getHeadings('focus')).toHaveLength(args.maxHeadings), {
      timeout: 3000,
    })
    await expect(getHandles()).toHaveLength(1)

    // The pointer reaches the other note without the first one hearing it leave.
    other.dispatchEvent(new PointerEvent('pointerenter'))

    await waitFor(async () => await expect(getHeadings('focus')).toHaveLength(0))
    await expect(getHandles()).toHaveLength(1)
    await expect(focus.querySelector('.plex__handle')).toBeNull()
    await expect(other.querySelector('.plex__handle')).not.toBeNull()

    await waitFor(
      async () => await expect(getHeadings('focus/jump-0')).toHaveLength(args.maxHeadings),
      {
        timeout: 3000,
      },
    )
    await expect(getHeadings('focus')).toHaveLength(0)
  },
}

const renderInSmallWindow: NonNullable<Story['render']> = (args) => ({
  components: { Plex },
  setup: () => {
    const type = useTypeSize()
    return { args, options: computed(() => createOptions(args, type.value)) }
  },
  template: `
      <div style="height:100vh;display:grid;place-items:center;background:#8883">
        <div style="width:420px;height:320px;outline:1px solid var(--numen-rule)">
          <Plex
            :graph="args.graph"
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
})

/** Renders the plex inside a small fixed container. */
export const SmallWindow: Story = {
  args: { graph: graphs.crowded },
  render: renderInSmallWindow,
}

const withReferences: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Salt marsh', relation: 'focus' },
    { id: 'parent', title: 'Coastal ecology', relation: 'parent' },
    { id: 'child', title: 'Tidal creeks', relation: 'child' },
    { id: 'jump', title: 'Dune grasses', relation: 'jump' },
    { id: 'sibling', title: 'Mudflats', relation: 'sibling' },
    { id: 'ref-out', title: 'Heron nesting notes', relation: 'reference' },
    { id: 'ref-in', title: 'Survey of the estuary', relation: 'reference' },
    { id: 'ref-both', title: 'Tide tables', relation: 'reference' },
  ],
  edges: [
    { from: 'parent', to: 'focus' },
    { from: 'focus', to: 'child' },
    { from: 'jump', to: 'focus' },
    { from: 'parent', to: 'sibling' },
    { from: 'focus', to: 'ref-out', arrows: ['to'] },
    { from: 'focus', to: 'ref-in', arrows: ['from'] },
    { from: 'focus', to: 'ref-both', arrows: ['from', 'to'] },
  ],
}

/** Renders the references as nodes of their own relation, pressable like any other, with an arrowhead at the end each link points to. */
export const References: Story = {
  args: { graph: withReferences, duration: 0 },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(async () => {
      await expect(canvasElement.querySelectorAll('.plex__node--reference')).toHaveLength(3)
    })
    await expect(canvas.getByLabelText('Heron nesting notes, reference')).toBeTruthy()
    await expect(canvas.getByLabelText('Survey of the estuary, reference')).toBeTruthy()
    await expect(canvas.getByLabelText('Tide tables, reference')).toBeTruthy()

    // One head for the outgoing and the incoming link, two for the mutual one.
    await waitFor(async () => {
      await expect(canvasElement.querySelectorAll('.plex__edge-arrow')).toHaveLength(4)
    })

    await userEvent.click(canvas.getByLabelText('Heron nesting notes, reference'))
    await expect(args.onActivate).toHaveBeenCalledWith('ref-out')
  },
}

const crowdedWithReferences: PlexGraph = build('Salt marsh', {
  parent: 1,
  child: 3,
  jump: 2,
  reference: 30,
})

/** Counts the references left out of a small window like any other relation. */
export const ManyReferencesInASmallWindow: Story = {
  args: { graph: crowdedWithReferences, duration: 0 },
  render: renderInSmallWindow,
  play: async ({ canvasElement }) => {
    const status = await waitFor(() => {
      const found = canvasElement.querySelector('[role="status"]')
      if (!found) throw new Error('nothing says what is not shown')
      return found
    })
    await expect(status.textContent).toMatch(
      /\d+ references?\b.* not shown|\d+ references? not shown/,
    )
  },
}
