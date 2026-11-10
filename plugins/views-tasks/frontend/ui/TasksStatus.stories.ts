/**
 * The tasks chip of the title bar in its states: ready, a task running, tasks failed. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`, which reads the size and
 * the contrast of the chip as drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { createTask } from '../testing/tasks'
import TasksStatus from './TasksStatus.vue'

const RUNNING = [
  createTask({ id: 'a', state: 'running', progress: { done: 3, total: 10 } }),
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
        <TasksStatus v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof TasksStatus>

export default meta
type Story = StoryObj<typeof meta>

const getChip = (canvasElement: HTMLElement): HTMLElement =>
  canvasElement.querySelector('.tasks-status__trigger') as HTMLElement

const checkIdle: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.textContent?.trim()).toBe('Idle')
  await expect(chip.getAttribute('aria-label')).toBe('No tasks running')
  await expect(chip.getAnimations()).toHaveLength(0)
  await expect(chip.querySelector('.tasks-status__ring-fill')).toBeNull()
  await expect(getComputedStyle(chip).borderTopWidth).toBe('0px')
  await expect(chip.getBoundingClientRect().height).toBe(20)
}

const checkActive: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await waitFor(() => expect(chip.textContent?.trim()).toBe('0 of 2'), { timeout: 2000 })
  await expect(chip.getAttribute('aria-label')).toBe('0 of 2 tasks done')
  await expect(chip.getAnimations()).toHaveLength(0)
  await expect(chip.querySelector('.tasks-status__ring-fill')).not.toBeNull()
}

const checkFailed: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.getAttribute('aria-label')).toBe('1 failed')
  await expect(chip.textContent?.trim()).toBe('Idle')
  await expect(chip.querySelector('.tasks-status__badge')).toBeNull()
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
    const boxes = [...canvasElement.querySelectorAll<HTMLElement>('.tasks-status__trigger')].map(
      (one) => {
        const { width, height } = one.getBoundingClientRect()
        return { width, height }
      },
    )
    await expect(boxes).toHaveLength(3)
    for (const box of boxes) await expect(box.height).toBe(20)
  },
}
