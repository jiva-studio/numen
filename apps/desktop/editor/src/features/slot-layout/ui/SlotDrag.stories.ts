/**
 * The Cmd/Ctrl+drag of the entries of the bars, over two bars drawn the way the bar plugins mark
 * them. Where a drop lands depends on where the entries are drawn, which needs a browser. Every
 * plugin and entry here is invented.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, waitFor } from 'storybook/test'
import { defineComponent, h, type PropType } from 'vue'
import { createSlotDrag } from '../model/slotDrag'

const STATS = { item: 'org.example.reef/view/stats', layout: 'org.example.reef/stats' }
const TASKS = { item: 'org.example.tide/view/status', layout: 'org.example.tide/status' }
const CLOCK = {
  item: 'org.example.harbor/command/org.example.harbor:clock.open',
  layout: 'org.example.harbor/clock.open',
}

const LAYOUT_KEYS: Readonly<Record<string, string>> = {
  [STATS.item]: STATS.layout,
  [TASKS.item]: TASKS.layout,
  [CLOCK.item]: CLOCK.layout,
}

const DRAWN: Readonly<Record<string, readonly string[]>> = {
  'titleBar.commands': [],
  'titleBar.status': [STATS.layout, TASKS.layout],
  'statusBar.left': [],
  'statusBar.right': [CLOCK.layout],
}

type Run = (entry: string) => void
type Write = (change: unknown) => Promise<void>

const drawEntry = (key: string, name: string, onRun: Run) =>
  h('div', { 'data-numen-slot-entry': key, style: 'display: flex' }, [
    h(
      'button',
      {
        type: 'button',
        'data-entry': name,
        style: 'inline-size: 80px; block-size: 24px',
        onClick: () => onRun(name),
      },
      name,
    ),
  ])

/** Two bars marked as the bar plugins mark them, with the drag of the window over them. */
const Bars = defineComponent({
  props: {
    isMac: { type: Boolean, default: false },
    onRun: { type: Function as PropType<Run>, required: true },
    writeLayout: { type: Function as PropType<Write>, required: true },
  },
  setup(props) {
    createSlotDrag({
      isMac: () => props.isMac,
      findLayoutKey: (itemKey) => LAYOUT_KEYS[itemKey],
      listLayoutKeys: (slot) => DRAWN[slot] ?? [],
      getLayout: () => ({}),
      writeLayout: (change) => props.writeLayout(change),
    })
    const row = 'display: flex; align-items: center; gap: 4px; block-size: 32px'
    return () =>
      h(
        'div',
        {
          style:
            'display: flex; flex-direction: column; gap: 24px; inline-size: 600px; padding: 16px',
        },
        [
          h('header', { style: row }, [
            h('div', {
              'data-numen-slot': 'titleBar.commands',
              style: `${row}; min-inline-size: 40px`,
            }),
            h('div', { 'data-numen-slot': 'titleBar.status', style: row }, [
              drawEntry(STATS.item, 'stats', props.onRun),
              drawEntry(TASKS.item, 'tasks', props.onRun),
            ]),
          ]),
          h('main', { 'data-outside': '', style: 'block-size: 120px' }, 'The editor'),
          h('footer', { style: `${row}; justify-content: space-between` }, [
            h('div', {
              'data-numen-slot': 'statusBar.left',
              style: `${row}; min-inline-size: 120px`,
            }),
            h('div', { 'data-numen-slot': 'statusBar.right', style: row }, [
              drawEntry(CLOCK.item, 'clock', props.onRun),
            ]),
          ]),
        ],
      )
  },
})

const meta = {
  title: 'Window/Slot Drag',
  component: Bars,
  args: { isMac: false, onRun: fn(), writeLayout: fn(() => Promise.resolve()) },
} satisfies Meta<typeof Bars>

export default meta
type Story = StoryObj<typeof meta>

interface Point {
  readonly x: number
  readonly y: number
}

interface Keys {
  readonly ctrlKey?: boolean
  readonly metaKey?: boolean
}

const getEntry = (canvas: HTMLElement, name: string): HTMLElement =>
  canvas.querySelector<HTMLElement>(`[data-entry="${name}"]`)!

const getPart = (canvas: HTMLElement, selector: string): HTMLElement =>
  canvas.querySelector<HTMLElement>(selector)!

const getCenter = (element: Element): Point => {
  const box = element.getBoundingClientRect()
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 }
}

