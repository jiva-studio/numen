/**
 * The tasks chip of the title bar in its states: ready, a task running, tasks failed. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`, which reads the size, the
 * pulse and the contrast of the chip as drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import { createTask } from '../testing/tasks'
import TasksStatus from './TasksStatus.vue'
import { expectText } from '../../../../libs/ui/core/.storybook/contrastBrowser'

const RUNNING = [createTask({ id: 'a', state: 'running', done: 3, total: 10 })]
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
  await expect(chip.textContent?.trim()).toBe('Ready')
  await expect(getComputedStyle(chip).animationName).toBe('none')
  await expectText(chip, 'tasks chip', 'ready')
}

const checkActive: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.textContent?.trim()).toBe('1 active')
  await expect(getComputedStyle(chip).animationName).not.toBe('none')
  await expect(chip.getAnimations().some((one) => one.playState === 'running')).toBe(true)
  await expect(canvasElement.querySelector('.circular-progress--indeterminate')).toBeNull()
  await expectText(chip, 'tasks chip', 'active')
}

const checkFailed: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.getAttribute('aria-label')).toBe('Ready, 1 failed')
  const badge = chip.querySelector('.tasks-status__badge') as HTMLElement
  await expect(badge.textContent?.trim()).toBe('1')
  await expectText(badge, 'tasks chip', 'failed badge')
}

/** No task runs: the chip says Ready, on the light tokens. */
export const IdleLight: Story = { globals: { theme: 'light' }, play: checkIdle }

/** The same on the dark tokens. */
export const IdleDark: Story = { globals: { theme: 'dark' }, play: checkIdle }

/** A task runs: the chip pulses, on the light tokens. */
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

/** A task failed: a red badge sits on the corner of the chip, on the light tokens. */
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

/** The chip keeps its height and its width whether it is ready, running or failed. */
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
    for (const box of boxes) await expect(box).toStrictEqual(boxes[0])
  },
}
