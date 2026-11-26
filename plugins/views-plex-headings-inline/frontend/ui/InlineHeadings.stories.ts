/**
 * The headings of a note as the slot under a node writes them. Also the test corpus: each story is run in a
 * browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import InlineHeadings from './InlineHeadings.vue'

const meta = {
  title: 'Plex/Inline headings',
  component: InlineHeadings,
  args: {
    headings: [
      { text: 'What it measures', level: 2, line: 4 },
      { text: "Boltzmann's formula", level: 3, line: 19 },
      { text: 'The arrow of time', level: 2, line: 57 },
    ],
    onChoose: fn(),
  },
  render: (args) => ({
    components: { InlineHeadings },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;justify-content:center;inline-size:12rem;padding:1rem;background:var(--numen-surface)">
        <InlineHeadings v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof InlineHeadings>

export default meta
type Story = StoryObj<typeof meta>

/** Three headings under a note, each after a hash; choosing one reports its line. */
export const ChoosesAHeading: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: '# The arrow of time' }))
    await expect(args.onChoose).toHaveBeenCalledWith(57)
    await expect(args.onChoose).toHaveBeenCalledTimes(1)
  },
}

/** A note with no headings draws nothing. */
export const NoHeadings: Story = {
  args: { headings: [] },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).queryByRole('button')).toBeNull()
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

/** The headings of the note in focus written under its node, in a plex tab of the window. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: PLEX_FRAME,
    screenshot: [{ name: 'headings' }],
  },
  render: (args) => ({
    components: { InlineHeadings },
    setup: () => ({ args, ...PLEX_HELPERS }),
    template: renderBoard(`
      <div :style="focusStyle">Entropy</div>
      <div :style="getBelowStyle(0, 0)"><InlineHeadings v-bind="args" /></div>
    `),
  }),
}