/** A point over the left quarter of the element: a drop there goes before it. */
const getLeftQuarter = (element: Element): Point => {
  const box = element.getBoundingClientRect()
  return { x: box.left + box.width / 4, y: box.top + box.height / 2 }
}

const sendPointer = (type: string, at: Point, keys: Keys): PointerEvent => {
  const event = new PointerEvent(type, {
    bubbles: true,
    cancelable: true,
    composed: true,
    pointerId: 1,
    pointerType: 'mouse',
    isPrimary: true,
    button: 0,
    buttons: type === 'pointerup' ? 0 : 1,
    clientX: at.x,
    clientY: at.y,
    ...keys,
  })
  ;(document.elementFromPoint(at.x, at.y) ?? document.body).dispatchEvent(event)
  return event
}

const moveTo = (from: Point, to: Point, keys: Keys): void => {
  for (let step = 1; step <= 6; step++) {
    const part = step / 6
    sendPointer(
      'pointermove',
      { x: from.x + (to.x - from.x) * part, y: from.y + (to.y - from.y) * part },
      keys,
    )
  }
}

/** Presses the entry, moves past any threshold to the point and releases there; the pointerdown is returned. */
const dragTo = (entry: HTMLElement, to: Point, keys: Keys): PointerEvent => {
  const from = getCenter(entry)
  const down = sendPointer('pointerdown', from, keys)
  moveTo(from, to, keys)
  sendPointer('pointerup', to, keys)
  return down
}

const click = (entry: HTMLElement, keys: Keys): void => {
  entry.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, ...keys }))
}

const isDragging = (): boolean => document.documentElement.hasAttribute('data-numen-slot-dragging')

/** Two frames pass, so a write or a run that would follow a release has had its turn. */
const settle = async (): Promise<void> => {
  for (let frame = 0; frame < 2; frame++) await new Promise((done) => requestAnimationFrame(done))
}

const CTRL: Keys = { ctrlKey: true }
const META: Keys = { metaKey: true }

/** On Linux, Ctrl+drag of the second entry over the left half of the first puts it first, with one write. */
export const CtrlDragReordersASlot: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')
    const from = getCenter(tasks)
    const to = getLeftQuarter(getEntry(canvasElement, 'stats'))

    const down = sendPointer('pointerdown', from, CTRL)
    moveTo(from, to, CTRL)
    await expect(down.defaultPrevented).toBe(true)
    await expect(isDragging()).toBe(true)
    sendPointer('pointerup', to, CTRL)

    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(1))
    await expect(args.writeLayout).toHaveBeenCalledWith({
      'titleBar.status': [TASKS.layout, STATS.layout],
    })
    await waitFor(() => expect(isDragging()).toBe(false))
  },
}

/** On Linux the Meta key starts no drag: the press is the entry's own. */
export const MetaDragOnLinuxIsAPress: Story = {
  play: async ({ canvasElement, args }) => {
    const down = dragTo(
      getEntry(canvasElement, 'tasks'),
      getLeftQuarter(getEntry(canvasElement, 'stats')),
      META,
    )
    await settle()

    await expect(down.defaultPrevented).toBe(false)
    await expect(isDragging()).toBe(false)
    await expect(args.writeLayout).not.toHaveBeenCalled()
  },
}

/** On macOS, Cmd+drag moves the entry. */
export const CmdDragOnMacReordersASlot: Story = {
  args: { isMac: true },
  play: async ({ canvasElement, args }) => {
    dragTo(getEntry(canvasElement, 'tasks'), getLeftQuarter(getEntry(canvasElement, 'stats')), META)

    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(1))
    await expect(args.writeLayout).toHaveBeenCalledWith({
      'titleBar.status': [TASKS.layout, STATS.layout],
    })
  },
}

/** A press without the modifier is left to the entry, however far it moves, and its click runs the entry. */
export const PlainPressIsLeftToTheEntry: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')
    const from = getCenter(tasks)
    const to = getLeftQuarter(getEntry(canvasElement, 'stats'))

    const down = sendPointer('pointerdown', from, {})
    moveTo(from, to, {})
    await expect(isDragging()).toBe(false)
    sendPointer('pointerup', to, {})
    click(tasks, {})
    await settle()

    await expect(down.defaultPrevented).toBe(false)
    await expect(args.writeLayout).not.toHaveBeenCalled()
    await expect(args.onRun).toHaveBeenCalledWith('tasks')
  },
}

