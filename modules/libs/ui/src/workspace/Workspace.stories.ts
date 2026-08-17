/**
 * Every situation the workspace has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the tabs hold is a plain panel with a name on it. The workspace knows
 * nothing about its contents, and a story that reached for the plex or the
 * agent would be the door that dependency comes back in through.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref, watch } from 'vue'
import Workspace from './Workspace.vue'
import Filling from './fixtures/Filling.vue'
import { arrangeWorkspace } from './arrange'
import { groupWithTab, groupsOf, isBranch, type TabLabel, type Workspace as State } from './model'
import { crowded, deep, empty, oneStack, sideBySide } from './fixtures/build'

const TITLES: Readonly<Record<string, string>> = {
  plex: 'Plex',
  chat: 'Chat',
  notes: 'Notes',
  one: 'One',
  two: 'Two',
  three: 'Three',
  four: 'Four',
  five: 'Five',
  six: 'Six',
  seven: 'Seven',
  eight: 'Eight',
}

const labelled = (state: State): readonly TabLabel[] =>
  groupsOf(state.root)
    .flatMap((group) => group.tabs)
    .map((id) => ({ id, title: TITLES[id] ?? id }))

interface Knobs {
  workspace: State
  /** How close to the outer edge divides the whole workspace. */
  edge: number
}

