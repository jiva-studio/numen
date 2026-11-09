/**
 * The tasks chip of the title bar in its states: ready, a task running, tasks failed. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`, which reads the size and
 * the contrast of the chip as drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { createTask } from '../testing/tasks'
import TasksStatus from './TasksStatus.vue'

const RUNNING = [
  createTask({ id: 'a', state: 'running', done: 3, total: 10 }),
  createTask({ id: 'b', state: 'running' }),
]
const FAILED = [createTask({ id: 'f', state: 'failed', error: 'The disk is full' })]

const meta = {
  title: 'Tasks/Status chip',
  component: TasksStatus,
  parameters: { layout: 'fullscreen' },
  args: { tasks: [] },
  render: (args) => ({
    components: { TasksStatus },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:center;justify-content:flex-end;block-size:2rem;background:var(--numen-titlebar);border-bottom:1px solid var(--numen-rule);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <span data-neutral-probe style="position: fixed; top: 400px; left: 0; inline-size: 4px; block-size: 4px; background: var(--numen-surface)"></span>
        <span data-alarm-probe style="position: fixed; top: 400px; left: 8px; inline-size: 4px; block-size: 4px; background: var(--numen-alarm-bg)"></span>
        <TasksStatus v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof TasksStatus>

export default meta
type Story = StoryObj<typeof meta>

const getChip = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement).getByRole('button')

const getGround = (element: Element): string => getComputedStyle(element).backgroundColor

const expectTone = async (canvasElement: HTMLElement, tone: 'neutral' | 'alarm') => {
  const probe = canvasElement.querySelector(`[data-${tone}-probe]`)!
  await expect(getGround(getChip(canvasElement))).toBe(getGround(probe))
}

const checkIdle: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.textContent?.trim()).toBe('Idle')
  await expect(chip.getAttribute('aria-label')).toBe('No tasks running')
  await expect(chip.getAnimations()).toHaveLength(0)
  await expect(chip.querySelector('.tasks-status__ring-fill')).toBeNull()
  await expect(getComputedStyle(chip).borderTopWidth).toBe('0px')
  await expect(chip.getBoundingClientRect().height).toBe(20)
  await expectTone(canvasElement, 'neutral')
}

const checkActive: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await waitFor(() => expect(chip.textContent?.trim()).toBe('0 of 2'), { timeout: 2000 })
  await expect(chip.getAttribute('aria-label')).toBe('0 of 2 tasks done')
  await expect(chip.getAnimations()).toHaveLength(0)
  await expect(chip.querySelector('.tasks-status__ring-fill')).not.toBeNull()
  await expectTone(canvasElement, 'neutral')
}

const checkFailed: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.getAttribute('aria-label')).toBe('1 failed')
  await expect(chip.textContent?.trim()).toBe('Idle')
  await expect(chip.querySelector('.tasks-status__badge')).toBeNull()
  await expectTone(canvasElement, 'alarm')
  await expect(getComputedStyle(chip).borderTopWidth).toBe('0px')
}

/** No task runs: an empty ring and Idle, on the light tokens. */
export const IdleLight: Story = { globals: { theme: 'light' }, play: checkIdle }

/** The same on the dark tokens. */
export const IdleDark: Story = { globals: { theme: 'dark' }, play: checkIdle }

/** Tasks run: the ring is filled by the progress and the chip counts the tasks done, on the light tokens. */
export const ActiveLight: Story = {
  args: { tasks: RUNNING },
  globals: { theme: 'light' },
  play: checkActive,
}

/** The same on the dark tokens. */
export const ActiveDark: Story = {
  args: { tasks: RUNNING },
  globals: { theme: 'dark' },
  play: checkActive,
}

/** A task failed: the chip has the alarm ground and keeps its text, on the light tokens. */
export const FailedLight: Story = {
  args: { tasks: FAILED },
  globals: { theme: 'light' },
  play: checkFailed,
}

/** The same on the dark tokens. */
export const FailedDark: Story = {
  args: { tasks: FAILED },
  globals: { theme: 'dark' },
  play: checkFailed,
}

