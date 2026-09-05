/**
 * The splitter that composes panes, drawn on its own.
 *
 * A branch is told what the workspace holds rather than handed it, so these
 * provide that themselves and keep the shares a handle settles on.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, provide, ref } from 'vue'
import WorkspaceBranch from './WorkspaceBranch.vue'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from './context'
import Filling from '../fixtures/Filling.vue'
import { split, stack } from '../fixtures/build'
import { type Branch, type NodeId, type Tab, type TabId } from '../node'
import { fit } from '../shares'

const TITLES: Readonly<Record<string, string>> = {
  one: 'One',
  two: 'Two',
  three: 'Three',
  four: 'Four',
}

/** The arrangements a reader can start from. */
const ARRANGEMENTS = {
  'two panes': () =>
    split('root', [stack('left', 'one'), stack('right', 'two')], [0.7, 0.3]) as Branch,
  nested: () =>
    split('root', [
      stack('left', 'one'),
      split('down', [stack('upper', 'two'), stack('lower', 'three')]),
    ]) as Branch,
  'three across': () =>
    split('root', [stack('a', 'one'), stack('b', 'two'), stack('c', 'three')]) as Branch,
  'three across, unevenly': () =>
    split(
      'root',
      [stack('a', 'one'), stack('b', 'two'), stack('c', 'three')],
      [0.7, 0.2, 0.1],
    ) as Branch,
} satisfies Record<string, () => Branch>

type Arrangement = keyof typeof ARRANGEMENTS

interface Knobs {
  arrangement: Arrangement
  /** The least room a pane is worth drawing in. */
  minimum: number
  /** How wide the box holding the branch is drawn. */
  room: string
  /** Given by the story, and nothing a reader turns. */
  node?: never
  axis?: never
  depth?: never
}

const meta: Meta<Knobs> = {
  title: 'Workspace/Branch',
  component: WorkspaceBranch,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    arrangement: {
      control: 'select',
      options: Object.keys(ARRANGEMENTS),
      description: 'What the branch starts as. Changing it starts afresh.',
    },
    minimum: { control: { type: 'range', min: 40, max: 400, step: 20 } },
    room: { control: 'text' },
    node: { table: { disable: true } },
    axis: { table: { disable: true } },
    depth: { table: { disable: true } },
  },
  args: { arrangement: 'two panes', minimum: 220, room: '100%' },
  render: (args) => ({
    components: { WorkspaceBranch, Filling },
    setup() {
      const held = ref<Branch>(ARRANGEMENTS[args.arrangement]())

      /** The shares the last handle settled on, written where a story can read them. */
      const resize = (branch: NodeId, sizes: readonly number[]) => {
        held.value = kept(held.value, branch, sizes)
      }

      provide(
        WORKSPACE_CONTEXT,
        computed<WorkspaceContext>(() => ({
          tabOf: (id: TabId): Tab | undefined =>
            TITLES[id] === undefined ? undefined : { id, title: TITLES[id] },
          focus: 'left',
          minimum: args.minimum,
          choose: () => {},
          close: () => {},
          lift: () => {},
          claim: () => {},
          resize,
          show: () => {},
        })),
      )

      return { held, args }
    },
    template: `
      <div :style="{ height: '100vh', inlineSize: args.room, padding: 0 }">
        <WorkspaceBranch :node="held" axis="horizontal" :depth="0">
          <template #tab="{ id }"><Filling :name="id" /></template>
        </WorkspaceBranch>
      </div>
    `,
  }),
}

/** The branch with one of its own given other shares. */
function kept(node: Branch, branch: NodeId, sizes: readonly number[]): Branch {
  if (node.id === branch) return { ...node, sizes: fit(sizes, node.children.length) }
  return {
    ...node,
    children: node.children.map((child) =>
      child.kind === 'branch' ? kept(child, branch, sizes) : child,
    ),
  }
}

export default meta
type Story = StoryObj<Knobs>

/** Two panes side by side, the first given the larger share. */
export const TwoPanes: Story = {
  play: async ({ canvasElement }) => {
    const panes = canvasElement.querySelectorAll('[data-workspace-pane]')
    await expect(panes).toHaveLength(2)

    const [left, right] = [...panes].map((pane) => pane.getBoundingClientRect().width)
    await expect(left).toBeGreaterThan(right!)
  },
}

/** A branch inside a branch, each turning a quarter from the one above. */
export const Nested: Story = {
  args: { arrangement: 'nested' },
  play: async ({ canvasElement }) => {
    const panes = [...canvasElement.querySelectorAll('[data-workspace-pane]')]
    await expect(panes.map((pane) => pane.getAttribute('data-workspace-pane'))).toEqual([
      'left',
      'upper',
      'lower',
    ])

    // The outer branch divides across and the inner one down.
    const upper = panes[1]!.getBoundingClientRect()
    const lower = panes[2]!.getBoundingClientRect()
    await expect(lower.top).toBeGreaterThan(upper.top)
    await expect(Math.round(lower.left)).toBe(Math.round(upper.left))
  },
}

/** Three panes across, so a branch carries more than one handle. */
export const ThreeAcross: Story = {
  args: { arrangement: 'three across' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-workspace-pane]')).toHaveLength(3)
    await expect(canvasElement.querySelectorAll('[role="separator"]')).toHaveLength(2)
  },
}

/**
 * A pane given less room than it is worth drawing in: three panes in a box too
 * narrow to give any of them the least room. The floor a handle leaves is then
 * an equal share, and driving the first handle as far as it goes stops there.
 */
export const CrowdedByTheLeastRoom: Story = {
  args: { arrangement: 'three across, unevenly', minimum: 400, room: '600px' },
  play: async ({ canvasElement }) => {
    const widths = () =>
      [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(
        (pane) => pane.getBoundingClientRect().width,
      )
    await expect(widths()).toHaveLength(3)

    // The crowding the story is named for: three panes each worth 400 pixels do
    // not fit in the room the branch has.
    const room = widths().reduce((all, one) => all + one, 0)
    await expect(room).toBeLessThan(3 * 400)

    const handle = canvasElement.querySelector<HTMLElement>('[role="separator"]')
    handle?.focus()
    for (let press = 0; press < 20; press += 1) await userEvent.keyboard('{ArrowLeft}')

    await waitFor(async () => {
      await expect(Math.abs((widths()[0] ?? 0) - room / 3)).toBeLessThan(2)
    })
  },
}