/** A Ctrl press released before the drag starts leaves the click to the entry, which runs. */
export const CtrlPressWithoutMovementRunsTheEntry: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')
    const at = getCenter(tasks)

    sendPointer('pointerdown', at, CTRL)
    sendPointer('pointerup', at, CTRL)
    click(tasks, CTRL)
    await settle()

    await expect(isDragging()).toBe(false)
    await expect(args.writeLayout).not.toHaveBeenCalled()
    await expect(args.onRun).toHaveBeenCalledWith('tasks')
  },
}

/** The click the browser sends after a drop does not run the dragged entry. */
export const ClickAfterADropDoesNothing: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')

    dragTo(tasks, getLeftQuarter(getEntry(canvasElement, 'stats')), CTRL)
    click(tasks, CTRL)
    await settle()

    await expect(args.writeLayout).toHaveBeenCalledTimes(1)
    await expect(args.onRun).not.toHaveBeenCalled()
  },
}

/** Escape ends a drag with no write, and the release that follows writes nothing. */
export const EscapeCancelsADrag: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')
    const from = getCenter(tasks)
    const to = getLeftQuarter(getEntry(canvasElement, 'stats'))

    sendPointer('pointerdown', from, CTRL)
    moveTo(from, to, CTRL)
    await expect(isDragging()).toBe(true)
    document.body.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }),
    )
    await waitFor(() => expect(isDragging()).toBe(false))
    sendPointer('pointerup', to, CTRL)
    await settle()

    await expect(args.writeLayout).not.toHaveBeenCalled()
  },
}

/** A release outside the four bar slots ends the drag with no write. */
export const DropOutsideTheBarsWritesNothing: Story = {
  play: async ({ canvasElement, args }) => {
    const tasks = getEntry(canvasElement, 'tasks')
    const from = getCenter(tasks)
    const to = getCenter(getPart(canvasElement, '[data-outside]'))

    sendPointer('pointerdown', from, CTRL)
    moveTo(from, to, CTRL)
    await expect(isDragging()).toBe(true)
    sendPointer('pointerup', to, CTRL)
    await settle()

    await expect(isDragging()).toBe(false)
    await expect(args.writeLayout).not.toHaveBeenCalled()
  },
}

/** A drop on an empty slot moves the entry there, and the one write holds both lists. */
export const DropOnAnEmptySlot: Story = {
  play: async ({ canvasElement, args }) => {
    dragTo(
      getEntry(canvasElement, 'tasks'),
      getCenter(getPart(canvasElement, '[data-numen-slot="statusBar.left"]')),
      CTRL,
    )

    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(1))
    await expect(args.writeLayout).toHaveBeenCalledWith({
      'titleBar.status': [STATS.layout],
      'statusBar.left': [TASKS.layout],
    })
  },
}

/** A drop over the left half of an entry of another slot puts the dragged entry before it. */
export const DropBeforeAnEntryOfAnotherSlot: Story = {
  play: async ({ canvasElement, args }) => {
    dragTo(getEntry(canvasElement, 'tasks'), getLeftQuarter(getEntry(canvasElement, 'clock')), CTRL)

    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(1))
    await expect(args.writeLayout).toHaveBeenCalledWith({
      'titleBar.status': [STATS.layout],
      'statusBar.right': [TASKS.layout, CLOCK.layout],
    })
  },
}

/**
 * A write the settings store answers with a failure: the store reports it and reads the file again,
 * and its promise settles. The drag ends, the bars stay as the file draws them, and the next drag
 * writes again.
 */
export const FailedWriteEndsTheDrag: Story = {
  args: { writeLayout: fn(() => new Promise<void>((done) => requestAnimationFrame(() => done()))) },
  play: async ({ canvasElement, args }) => {
    const order = () =>
      [...canvasElement.querySelectorAll('[data-entry]')].map((one) =>
        one.getAttribute('data-entry'),
      )

    dragTo(getEntry(canvasElement, 'tasks'), getLeftQuarter(getEntry(canvasElement, 'stats')), CTRL)
    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(isDragging()).toBe(false))
    await expect(order()).toStrictEqual(['stats', 'tasks', 'clock'])

    dragTo(getEntry(canvasElement, 'tasks'), getLeftQuarter(getEntry(canvasElement, 'stats')), CTRL)
    await waitFor(() => expect(args.writeLayout).toHaveBeenCalledTimes(2))
  },
}
