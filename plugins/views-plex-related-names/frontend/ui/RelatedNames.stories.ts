/**
 * The names of the notes related to a note of the plex, as the slot of a node draws them. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import type { RelatedName } from '../types'
import RelatedNames from './RelatedNames.vue'

const toName = (title: string): RelatedName => ({
  path: `notes/${title}.md`,
  title,
  relation: 'child',
  label: `${title}, child of Pear rows`,
})

const meta = {
  title: 'Plex/Related names',
  component: RelatedNames,
  args: {
    names: ['Rootstock list', 'Grafting', 'Pollination'].map(toName),
    onChoose: fn(),
  },
  render: (args) => ({
    components: { RelatedNames },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;justify-content:center;inline-size:12rem;padding:1rem;background:var(--numen-surface)">
        <RelatedNames v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof RelatedNames>

export default meta
type Story = StoryObj<typeof meta>

/** Three names under a note; choosing one reports the path of its note. */
export const ChoosesAName: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(
      await canvas.findByRole('button', { name: 'Grafting, child of Pear rows' }),
    )
    await expect(args.onChoose).toHaveBeenCalledWith('notes/Grafting.md')
    await expect(args.onChoose).toHaveBeenCalledTimes(1)
  },
}

/** Titles too long for the width wrap onto more lines, each name whole. */
export const LongTitles: Story = {
  args: {
    names: [
      'A very long note about grafting pear scions onto quince rootstock in early spring',
      'Pollination',
    ].map(toName),
  },
}

/** A note with no names draws nothing. */
export const NoNames: Story = {
  args: { names: [] },
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

const childOf = (parent: string) => (title: string): RelatedName => ({
  path: `Physics/${title}.md`,
  title,
  relation: 'child',
  label: `${title}, child of ${parent}`,
})

/** The children of a note written under its node, in a plex tab of the window. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: PLEX_FRAME,
    screenshot: [{ name: 'names' }],
  },
  args: {
    names: ['Clausius inequality', 'Carnot cycle', 'Heat engines'].map(childOf('The second law')),
  },
  render: (args) => ({
    components: { RelatedNames },
    setup: () => ({ args, ...PLEX_HELPERS }),
    template: renderBoard(`
      <div :style="focusStyle">Entropy</div>
      <div :style="getBelowStyle(-200, 130)"><RelatedNames v-bind="args" /></div>
    `),
  }),
}
