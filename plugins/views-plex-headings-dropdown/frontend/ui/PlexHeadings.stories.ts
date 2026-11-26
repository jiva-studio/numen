/**
 * The headings of a note as the plex draws them under a node: the view of the slot over the whole node, run as
 * the plugin registers it, in a surface that is a stand-in for the plex. Also the test corpus: each story is run in
 * a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import type { Component } from 'vue'
import type { NoteHeading, VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from '../index'
import { HOVER_WAIT } from '../model/useHeadings'

const PATH = 'physics/Entropy.md'

const SECTIONS: readonly NoteHeading[] = [
  { text: 'What it measures', level: 2, line: 4 },
  { text: "Boltzmann's formula", level: 3, line: 19 },
  { text: 'Gibbs and the ensemble', level: 3, line: 38 },
  { text: 'The arrow of time', level: 2, line: 57 },
  { text: 'Missing information', level: 2, line: 76 },
  { text: "Shannon's H", level: 3, line: 94 },
  { text: 'The demon, and its bookkeeping', level: 2, line: 118 },
]

const MANY: readonly NoteHeading[] = Array.from({ length: 30 }, (_, index) => ({
  text: `Section ${index + 1}`,
  level: 1,
  line: index * 3,
}))

/** The most headings the setting allows by default. */
const LIMIT = 6

const WIDTH = 144
const HEIGHT = 36
const NODE = { x: 0, y: -120, width: WIDTH, height: HEIGHT }

interface Knobs {
  headings: readonly NoteHeading[]
  onOpen: (path: string, line: number) => void
}

const createView = (args: Knobs): Component => {
  const context = createMockPluginContext({ pluginId: 'md.numen.views.plex.headings.dropdown' })
  const notes = {
    getHeadings: async () => new Map([[PATH, args.headings]]),
  } as unknown as VaultNotes
  const workspace = {
    files: {
      open: (path: string, options?: { line?: number }) => args.onOpen(path, options?.line ?? -1),
    },
  } as unknown as WorkspaceHost
  activate(context, createMockHostServices({ notes, workspace }))
  return context.getRegisteredSlotViews()[0]!.component as Component
}

const meta = {
  title: 'Plex Headings Dropdown',
  args: { headings: SECTIONS, onOpen: fn() },
  parameters: { layout: 'fullscreen' },
  render: (args) => ({
    setup: () => {
      const view = createView(args)
      const box = {
        left: `${window.innerWidth / 2 + NODE.x - WIDTH / 2}px`,
        top: `${window.innerHeight / 2 + NODE.y - HEIGHT / 2}px`,
      }
      return { view, box, path: PATH, node: NODE }
    },
    template: `
      <div class="numen" style="position:relative;block-size:100vh;background:var(--numen-surface)">
        <div class="node" :style="{ position:'absolute', inlineSize:'144px', blockSize:'36px', boxSizing:'border-box', left: box.left, top: box.top, border:'var(--numen-stroke) solid var(--numen-rule)', borderRadius:'0.375rem', background:'var(--numen-raised)', color:'var(--numen-ink)', fontFamily:'var(--numen-font-sans)', fontSize:'var(--numen-font-size)', display:'grid', placeItems:'center' }">
          Entropy
          <component :is="view" :path="path" :x="node.x" :y="node.y" :width="node.width" :height="node.height" :scale="1" />
        </div>
      </div>
    `,
  }),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const getView = (canvasElement: HTMLElement) =>
  canvasElement.querySelector<HTMLElement>('.headings')!

const getRows = () => [...document.querySelectorAll('.headings__row')]

/** The pointer resting on the node brings the first six of its headings out under the box, over the window, and leaving puts them away. */
export const Nesting: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'parts', app: 'docs', over: '.node' },
    ],
  },
  play: async ({ canvasElement }) => {
    const view = getView(canvasElement)
    await expect(getRows()).toHaveLength(0)

    await userEvent.hover(view)
    await waitFor(async () => await expect(getRows()).toHaveLength(LIMIT), {
      timeout: HOVER_WAIT * 6,
    })
    await expect(within(document.body).getByText('The arrow of time')).toBeInTheDocument()
    await expect(within(document.body).queryByText('The demon, and its bookkeeping')).toBeNull()

    await userEvent.unhover(view)
    await waitFor(async () => await expect(getRows()).toHaveLength(0))
  },
}

/** Choosing a heading opens its note at its line. */
export const ChoosesAHeading: Story = {
  play: async ({ args, canvasElement }) => {
    await userEvent.hover(getView(canvasElement))
    await waitFor(async () => await expect(getRows()).toHaveLength(LIMIT), {
      timeout: HOVER_WAIT * 6,
    })

    await userEvent.click(within(document.body).getByText('Missing information'))

    await expect(args.onOpen).toHaveBeenCalledWith(PATH, 76)
  },
}

