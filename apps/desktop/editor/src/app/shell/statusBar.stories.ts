/**
 * The window's status bar over the task list the core sends: tasks grouped by the plugin that
 * reported them, with labels that name no domain.
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
})

/** Two tasks in each group, since the popover draws a group heading only over more than one. */
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

/** Each plugin's tasks under the plugin's name, and the core's work under its own group. */
export const GroupedByPlugin: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(await within(canvasElement).findByRole('button', { name: /active/ }))
    await waitFor(() => {
      const titles = [...canvasElement.querySelectorAll('.task-group__title')].map(
        (one) => one.textContent,
      )
      return expect(titles).toEqual(['Speech to text', 'Scan reader', 'Numen'])
    })
  },
}