const meta: Meta<Knobs> = {
  title: 'Workspace',
  component: Workspace,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    edge: { control: { type: 'range', min: 4, max: 80, step: 2 } },
    workspace: { control: false },
  },
  args: { workspace: sideBySide(), edge: 22 },
  render: (args) => ({
    components: { Workspace, Filling },
    setup() {
      const held = ref<State>(args.workspace)
      // The knob is the source of a fresh arrangement; after that the
      // workspace is, and the story follows it.
      watch(
        () => args.workspace,
        (next) => {
          held.value = next
        },
      )
      const titles = () => labelled(held.value)
      return { held, args, titles }
    },
    template: `
      <div style="height: 100vh; padding: 0">
        <Workspace
          v-model="held"
          :tabs="titles()"
          :edge="args.edge"
          :naming="{ id: () => 'made-' + Math.random().toString(36).slice(2, 8) }"
        >
          <template #tab="{ id }"><Filling :name="id" /></template>
        </Workspace>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** What the desktop opens with. */
export const SideBySide: Story = {}

/** Both in one stack, so that only one shows at a time. */
export const OneStack: Story = { args: { workspace: oneStack() } }

/** Four levels, each turning a quarter from the one above. */
export const Nested: Story = { args: { workspace: deep() } }

/** More tabs than a strip has room for. */
export const Crowded: Story = { args: { workspace: crowded() } }

/** The last tab closed. */
export const Empty: Story = { args: { workspace: empty() } }

const boxOf = (element: Element) => element.getBoundingClientRect()

const tabIn = (canvas: HTMLElement, tab: string) => {
  const found = canvas.querySelector(`[data-workspace-tab="${tab}"]`)
  if (!found) throw new Error(`no tab called ${tab}`)
  return found
}

const groupBox = (canvas: HTMLElement, group: string) => {
  const found = canvas.querySelector(`[data-workspace-group="${group}"]`)
  if (!found) throw new Error(`no group called ${group}`)
  return boxOf(found)
}

/** A tab picked up and let go somewhere, in as many steps as a hand takes. */
async function dragTo(from: Element, to: { x: number; y: number }): Promise<void> {
  const start = boxOf(from)
  const at = { clientX: start.x + start.width / 2, clientY: start.y + start.height / 2 }

  await userEvent.pointer([
    { keys: '[MouseLeft>]', target: from, coords: at },
    { coords: { clientX: (at.clientX + to.x) / 2, clientY: (at.clientY + to.y) / 2 } },
    { coords: { clientX: to.x, clientY: to.y } },
    { keys: '[/MouseLeft]' },
  ])
}

/** Dragging a tab to the right edge of a group divides that group. */
export const DividesOnAnEdge: Story = {
  play: async ({ canvasElement }) => {
    const before = groupBox(canvasElement, 'aside')
    const main = groupBox(canvasElement, 'main')

    await dragTo(tabIn(canvasElement, 'chat'), {
      x: main.right - main.width * 0.05,
      y: main.y + main.height / 2,
    })

    // The group it came from held nothing else, so that group went and a fresh
    // one holds the chat.
    const now = [...canvasElement.querySelectorAll('[data-workspace-group]')]
    const named = now.map((group) => group.getAttribute('data-workspace-group'))

    await expect(now).toHaveLength(2)
    await expect(named).toContain('main')
    await expect(named).not.toContain('aside')

    // The two share the room the two before them had, and evenly.
    const boxes = now.map(boxOf)
    expect(boxes.reduce((wide, box) => wide + box.width, 0)).toBeCloseTo(
      main.width + before.width,
      0,
    )
    expect(boxes[0]?.width).toBeCloseTo(boxes[1]?.width ?? 0, 0)
  },
}

/** Dragging a tab into the middle of another group joins its stack. */
export const JoinsAStack: Story = {
  play: async ({ canvasElement }) => {
    const main = groupBox(canvasElement, 'main')

    await dragTo(tabIn(canvasElement, 'chat'), {
      x: main.x + main.width / 2,
      y: main.y + main.height / 2,
    })

    await expect(canvasElement.querySelectorAll('[data-workspace-group]')).toHaveLength(1)
    await expect(canvasElement.querySelectorAll('[data-workspace-tab]')).toHaveLength(2)
  },
}

/** Letting a tab go where it was picked up changes nothing. */
export const StaysPut: Story = {
  args: { workspace: sideBySide() },
  play: async ({ canvasElement }) => {
    const before = [...canvasElement.querySelectorAll('[data-workspace-group]')].map((group) => [
      group.getAttribute('data-workspace-group'),
      boxOf(group).width,
    ])

    const aside = groupBox(canvasElement, 'aside')
    await dragTo(tabIn(canvasElement, 'chat'), {
      x: aside.x + aside.width / 2,
      y: aside.y + aside.height / 2,
    })

    const after = [...canvasElement.querySelectorAll('[data-workspace-group]')].map((group) => [
      group.getAttribute('data-workspace-group'),
      boxOf(group).width,
    ])
    await expect(after).toStrictEqual(before)
  },
}

/** Dragging to the outer edge divides the whole workspace. */
export const DividesTheWorkspace: Story = {
  args: { workspace: crowded() },
  play: async ({ canvasElement }) => {
    const frame = boxOf(canvasElement.querySelector('.workspace') as Element)

    await dragTo(tabIn(canvasElement, 'two'), {
      x: frame.x + frame.width / 2,
      y: frame.bottom - 6,
    })

    const groups = [...canvasElement.querySelectorAll('[data-workspace-group]')].map(boxOf)
    await expect(groups).toHaveLength(3)

    // One of them runs the whole width along the foot.
    const along = groups.filter((box) => Math.abs(box.width - frame.width) < 2)
    await expect(along).toHaveLength(1)
  },
}

/** A tab dropped on a strip takes its place among the others. */
export const ReordersInAStrip: Story = {
  args: { workspace: crowded() },
  play: async ({ canvasElement }) => {
    const first = boxOf(tabIn(canvasElement, 'one'))

    await dragTo(tabIn(canvasElement, 'three'), { x: first.x + 2, y: first.y + first.height / 2 })

    const strip = canvasElement.querySelector('[data-workspace-strip="main"]') as HTMLElement
    const order = [...strip.querySelectorAll('[data-workspace-tab]')].map((tab) =>
      tab.getAttribute('data-workspace-tab'),
    )
    await expect(order.slice(0, 3)).toStrictEqual(['three', 'one', 'two'])
  },
}

/** Closing the last tab of a group clears the group away. */
export const ClosesAGroup: Story = {
  play: async ({ canvasElement }) => {
    const main = groupBox(canvasElement, 'main')
    const close = within(tabIn(canvasElement, 'chat') as HTMLElement).getByRole('button')

    await userEvent.click(close)

    const groups = [...canvasElement.querySelectorAll('[data-workspace-group]')]
    await expect(groups).toHaveLength(1)
    await expect(boxOf(groups[0] as Element).width).toBeGreaterThan(main.width)
  },
}

/** The model is what is drawn: change it, and the picture follows. */
export const FollowsTheModel: Story = {
  play: async ({ canvasElement }) => {
    const state = sideBySide()
    const boxes = arrangeWorkspace(state, { x: 0, y: 0, width: 1000, height: 600 })

    // What the model says of itself, before any of it is drawn.
    await expect(boxes.get('main')?.width).toBe(720)
    await expect(groupWithTab(state.root, 'chat')?.id).toBe('aside')
    await expect(isBranch(state.root)).toBe(true)

    const drawn = [...canvasElement.querySelectorAll('[data-workspace-group]')].map(boxOf)
    const total = drawn.reduce((wide, box) => wide + box.width, 0)
    await expect((drawn[0]?.width ?? 0) / total).toBeCloseTo(0.72, 1)
  },
}
