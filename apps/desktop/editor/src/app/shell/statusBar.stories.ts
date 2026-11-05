/**
 * The window's status bar over the task list the core sends: tasks listed newest first without
 * group headings, with labels that name no domain.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { StatusBar } from '@numen/shell'
import { shallowRef } from 'vue'
import type { Task } from '@/shared/notices/task'
import { useWindowStatusBar } from './useWindowStatusBar'

const task = (id: string, pluginId: string, pluginName: string): Task => ({
  id,
  label: `Step ${id}`,
  about: `library/${id}.dat`,
  completedCount: 1,
  total: 4,
  counting: 'items',
  error: '',
  isUserStarted: false,
  pluginId,
  pluginName,
  isCancellable: true,
  isNotice: false,
})

const TASKS: readonly Task[] = [
  task('one', 'md.example.speech', 'Speech to text'),
  task('two', 'md.example.scans', 'Scan reader'),
  task('three', 'md.example.speech', 'Speech to text'),
  task('four', '', ''),
  task('five', 'md.example.scans', 'Scan reader'),
  task('six', '', ''),
]

const meta: Meta = {
  title: 'Desktop/StatusBar',
  parameters: { layout: 'padded' },
  render: () => ({
    components: { StatusBar },
    setup: () => ({ statusBar: useWindowStatusBar({ tasks: shallowRef(TASKS) }) }),
    template: `
      <div style="display: flex; align-items: flex-end; block-size: 24rem">
        <StatusBar :tasks="statusBar.tasks.value" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj

/** All tasks in one flat list, newest first, without plugin or type headings. */
export const FlatNewestFirst: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(await within(canvasElement).findByRole('button', { name: /active/ }))
    await waitFor(() => {
      const ids = [...canvasElement.querySelectorAll('.task-item')].map((one) =>
        one.getAttribute('data-task-id'),
      )
      return expect(ids).toEqual(['six', 'five', 'four', 'three', 'two', 'one'])
    })
    await expect(canvasElement.querySelectorAll('.task-group, .task-group__title')).toHaveLength(0)
  },
}
