/**
 * The tasks chip of the title bar in its states: ready, a task running, tasks failed. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { createTask } from '../testing/tasks'
import TasksStatus from './TasksStatus.vue'

const RUNNING = [
  createTask({ id: 'a', state: 'running', progress: { completedCount: 3, total: 10 } }),
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
  await expect(chip.querySelector('.tasks-status__ring-fill')).toBeNull()
}

const checkActive: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await waitFor(() => expect(chip.textContent?.trim()).toBe('0 of 2'), { timeout: 2000 })
  await expect(chip.getAttribute('aria-label')).toBe('0 of 2 tasks done')
  await expect(chip.querySelector('.tasks-status__ring-fill')).not.toBeNull()
}

const checkFailed: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.getAttribute('aria-label')).toBe('1 failed')
  await expect(chip.textContent?.trim()).toBe('Idle')
  await expect(chip.querySelector('.tasks-status__badge')).toBeNull()
}

/** No task runs: an empty ring and Idle. */
export const Idle: Story = { play: checkIdle }

/** Tasks run: the ring is filled by the progress and the chip counts the tasks done. */
export const Active: Story = { args: { tasks: RUNNING }, play: checkActive }

/** A task failed: the chip keeps its text and says how many failed. */
export const Failed: Story = { args: { tasks: FAILED }, play: checkFailed }
