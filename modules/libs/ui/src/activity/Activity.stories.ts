/**
 * The line of work at the foot of the window.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Activity from './Activity.vue'

const meta = {
  title: 'Generic/Activity',
  component: Activity,
} satisfies Meta<typeof Activity>

export default meta
type Story = StoryObj<typeof meta>

/** The line itself, which is what every story reads. */
const lineIn = (canvas: HTMLElement): HTMLElement | null => canvas.querySelector('.activity')

/** Nothing to say draws nothing. Quiet is not the same as finished. */
export const Quiet: Story = {
  args: { says: '' },
  play: async ({ canvasElement }) => {
    await expect(lineIn(canvasElement)).toBeNull()
  },
}

/** Work with no count: dots, and no bar to fill. */
export const Working: Story = {
  args: { says: 'Reading', about: 'Sabhaparva.epub', working: true },
  play: async ({ canvasElement }) => {
    const line = lineIn(canvasElement)
    await expect(line).toHaveAttribute('data-state', 'working')
    await expect(line).toHaveTextContent('Sabhaparva.epub')
  },
}

/** A count that moves: how far, and how long is left. */
export const Counting: Story = {
  args: {
    says: 'Learning what it says',
    working: true,
    tally: { done: 1200, total: 36560 },
    left: 'about 2 hours left',
  },
  play: async ({ canvasElement }) => {
    const line = lineIn(canvasElement)
    await expect(line).toHaveTextContent('3%')
    await expect(line).toHaveTextContent('about 2 hours left')
    await expect(line).not.toHaveTextContent('1 200')
    await expect(line).not.toHaveTextContent('36 560')
  },
}

/**
 * Words and no work: something worth knowing that nothing is going to change by
 * itself. Drawn without the marks of progress.
 */
export const Resting: Story = {
  args: { says: 'Searching by words — no model to learn what it says' },
  play: async ({ canvasElement }) => {
    const line = lineIn(canvasElement)
    await expect(line).toHaveAttribute('data-state', 'resting')
  },
}

/** A caution is read at leisure, and is marked so it is read. */
export const Caution: Story = {
  args: { says: 'No tab of this window is over a note', tone: 'caution' },
  play: async ({ canvasElement }) => {
    const line = lineIn(canvasElement)
    await expect(line).toHaveAttribute('data-tone', 'caution')
    await expect(line).toHaveAttribute('data-state', 'resting')
  },
}

/** Alarm outranks a count: a line that is both failing and counting says it is failing. */
export const Trouble: Story = {
  args: {
    says: 'Reading',
    about: 'permission denied',
    tally: { done: 2, total: 8 },
    tone: 'alarm',
  },
  play: async ({ canvasElement }) => {
    const line = lineIn(canvasElement)
    await expect(line).toHaveAttribute('data-state', 'trouble')
    await expect(line).toHaveTextContent('permission denied')
    await expect(line).not.toHaveTextContent('2 of 8')
  },
}
