/** The Back and Forward buttons of a board. Each story is run in a browser by `@storybook/addon-vitest`. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import PlexNavigation from './PlexNavigation.vue'

const meta = {
  title: 'Plex/Navigation',
  component: PlexNavigation,
  args: { canBack: true, canForward: true, onBack: fn(), onForward: fn() },
  render: (args) => ({
    components: { PlexNavigation },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="padding:1rem;background:var(--numen-surface)">
        <PlexNavigation v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof PlexNavigation>

export default meta
type Story = StoryObj<typeof meta>

/** Both directions are open; pressing a button reports it. */
export const GoesBackAndForward: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'Back' }))
    await userEvent.click(await canvas.findByRole('button', { name: 'Forward' }))
    await expect(args.onBack).toHaveBeenCalledTimes(1)
    await expect(args.onForward).toHaveBeenCalledTimes(1)
  },
}

/** A first visit has nothing behind it or ahead of it. */
export const NoHistory: Story = {
  args: { canBack: false, canForward: false },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('button', { name: 'Back' })).toBeDisabled()
    await expect(await canvas.findByRole('button', { name: 'Forward' })).toBeDisabled()
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

/** Back and Forward at the top of a plex tab of the window. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: PLEX_FRAME,
    screenshot: [{ name: 'navigation', over: '[aria-label="Back"]' }],
  },
  render: (args) => ({
    components: { PlexNavigation },
    setup: () => ({ args, ...PLEX_HELPERS }),
    template: renderBoard(`
      <div :style="focusStyle">Entropy</div>
      <div style="position:absolute;inset-block-start:0.75rem;inset-inline-start:0.75rem"><PlexNavigation v-bind="args" /></div>
    `),
  }),
}
