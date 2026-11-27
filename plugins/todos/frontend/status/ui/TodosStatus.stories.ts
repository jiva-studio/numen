/**
 * The tasks chip of the title bar and the lists it opens. Also the test corpus: each story is run
 * in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { createFakeApi, createTask, SUMMARY } from '../../testing/fakeApi'
import { taskDoneLabel } from '../../words'
import TodosStatus from './TodosStatus.vue'

const TODAY = '2026-04-14'

const TASKS = {
  overdue: {
    tasks: [
      createTask({ path: 'Garden/Beds.md', line: 7, text: 'Turn the compost', due: '2026-04-10' }),
      createTask({
        path: 'Daily/2026-04-12.md',
        line: 3,
        text: 'Order seed potatoes',
        due: '2026-04-12',
      }),
    ],
    total: 2,
    revision: 1,
  },
  today: {
    tasks: [
      createTask({
        path: 'Garden/Greenhouse.md',
        line: 2,
        text: 'Pot on the tomatoes',
        due: TODAY,
      }),
      createTask({ path: 'Garden/Beds.md', line: 11, text: 'Net the brassicas', due: TODAY }),
      createTask({ path: 'Daily/2026-04-14.md', line: 4, text: 'Water the seedlings', due: TODAY }),
    ],
    total: 3,
    revision: 1,
  },
  nextSevenDays: {
    tasks: [
      createTask({ path: 'Garden/Beds.md', line: 15, text: 'Sow the carrots', due: '2026-04-16' }),
      createTask({ path: 'Plans/Shed.md', line: 5, text: 'Fix the shed door', due: '2026-04-19' }),
    ],
    total: 2,
    revision: 1,
  },
  none: {
    tasks: [createTask({ path: 'Ideas.md', line: 12, text: 'Build a cold frame' })],
    total: 1,
    revision: 1,
  },
}

const meta = {
  title: 'Todos/Status chip',
  component: TodosStatus,
  parameters: { layout: 'fullscreen' },
  args: {
    summary: SUMMARY,
    api: createFakeApi({ tasks: TASKS }).api,
    getToday: () => TODAY,
    shown: { overdue: true, today: true, nextSevenDays: true, noDate: true },
    onOpen: fn(),
    onShowError: fn(),
  },
  render: (args) => ({
    components: { TodosStatus },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:center;justify-content:flex-end;block-size:2rem;padding-inline-end:12rem;background:var(--numen-titlebar);border-bottom:1px solid var(--numen-rule);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <TodosStatus v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof TodosStatus>

export default meta
type Story = StoryObj<typeof meta>

/** The chip opened: the tasks due under Overdue, Today, Next 7 days and No date. */
export const Opened: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Tasks due: 8' }))
    const lists = await within(document.body).findByRole('dialog', { name: 'Tasks due' })
    await waitFor(() =>
      expect(
        within(lists).getByRole('checkbox', { name: taskDoneLabel('Pot on the tomatoes') }),
      ).toBeVisible(),
    )
    await expect(
      within(lists)
        .getAllByRole('heading')
        .map((one) => one.textContent?.replace(/\s+/g, ' ').trim()),
    ).toEqual(['Overdue 2', 'Today 3', 'Next 7 days 2', 'No date 1'])
  },
}

const task = (path: string, line: number, text: string, due = '') =>
  createTask({ path, line, text, ...(due ? { due } : {}) })

/** The tasks a vault of physics notes has due. */
const PHYSICS = {
  overdue: {
    tasks: [
      task('Physics/Entropy.md', 31, 'Check the units in the Boltzmann formula', '2026-04-10'),
      task('Reading list.md', 4, 'Return Shannon 1948 to the library', '2026-04-12'),
    ],
    total: 2,
    revision: 1,
  },
  today: {
    tasks: [
      task(
        "Computation/Landauer's principle.md",
        12,
        'Work out kT ln 2 at room temperature',
        TODAY,
      ),
      task('Lectures/Lecture 4.md', 3, 'Correct the transcript of Lecture 4', TODAY),
      task("Physics/Maxwell's demon.md", 18, 'Add the Szilard engine', TODAY),
    ],
    total: 3,
    revision: 1,
  },
  nextSevenDays: {
    tasks: [
      task('Physics/Free energy.md', 7, 'Derive the Helmholtz form', '2026-04-16'),
      task('Reading list.md', 9, 'Read Bennett 1982', '2026-04-19'),
    ],
    total: 2,
    revision: 1,
  },
  none: {
    tasks: [task('Inbox.md', 2, 'Ask about the Gibbs paradox')],
    total: 1,
    revision: 1,
  },
}

/** The chip in the title bar of the window, opened on the tasks due. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'title-status' },
    screenshot: [{ name: 'tasks-due' }],
  },
  args: { api: createFakeApi({ tasks: PHYSICS }).api },
  render: (args) => ({
    components: { TodosStatus },
    setup: () => ({ args }),
    template: '<TodosStatus v-bind="args" />',
  }),
  play: async ({ canvasElement }) => {
    const chip = within(canvasElement).getByRole('button', { name: 'Tasks due: 8' })
    await userEvent.click(chip)
    await userEvent.unhover(chip)
    const lists = await within(document.body).findByRole('dialog', { name: 'Tasks due' })
    await waitFor(() =>
      expect(
        within(lists).getByRole('checkbox', { name: taskDoneLabel('Add the Szilard engine') }),
      ).toBeVisible(),
    )
  },
}
