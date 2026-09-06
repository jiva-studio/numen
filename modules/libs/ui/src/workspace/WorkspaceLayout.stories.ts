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
import WorkspaceLayout from './WorkspaceLayout.vue'
import TabStub from './fixtures/TabStub.vue'
import type { Tab, Workspace as State } from './node'
import { panesOf } from './tree'
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
  /** What each tab is carrying, by tab. */
  marks: Readonly<Record<string, string>>
  /** Given by the story, and nothing a reader turns. */
  tabs?: never
  naming?: never
  modelValue?: never
}

const meta: Meta<Knobs> = {
  title: 'Workspace',
  component: WorkspaceLayout,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    arrangement: {
      control: 'select',
      options: Object.keys(ARRANGEMENTS),
      description: 'What the workspace starts as. Changing it starts afresh.',
    },
    edge: { control: { type: 'range', min: 4, max: 80, step: 2 } },
    threshold: { control: { type: 'range', min: 0, max: 24, step: 1 } },
    marks: { control: 'object' },
    tabs: { table: { disable: true } },
    naming: { table: { disable: true } },
    modelValue: { table: { disable: true } },
  },
  args: { arrangement: 'side by side', edge: 22, threshold: 4, marks: {} },
  render: (args) => ({
    components: { WorkspaceLayout, TabStub },
    setup() {
      const held = ref<State>(ARRANGEMENTS[args.arrangement]())
      watch(
        () => args.arrangement,
        (next) => {
          held.value = ARRANGEMENTS[next]()
        },
      )
      const tabs = () => named(held.value, args.marks)

      return { held, args, tabs }
    },
    template: `
      <div style="height: 100vh; padding: 0">
        <WorkspaceLayout
          v-model="held"
          :tabs="tabs()"
          :edge="args.edge"
          :threshold="args.threshold"
          :naming="() => 'made-' + Math.random().toString(36).slice(2, 8)"
        >
          <template #tab="{ id }"><TabStub :name="id" /></template>
        </WorkspaceLayout>
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

/** A workspace holding nothing: one pane, no strip, and the silence. */
export const Empty: Story = { args: { arrangement: 'empty' } }

/** Tabs carrying something: work not yet saved, and a tab that is stuck. */
export const Marked: Story = {
  args: { arrangement: 'crowded', marks: { one: 'unsaved', three: 'stuck' } },
}

/** One tab left in the workspace. */
export const Alone: Story = { args: { arrangement: 'alone' } }

const rectOf = (element: Element) => element.getBoundingClientRect()

const tabIn = (canvas: HTMLElement, tab: string) => {
  const found = canvas.querySelector(`[data-workspace-tab="${tab}"]`)
  if (!found) throw new Error(`no tab called ${tab}`)
  return found
}

const paneBox = (canvas: HTMLElement, pane: string) => {
  const found = canvas.querySelector(`[data-workspace-pane="${pane}"]`)
  if (!found) throw new Error(`no pane called ${pane}`)
  return rectOf(found)
}

interface Position {
  readonly x: number
  readonly y: number
}

/**
 * A place of the story told to the window that drives the pointer, and where
 * that whole pixel of the window falls back in the story. The story is drawn
 * at a scale, and the pointer goes to whole pixels of the window.
 */
const framedIn = (at: Position): { window: Position; story: Position } => {
  const frame = window.frameElement as HTMLElement | null
  if (!frame) {
    const whole = { x: Math.round(at.x), y: Math.round(at.y) }
    return { window: whole, story: whole }
  }

  const box = frame.getBoundingClientRect()
  const across = box.width / window.innerWidth
  const down = box.height / window.innerHeight
  const driven = {
    x: Math.round(box.x + at.x * across),
    y: Math.round(box.y + at.y * down),
  }
  return {
    window: driven,
    story: { x: (driven.x - box.x) / across, y: (driven.y - box.y) / down },
  }
}

/**
 * A drag the browser makes itself, from one place to another. The splitter
 * catches the pointer by where it is, which is something only a browser says.
 */
const swept = async (from: Position, to: Position): Promise<void> => {
  const context = await import('vitest/browser')
  await context.commands.sweep(framedIn(from).window, framedIn(to).window)
  await new Promise((done) => setTimeout(done, 16))
}

/** The cursor the splitter puts over the whole page while it has the pointer. */
function heldCursor(): string | null {
  const put = '*{cursor:'

  for (const style of document.head.querySelectorAll('style')) {
    const text = (style.textContent ?? '').trim()
    if (text.startsWith(put)) return text.slice(put.length, text.indexOf('!')).trim()
  }
  return null
}

/** A place the pointer was at, and the cursor the splitter drew there. */
interface Caught {
  readonly away: number
  readonly at: Position
  readonly cursor: string | null
}

/**
 * How far either side of the line the pointer is put, in pixels. The reach is a
 * rectangle seven pixels out, so anywhere between two places it catches is
 * caught too; what is worth asking about is where it ends. The line itself, the
 * panel either side of it, the far edge of the reach a pixel in for where the
 * window rounds the pointer to, and clear of it.
 */
const AWAY = [-10, -6, -3, 0, 3, 6, 10]

/**
 * Where the splitter has the pointer along a line across a handle, and the
 * cursor it draws there. The line is taken a quarter of the way along the
 * handle, clear of the handles a branch further in lays across this one.
 */
async function caughtAcross(handle: HTMLElement, along: 'x' | 'y'): Promise<readonly Caught[]> {
  const box = rectOf(handle)
  const found: Caught[] = []

  for (const away of AWAY) {
    const at =
      along === 'x'
        ? { x: box.x + box.width / 2 + away, y: box.y + box.height / 4 }
        : { x: box.x + box.width / 4, y: box.y + box.height / 2 + away }

    await swept(at, at)
    found.push({ away, at, cursor: heldCursor() })
  }
  return found
}

/** What a handle that catches its whole reach, and no further, draws at those places. */
const reaching = (cursor: string) => AWAY.map((away) => [away, Math.abs(away) <= 6 ? cursor : null])

/** The furthest out along that line the splitter still had the pointer. */
function furthest(caught: readonly Caught[]): Position {
  const held = caught.filter((place) => place.cursor)
  const edge = held[held.length - 1]?.at
  if (!edge) throw new Error('the handle caught nothing')
  return edge
}

/** A tab picked up and let go somewhere, in as many steps as a hand takes. */
async function dragTo(from: Element, to: { x: number; y: number }): Promise<void> {
  const start = rectOf(from)
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
    const boxes = now.map(rectOf)
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
      rectOf(pane).width,
    ])

    const aside = paneBox(canvasElement, 'aside')
    await dragTo(tabIn(canvasElement, 'chat'), {
      x: aside.x + aside.width / 2,
      y: aside.y + aside.height / 2,
    })

    const after = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map((pane) => [
      pane.getAttribute('data-workspace-pane'),
      rectOf(pane).width,
    ])
    await expect(after).toStrictEqual(before)
  },
}

/** Dragging to the outer edge divides the whole workspace. */
export const DividesTheWorkspace: Story = {
  tags: ['!dev'],
  args: { arrangement: 'crowded' },
  play: async ({ canvasElement }) => {
    const frame = rectOf(canvasElement.querySelector('.workspace') as Element)

    await dragTo(tabIn(canvasElement, 'two'), {
      x: frame.x + frame.width / 2,
      y: frame.bottom - 6,
    })

    const panes = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(rectOf)
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
    const first = rectOf(tabIn(canvasElement, 'one'))

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
    await expect(rectOf(panes[0] as Element).width).toBeGreaterThan(main.width)
  },
}

/** Closing the last tab in the workspace leaves one pane, holding nothing. */
export const ClosesTheLastTab: Story = {
  tags: ['!dev'],
  args: { arrangement: 'alone' },
  play: async ({ canvasElement }) => {
    const close = within(tabIn(canvasElement, 'plex') as HTMLElement).getByRole('button')

    await userEvent.click(close)

    await expect(canvasElement.querySelectorAll('[data-workspace-pane]')).toHaveLength(1)
    await expect(canvasElement.querySelectorAll('[data-workspace-tab]')).toHaveLength(0)
    await expect(canvasElement.querySelector('[data-workspace-strip]')).toBeNull()
    await expect(within(canvasElement).getByText('Nothing open')).toBeInTheDocument()
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

/** The shares the model holds are the shares that are drawn. */
export const FollowsTheModel: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const drawn = [...canvasElement.querySelectorAll('[data-workspace-pane]')].map(rectOf)
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
    const at = rectOf(handle)
    const from = { clientX: at.x + at.width / 2, clientY: at.y + at.height / 2 }

    /** The branch being resized, while the pointer is on its way. */
    const resizing = () => canvasElement.querySelector('.branch[data-resizing]')

    let marked = false
    const watch = () => {
      marked ||= resizing() !== null
    }
    window.addEventListener('pointermove', watch, true)

    await expect(resizing()).toBeNull()

    await userEvent.pointer([
      { keys: '[MouseLeft>]', target: handle, coords: from },
      { coords: { clientX: from.clientX - 120, clientY: from.clientY } },
      { coords: { clientX: from.clientX + 160, clientY: from.clientY + 40 } },
      { keys: '[/MouseLeft]' },
    ])

    window.removeEventListener('pointermove', watch, true)

    await expect(marked).toBe(true)
    await expect(resizing()).toBeNull()
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
    const at = rectOf(handle)
    const middle = at.y + at.height / 2

    for (const away of [-4, 4]) {
      const x = at.x + at.width / 2 + away
      await expect(document.elementFromPoint(x, middle)).toBe(handle)
    }
  },
}

/**
 * The splitter has the pointer over the whole reach the handle draws, one
 * cursor throughout, and a press out at the far edge of that reach carries the
 * split with it.
 *
 * Only a browser can answer it: the splitter takes the pointer by where it is,
 * which a made-up event does not say.
 */
export const DragsFromItsWholeReach: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const handle = canvasElement.querySelector('.branch__handle') as HTMLElement
    const caught = await caughtAcross(handle, 'x')

    await expect(caught.map((place) => [place.away, place.cursor])).toStrictEqual(
      reaching('ew-resize'),
    )
    await expect(getComputedStyle(handle).cursor).toBe('ew-resize')

    const edge = furthest(caught)
    const before = paneBox(canvasElement, 'main').width

    await swept(edge, { x: edge.x + 40, y: edge.y })

    // The pointer goes to whole pixels of the window, which the story counts in
    // its own.
    const moved = paneBox(canvasElement, 'main').width - before
    await expect(Math.abs(moved - 40)).toBeLessThan(2)
  },
}

/** A handle lying across a branch is caught over its whole reach as well. */
export const DragsFromItsWholeReachDownwards: Story = {
  tags: ['!dev'],
  args: { arrangement: 'nested' },
  play: async ({ canvasElement }) => {
    const handle = canvasElement.querySelector(
      '.branch__handle[data-direction="vertical"]',
    ) as HTMLElement
    const caught = await caughtAcross(handle, 'y')

    await expect(caught.map((place) => [place.away, place.cursor])).toStrictEqual(
      reaching('ns-resize'),
    )
    await expect(getComputedStyle(handle).cursor).toBe('ns-resize')

    const edge = furthest(caught)
    const before = paneBox(canvasElement, 'b').height

    await swept(edge, { x: edge.x, y: edge.y + 30 })

    const moved = paneBox(canvasElement, 'b').height - before
    await expect(Math.abs(moved - 30)).toBeLessThan(2)
  },
}