/** The chip is 20px high whether it is idle, running or failed; only the width follows the text. */
export const KeepsItsSize: Story = {
  render: () => ({
    components: { TasksStatus },
    setup: () => ({ RUNNING, FAILED }),
    template: `
      <div class="numen" style="display:flex;align-items:center;gap:8px;block-size:2rem;background:var(--numen-titlebar);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <TasksStatus :tasks="[]" />
        <TasksStatus :tasks="RUNNING" />
        <TasksStatus :tasks="FAILED" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const boxes = within(canvasElement).getAllByRole('button').map(
      (one) => {
        const { width, height } = one.getBoundingClientRect()
        return { width, height }
      },
    )
    await expect(boxes).toHaveLength(3)
    for (const box of boxes) await expect(box.height).toBe(20)
  },
}

/** The scales, as [interface, text]: the designed size, each one larger, and both larger. */
const SCALES: readonly (readonly [string, string])[] = [
  ['1', '1'],
  ['2', '1'],
  ['1', '2'],
  ['2', '2'],
]

const measureAtEveryScale = <T>(measure: () => T): T[] => {
  const style = document.documentElement.style
  const found: T[] = []
  try {
    for (const [interfaceScale, textScale] of SCALES) {
      style.setProperty('--numen-interface-scale', interfaceScale)
      style.setProperty('--numen-text-scale', textScale)
      found.push(measure())
    }
  } finally {
    style.removeProperty('--numen-interface-scale')
    style.removeProperty('--numen-text-scale')
  }
  return found
}

/** The view as a bar draws it, with nothing around it, at the top or the bottom of the window. */
const inBar = (place: 'top' | 'bottom') => () => ({
  components: { TasksStatus },
  setup: () => ({ RUNNING, FAILED }),
  template: `
    <div class="numen" style="block-size: 100vh; background: var(--numen-surface); color: var(--numen-ink); font-family: var(--numen-font-sans)">
      <div data-view style="position: fixed; ${place}: 6px; right: 320px; display: inline-flex; align-items: center">
        <TasksStatus :tasks="[...RUNNING, ...FAILED]" />
      </div>
    </div>`,
})

const getViewChip = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement.querySelector<HTMLElement>('[data-view]')!).getByRole('button')

/** The chip is 20px high at every scale, and the view is exactly as wide and high as its chip. */
export const KeepsItsSizeAndAddsNoSpaceWhateverTheScale: Story = {
  render: inBar('top'),
  play: async ({ canvasElement }) => {
    const view = canvasElement.querySelector<HTMLElement>('[data-view]')!
    const found = measureAtEveryScale(() => {
      const chip = getViewChip(canvasElement).getBoundingClientRect()
      const box = view.getBoundingClientRect()
      return { chip: [chip.width, chip.height], view: [box.width, box.height] }
    })
    for (const one of found) {
      await expect(one.chip[1]).toBe(20)
      await expect(one.view).toStrictEqual(one.chip)
      await expect(one).toStrictEqual(found[0])
    }
  },
}

/** A press opens the list of tasks; its close button closes it. */
export const OpensTheTaskListAndClosesItFromItsButton: Story = {
  render: inBar('top'),
  play: async ({ canvasElement }) => {
    const chip = getViewChip(canvasElement)
    await userEvent.click(chip)
    const dialog = await within(document.body).findByRole('dialog', { name: 'Background tasks' })
    await expect(chip.getAttribute('aria-expanded')).toBe('true')
    await expect(within(dialog).getByText('The disk is full')).toBeVisible()
    await userEvent.click(within(dialog).getAllByRole('button', { name: 'Close' })[0]!)
    await waitFor(() => expect(within(document.body).queryByRole('dialog')).toBeNull())
    await expect(chip.getAttribute('aria-expanded')).toBe('false')
  },
}

/** In a bar at the bottom of the window the list opens above the chip, inside the window. */
export const OpensTheTaskListAboveAChipAtTheBottom: Story = {
  render: inBar('bottom'),
  play: async ({ canvasElement }) => {
    const chip = getViewChip(canvasElement)
    await userEvent.click(chip)
    const dialog = await within(document.body).findByRole('dialog', { name: 'Background tasks' })
    await waitFor(() =>
      expect(dialog.getBoundingClientRect().bottom).toBeLessThanOrEqual(
        chip.getBoundingClientRect().top,
      ),
    )
    await expect(dialog.getBoundingClientRect().top).toBeGreaterThanOrEqual(0)
  },
}
