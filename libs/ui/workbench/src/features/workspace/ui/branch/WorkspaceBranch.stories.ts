import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor } from 'storybook/test'
import { computed, provide, ref } from 'vue'
import WorkspaceBranch from './WorkspaceBranch.vue'
import { NO_STRIP_EDGE, WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import TabStub from '../../fixtures/TabStub.vue'
import { split, stack } from '../../fixtures/build'
import { type Branch, type NodeId, type Tab, type TabId } from '../../lib/node'
import { fit } from '../../lib/shares'

const TITLES: Readonly<Record<string, string>> = {
  one: 'One',
  two: 'Two',
  three: 'Three',
  four: 'Four',
}

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
  minimum: number
  room: string
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
      description: 'Initial branch layout.',
    },
    minimum: { control: { type: 'range', min: 40, max: 400, step: 20 } },
    room: { control: 'text' },
    node: { table: { disable: true } },
    axis: { table: { disable: true } },
    depth: { table: { disable: true } },
  },
  args: { arrangement: 'two panes', minimum: 220, room: '100%' },
  render: (args) => ({
    components: { WorkspaceBranch, TabStub },
    setup() {
      const target = ref<Branch>(ARRANGEMENTS[args.arrangement]())

      const resize = (branch: NodeId, sizes: readonly number[]) => {
        target.value = resizeBranch(target.value, branch, sizes)
      }

      provide(
        WORKSPACE_CONTEXT,
        computed<WorkspaceContext>(() => ({
          findTab: (id: TabId): Tab | undefined =>
            TITLES[id] === undefined ? undefined : { id, title: TITLES[id] },
          focus: 'left',
          minimum: args.minimum,
          getStripEdge: () => NO_STRIP_EDGE,
          choose: fn(),
          close: fn(),
          startTabPress: fn(),
          menu: fn(),
          claim: fn(),
          resize,
          show: fn(),
        })),
      )

      return { root: target, args }
    },
    template: `
      <div :style="{ height: '100vh', inlineSize: args.room, padding: 0 }">
        <WorkspaceBranch :node="root" axis="horizontal" :depth="0">
          <template #tab="{ id }"><TabStub :name="id" /></template>
        </WorkspaceBranch>
      </div>
    `,
  }),
}

function resizeBranch(node: Branch, branch: NodeId, sizes: readonly number[]): Branch {
  if (node.id === branch) return { ...node, sizes: fit(sizes, node.children.length) }
  return {
    ...node,
    children: node.children.map((child) =>
      child.kind === 'branch' ? resizeBranch(child, branch, sizes) : child,
    ),
  }
}

export default meta
type Story = StoryObj<Knobs>

/** Renders two panes side by side. */
export const TwoPanes: Story = {
  play: async ({ canvasElement }) => {
    const panes = canvasElement.querySelectorAll('[data-workspace-pane]')
    await expect(panes).toHaveLength(2)
  },
}

/** Renders a vertical branch nested inside a horizontal branch. */
export const Nested: Story = {
  args: { arrangement: 'nested' },
  play: async ({ canvasElement }) => {
    const panes = [...canvasElement.querySelectorAll('[data-workspace-pane]')]
    await expect(panes.map((pane) => pane.getAttribute('data-workspace-pane'))).toEqual([
      'left',
      'upper',
      'lower',
    ])

    const upper = panes[1]!.getBoundingClientRect()
    const lower = panes[2]!.getBoundingClientRect()
    await expect(lower.top).toBeGreaterThan(upper.top)
  },
}

/** Renders three panes horizontally with two resize handles. */
export const ThreeAcross: Story = {
  args: { arrangement: 'three across' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-workspace-pane]')).toHaveLength(3)
    await expect(canvasElement.querySelectorAll('[role="separator"]')).toHaveLength(2)
  },
}

/** Caps minimum pane size at an equal share when total width is below the sum of minimums. */
export const CrowdedByTheLeastSpace: Story = {
  args: { arrangement: 'three across, unevenly', minimum: 400, room: '600px' },
  play: async ({ canvasElement }) => {
    const widths = () =>
      [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(
        (pane) => pane.getBoundingClientRect().width,
      )
    await expect(widths()).toHaveLength(3)

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
