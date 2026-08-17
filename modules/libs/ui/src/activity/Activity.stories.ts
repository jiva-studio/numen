/**
 * The line of work at the foot of the window.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import Activity from './Activity.vue'

const meta = {
  title: 'Generic/Activity',
  component: Activity,
} satisfies Meta<typeof Activity>

export default meta
type Story = StoryObj<typeof meta>

/** Nothing to say draws nothing. Quiet is not the same as finished. */
export const Quiet: Story = {
  args: { says: '' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).queryByRole('status')).toBeNull()
  },
}

/** Work with no count: dots, and no bar to fill. */
export const Working: Story = {
  args: { says: 'Reading', about: 'Sabhaparva.epub', working: true },
  play: async ({ canvasElement }) => {
    const line = within(canvasElement).getByRole('status')
    await expect(line).toHaveAttribute('data-state', 'working')
    await expect(line).toHaveTextContent('Sabhaparva.epub')
  },
}

/** A count that moves, drawn as a bar and read as a number. */
export const Counting: Story = {
  args: {
    says: 'Learning what it says',
    working: true,
    tally: { done: 1200, total: 36560 },
    left: 'about 2 hours left',
  },
  play: async ({ canvasElement }) => {
    const line = within(canvasElement).getByRole('status')
    await expect(line).toHaveTextContent('1 200 of 36 560')
    await expect(line).toHaveTextContent('3%')
    await expect(line).toHaveTextContent('about 2 hours left')
  },
}

/**
 * Words and no work: something worth knowing that nothing is going to change by
 * itself. Drawn without the marks of progress.
 */
export const Resting: Story = {
  args: { says: 'Searching by words — no model to learn what it says' },
  play: async ({ canvasElement }) => {
    const line = within(canvasElement).getByRole('status')
    await expect(line).toHaveAttribute('data-state', 'resting')
  },
}

/** Trouble outranks a count: a line that is both failing and counting says it is failing. */
export const Trouble: Story = {
  args: { says: 'Reading', tally: { done: 2, total: 8 }, trouble: 'permission denied' },
  play: async ({ canvasElement }) => {
    const line = within(canvasElement).getByRole('status')
    await expect(line).toHaveAttribute('data-state', 'trouble')
    await expect(line).toHaveTextContent('permission denied')
    await expect(line).not.toHaveTextContent('2 of 8')
  },
}
