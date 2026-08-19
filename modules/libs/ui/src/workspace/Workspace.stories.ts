/**
 * Every situation the workspace has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the tabs hold is a plain panel with a name on it. The workspace knows
 * nothing about its contents, and neither do these.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref, watch } from 'vue'
import Workspace from './Workspace.vue'
import Filling from './fixtures/Filling.vue'
import { openTab } from './edit'
import { panesOf, type Tab, type Workspace as State } from './model'
import { crowded, deep, empty, oneStack, sideBySide, stack, workspaceOf } from './fixtures/build'

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

const named = (state: State, marks: Readonly<Record<string, string>>): readonly Tab[] =>
  panesOf(state.root)
    .flatMap((pane) => pane.tabs)
    .map((id) => ({ id, title: TITLES[id] ?? id, ...(marks[id] ? { mark: marks[id] } : {}) }))

/** The arrangements a reader can start from. */
const ARRANGEMENTS = {
  'side by side': sideBySide,
  'one stack': oneStack,
  nested: deep,
  crowded,
  empty,
  alone: () => workspaceOf(stack('main', 'plex')),
} satisfies Record<string, () => State>

type Arrangement = keyof typeof ARRANGEMENTS

interface Knobs {
  arrangement: Arrangement
  edge: number
  threshold: number
  /** What the way to a new tab is called. Emptied, no strip offers one. */
  newTab: string
  /** What each tab is carrying, by tab. */
  marks: Readonly<Record<string, string>>
  /** Given by the story, and nothing a reader turns. */
  tabs?: never
  naming?: never
  modelValue?: never
}