/** More headings than fit: six stand, a mark shows there are more, and the wheel scrolls a heading at a time. */
export const ScrollsWithTheWheel: Story = {
  args: { headings: MANY },
  play: async ({ canvasElement }) => {
    await userEvent.hover(getView(canvasElement))
    await waitFor(async () => await expect(getRows()).toHaveLength(LIMIT), {
      timeout: HOVER_WAIT * 6,
    })
    await expect(document.querySelectorAll('.headings__more')).toHaveLength(1)

    const list = document.querySelector('.headings__list')!
    list.dispatchEvent(
      new WheelEvent('wheel', { deltaY: 1, deltaMode: 1, bubbles: true, cancelable: true }),
    )

    await waitFor(async () => await expect(getRows()[0]!.textContent?.trim()).toBe('Section 2'))
    await expect(document.querySelectorAll('.headings__more')).toHaveLength(2)
  },
}

/** Words longer than the list can hold end in an ellipsis, and the row is shown. */
export const LongHeading: Story = {
  args: {
    headings: [
      {
        text: 'A heading that runs on well past anything the box it hangs from could hold, and past the window too',
        level: 1,
        line: 3,
      },
    ],
  },
  play: async ({ canvasElement }) => {
    await userEvent.hover(getView(canvasElement))
    await waitFor(async () => await expect(getRows()).toHaveLength(1), {
      timeout: HOVER_WAIT * 6,
    })

    await waitFor(async () => await expect(getRows()[0]).toBeVisible())
  },
}

/** The notes around Entropy, drawn as a stand-in for the plex the view is a slot of. */
const PLEX_NODES = [
  { title: 'Thermodynamics', x: -100, y: -120, relation: 'parent' },
  { title: 'Information theory', x: 100, y: -120, relation: 'parent' },
  { title: "Maxwell's demon", x: -290, y: 0, relation: 'jump' },
  { title: "Landauer's principle", x: 290, y: 0, relation: 'jump' },
  { title: 'The second law', x: -200, y: 130, relation: 'child' },
  { title: 'Heat death', x: 0, y: 130, relation: 'child' },
  { title: 'Free energy', x: 200, y: 130, relation: 'child' },
] as const

const NODE_BOX =
  'position:absolute;inline-size:150px;block-size:36px;box-sizing:border-box;display:grid;' +
  'place-items:center;border-radius:0.375rem;font-size:var(--numen-font-size)'

const PLEX_HELPERS = {
  nodes: PLEX_NODES,
  getNodeStyle: (x: number, y: number, relation: string) =>
    `${NODE_BOX};left:calc(50% + ${x - 75}px);top:calc(45% + ${y - 18}px);` +
    `background:var(--numen-raised);color:var(--numen-ink);` +
    `border:1.5px solid var(--numen-relation-${relation})`,
  focusStyle:
    `${NODE_BOX};left:calc(50% - 75px);top:calc(45% - 18px);` +
    'background:var(--numen-accent);color:var(--numen-accent-ink);border:none',
  getBelowStyle: (x: number, y: number) =>
    `position:absolute;inline-size:12rem;left:calc(50% + ${x - 96}px);top:calc(45% + ${y + 22}px);` +
    'display:flex;justify-content:center',
}

const renderBoard = (view: string) => `
  <div class="numen" style="position:relative;block-size:100%;overflow:hidden;background:var(--numen-surface);font-family:var(--numen-font-sans)">
    <svg style="position:absolute;left:50%;top:45%;overflow:visible" width="1" height="1" aria-hidden="true">
      <line v-for="node in nodes" :key="node.title" x1="0" y1="0" :x2="node.x" :y2="node.y" style="stroke:var(--numen-edge);stroke-width:1.5" />
    </svg>
    <div v-for="node in nodes" :key="node.title" :style="getNodeStyle(node.x, node.y, node.relation)">{{ node.title }}</div>
    ${view}
  </div>
`

const PLEX_FRAME = {
  place: 'tab',
  tab: { title: 'Entropy', icon: 'plex' },
  note: 'after',
  share: 0.62,
  selected: 'Physics/Entropy.md',
}

/** The headings of the note in focus hung under its node, in a plex tab of the window. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: PLEX_FRAME,
    screenshot: [{ name: 'headings', over: '.node' }],
  },
  render: (args) => ({
    setup: () => ({ view: createView(args), path: PATH, node: NODE, ...PLEX_HELPERS }),
    template: renderBoard(`
      <div class="node" :style="focusStyle">
        Entropy
        <component :is="view" :path="path" :x="node.x" :y="node.y" :width="node.width" :height="node.height" :scale="1" />
      </div>
    `),
  }),
}