const meta: Meta<Knobs> = {
  title: 'Workspace',
  component: Workspace,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    arrangement: {
      control: 'select',
      options: Object.keys(ARRANGEMENTS),
      description: 'What the workspace starts as. Changing it starts afresh.',
    },
    edge: { control: { type: 'range', min: 4, max: 80, step: 2 } },
    threshold: { control: { type: 'range', min: 0, max: 24, step: 1 } },
    newTab: { control: 'text' },
    marks: { control: 'object' },
    tabs: { table: { disable: true } },
    naming: { table: { disable: true } },
    modelValue: { table: { disable: true } },
  },
  args: { arrangement: 'side by side', edge: 22, threshold: 4, newTab: 'New tab', marks: {} },
  render: (args) => ({
    components: { Workspace, Filling },
    setup() {
      const held = ref<State>(ARRANGEMENTS[args.arrangement]())
      watch(
        () => args.arrangement,
        (next) => {
          held.value = ARRANGEMENTS[next]()
        },
      )
      const tabs = () => named(held.value, args.marks)

      /** The application's part: the workspace says where, and this says what. */
      let made = 0
      const open = (pane: string) => {
        held.value = openTab(held.value, `Untitled ${++made}`, pane)
      }

      return { held, args, tabs, open }
    },
    template: `
      <div style="height: 100vh; padding: 0">
        <Workspace
          v-model="held"
          :tabs="tabs()"
          :edge="args.edge"
          :threshold="args.threshold"
          :new-tab="args.newTab"
          :naming="() => 'made-' + Math.random().toString(36).slice(2, 8)"
          @open="open"
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
export const OneStack: Story = { args: { arrangement: 'one stack' } }

/** Four levels, each turning a quarter from the one above. */
export const Nested: Story = { args: { arrangement: 'nested' } }

/** More tabs than a strip has room for. */
export const Crowded: Story = { args: { arrangement: 'crowded' } }

/** The last tab closed. */
export const Empty: Story = { args: { arrangement: 'empty' } }

/** Tabs carrying something: work not yet saved, and a tab that is stuck. */
export const Marked: Story = {
  args: { arrangement: 'crowded', marks: { one: 'unsaved', three: 'stuck' } },
}

/** One tab left in the workspace, which is offered no close. */
export const Alone: Story = { args: { arrangement: 'alone' } }


const boxOf = (element: Element) => element.getBoundingClientRect()

const tabIn = (canvas: HTMLElement, tab: string) => {
  const found = canvas.querySelector(`[data-workspace-tab="${tab}"]`)
  if (!found) throw new Error(`no tab called ${tab}`)
  return found
}

const paneBox = (canvas: HTMLElement, pane: string) => {
  const found = canvas.querySelector(`[data-workspace-pane="${pane}"]`)
  if (!found) throw new Error(`no pane called ${pane}`)
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

/** Dragging a tab to the right edge of a pane divides that pane. */
export const DividesOnAnEdge: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const before = paneBox(canvasElement, 'aside')
    const main = paneBox(canvasElement, 'main')

    await dragTo(tabIn(canvasElement, 'chat'), {
      x: main.right - main.width * 0.05,
      y: main.y + main.height / 2,
    })

    // The pane it came from held nothing else, so that pane went and a fresh
    // one holds the chat.
    const now = [...canvasElement.querySelectorAll('[data-workspace-pane]')]
    const named = now.map((pane) => pane.getAttribute('data-workspace-pane'))

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

/** Dragging a tab into the middle of another pane joins its stack. */
export const JoinsAStack: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const main = paneBox(canvasElement, 'main')

    await dragTo(tabIn(canvasElement, 'chat'), {
      x: main.x + main.width / 2,
      y: main.y + main.height / 2,
    })

    await expect(canvasElement.querySelectorAll('[data-workspace-pane]')).toHaveLength(1)
    await expect(canvasElement.querySelectorAll('[data-workspace-tab]')).toHaveLength(2)
  },
}

/** Letting a tab go where it was picked up changes nothing. */
export const StaysPut: Story = {
  tags: ['!dev'],
  args: { arrangement: 'side by side' },
  play: async ({ canvasElement }) => {
    const before = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map((pane) => [
      pane.getAttribute('data-workspace-pane'),
      boxOf(pane).width,
    ])

    const aside = paneBox(canvasElement, 'aside')
    await dragTo(tabIn(canvasElement, 'chat'), {
      x: aside.x + aside.width / 2,
      y: aside.y + aside.height / 2,
    })

    const after = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map((pane) => [
      pane.getAttribute('data-workspace-pane'),
      boxOf(pane).width,
    ])
    await expect(after).toStrictEqual(before)
  },
}

/** Dragging to the outer edge divides the whole workspace. */
export const DividesTheWorkspace: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
  play: async ({ canvasElement }) => {
    const frame = boxOf(canvasElement.querySelector('.workspace') as Element)

    await dragTo(tabIn(canvasElement, 'two'), {
      x: frame.x + frame.width / 2,
      y: frame.bottom - 6,
    })

    const panes = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(boxOf)
    await expect(panes).toHaveLength(3)

    // One of them runs the whole width along the foot.
    const along = panes.filter((box) => Math.abs(box.width - frame.width) < 2)
    await expect(along).toHaveLength(1)
  },
}

/** A tab dropped on a strip takes its place among the others. */
export const ReordersInAStrip: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
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

/** Closing the last tab of a pane clears the pane away. */
export const ClosesAPane: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const main = paneBox(canvasElement, 'main')
    const close = within(tabIn(canvasElement, 'chat') as HTMLElement).getByRole('button')

    await userEvent.click(close)

    const panes = [...canvasElement.querySelectorAll('[data-workspace-pane]')]
    await expect(panes).toHaveLength(1)
    await expect(boxOf(panes[0] as Element).width).toBeGreaterThan(main.width)
  },
}

/** The strip is walked with the arrows, and what is reached is shown. */
export const WalksWithTheKeyboard: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
  play: async ({ canvasElement }) => {
    const first = tabIn(canvasElement, 'one') as HTMLElement
    first.focus()

    await userEvent.keyboard('{ArrowRight}{ArrowRight}')

    const showing = canvasElement.querySelector('[data-workspace-tab][aria-selected="true"]')
    await expect(showing?.getAttribute('data-workspace-tab')).toBe('three')
    await expect(document.activeElement).toBe(tabIn(canvasElement, 'three'))

    await userEvent.keyboard('{Home}')
    await expect(document.activeElement).toBe(tabIn(canvasElement, 'one'))
  },
}

/**
 * A tab pressed takes the keyboard, so the strip is walked on from the tab the
 * hand chose.
 *
 * Only a browser can answer it: the press refuses the default the browser
 * would answer with, and the focus that comes with it is part of that default.
 */
export const WalksOnFromAPress: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
  play: async ({ canvasElement }) => {
    // Somewhere else entirely to start from, as a panel that was being typed in.
    const panel = canvasElement.querySelector('[role="tabpanel"][data-showing]') as HTMLElement
    panel.focus()

    await userEvent.click(tabIn(canvasElement, 'three'))
    await expect(document.activeElement).toBe(tabIn(canvasElement, 'three'))

    await userEvent.keyboard('{ArrowRight}')

    const showing = canvasElement.querySelector('[data-workspace-tab][aria-selected="true"]')
    await expect(showing?.getAttribute('data-workspace-tab')).toBe('four')
    await expect(document.activeElement).toBe(tabIn(canvasElement, 'four'))
  },
}

/** Escape in a panel comes back out to the tab the panel is held under. */
export const LeavesAPanel: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const panel = canvasElement.querySelector('[role="tabpanel"][data-showing]') as HTMLElement
    panel.focus()

    await userEvent.keyboard('{Escape}')

    await expect(document.activeElement).toBe(tabIn(canvasElement, 'plex'))
  },
}

/**
 * The way to one more tab, at the end of a strip already too full for its own.
 *
 * The workspace says which pane it was asked in and opens nothing: what a tab
 * holds is settled here, which is the application's part being played.
 */
export const AsksForANewTab: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
  play: async ({ canvasElement }) => {
    const strip = canvasElement.querySelector('[data-workspace-strip="main"]') as HTMLElement
    const asking = strip.querySelector('[data-workspace-new]') as HTMLElement

    // It keeps its whole width where the tabs beside it have given theirs up.
    await expect(boxOf(asking).width).toBeGreaterThan(0)
    await expect(boxOf(asking).right).toBeLessThanOrEqual(boxOf(strip).right + 1)
    await expect(asking.tabIndex).toBe(0)

    await userEvent.click(asking)

    const now = [...strip.querySelectorAll('[data-workspace-tab]')]
    await expect(now).toHaveLength(9)
    await expect(now[8]?.getAttribute('aria-selected')).toBe('true')
    await expect(within(now[8] as HTMLElement).getByTitle('Untitled 1')).toBeInTheDocument()

    // The arrows walk the tabs and stop at their own ends, so the last of
    // them steps round to the first.
    ;(now[8] as HTMLElement).focus()
    await userEvent.keyboard('{ArrowRight}')
    await expect(document.activeElement).toBe(now[0])
  },
}

/** The shares the model holds are the shares that are drawn. */
export const FollowsTheModel: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const drawn = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(boxOf)
    const total = drawn.reduce((wide, box) => wide + box.width, 0)
    await expect((drawn[0]?.width ?? 0) / total).toBeCloseTo(0.72, 1)
  },
}

/**
 * A handle dragged across the text moves the handle and selects nothing.
 *
 * A selection running through what the panels hold is one the engine works out
 * again for every movement of the pointer, which is what a person feels.
 */
export const SelectsNothingWhileResizing: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const handle = canvasElement.querySelector('.branch__handle')
    if (!handle) throw new Error('no handle')
    const at = boxOf(handle)
    const from = { clientX: at.x + at.width / 2, clientY: at.y + at.height / 2 }

    /** Whether the root was marked while the pointer was on its way. */
    let marked = false
    const watch = () => {
      marked ||= document.documentElement.hasAttribute('data-resizing')
    }
    window.addEventListener('pointermove', watch, true)

    await expect(document.documentElement.hasAttribute('data-resizing')).toBe(false)

    await userEvent.pointer([
      { keys: '[MouseLeft>]', target: handle, coords: from },
      { coords: { clientX: from.clientX - 120, clientY: from.clientY } },
      { coords: { clientX: from.clientX + 160, clientY: from.clientY + 40 } },
      { keys: '[/MouseLeft]' },
    ])

    window.removeEventListener('pointermove', watch, true)

    await expect(marked).toBe(true)
    await expect(document.documentElement.hasAttribute('data-resizing')).toBe(false)
    await expect(getSelection()?.toString() ?? '').toBe('')
  },
}

/**
 * The reach around a handle catches a press on either side of the line.
 *
 * The splitter answers a press from a few pixels away, and those pixels stand
 * over the panels. Whichever panel is drawn after the handle takes them unless
 * the reach stands above it, and what that panel holds then reads the press as
 * its own — an editor there starts a selection and runs it through its text for
 * the whole drag.
 */
export const CatchesAPressOnEitherSideOfTheLine: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const handle = canvasElement.querySelector('.branch__handle')
    if (!handle) throw new Error('no handle')
    const at = boxOf(handle)
    const middle = at.y + at.height / 2

    for (const away of [-4, 4]) {
      const x = at.x + at.width / 2 + away
      await expect(document.elementFromPoint(x, middle)).toBe(handle)
    }
  },
}
