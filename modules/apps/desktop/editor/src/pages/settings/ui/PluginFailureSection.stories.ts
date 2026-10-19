/**
 * A plugin the host suspends, drawn in a browser.
 *
 * What is asked here is what only a browser can answer: that the reason and the
 * two answers fit the page at the width of a narrow window, and that each
 * answer is announced with the plugin it is for.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import PluginFailureSection from './PluginFailureSection.vue'
import type { PluginInfo } from '@/entities/settings'
import { createInstallation, createPluginInfo } from '@/testing/installation'

const PLUGINS: readonly PluginInfo[] = [
  createPluginInfo('sample.stopped', 'Sample Stopped', {
    failure: {
      reason: 'The plugin could not be started: exec format error',
      failedAt: Date.UTC(2026, 9, 19, 12, 30),
    },
  }),
  createPluginInfo('sample.running', 'Sample Running', { isActive: true }),
]

const retryPlugin = fn()
const write = fn()

const installation = createInstallation({ write, retryPlugin, plugins: ref(PLUGINS) })

/** The section at the width a story gives. */
const section = (width: string) => () => ({
  components: { PluginFailureSection },
  setup: () => ({ state: { installation }, width }),
  template: `
    <div class="numen" :style="{ inlineSize: width, padding: '1rem', background: 'var(--numen-surface)' }">
      <PluginFailureSection :state="state" />
    </div>
  `,
})

const meta: Meta<typeof PluginFailureSection> = {
  title: 'Window/Plugin failures',
  component: PluginFailureSection,
  parameters: { layout: 'fullscreen' },
  render: section('48rem'),
}

export default meta
type Story = StoryObj<typeof meta>

/** A plugin suspended is drawn with its reason, and only that plugin. */
export const APluginHeldOff: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const group = canvas.getByRole('group', { name: 'Sample Stopped' })
    await expect(group).toHaveTextContent('The plugin could not be started: exec format error')
    await expect(canvas.queryByRole('group', { name: 'Sample Running' })).toBeNull()
  },
}

/** Each answer is announced with its plugin, and asks the window for it. */
export const TheTwoAnswers: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(canvas.getByRole('button', { name: 'Retry Sample Stopped' }))
    await expect(retryPlugin).toHaveBeenLastCalledWith('sample.stopped')

    await userEvent.click(canvas.getByRole('button', { name: 'Disable Sample Stopped' }))
    await expect(write).toHaveBeenLastCalledWith([
      { at: ['plugins', 'sample.stopped', 'enabled'], value: 'false' },
    ])
  },
}

/** The reason and both answers fit a narrow window. */
export const InANarrowWindow: Story = {
  render: section('20rem'),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const frame = canvasElement.querySelector('.numen')
    if (!frame) throw new Error('the page is not drawn')
    const page = frame.getBoundingClientRect()
    const retry = canvas.getByRole('button', { name: 'Retry Sample Stopped' })
    const disable = canvas.getByRole('button', { name: 'Disable Sample Stopped' })
    await expect(retry.getBoundingClientRect().right).toBeLessThanOrEqual(page.right)
    await expect(disable.getBoundingClientRect().right).toBeLessThanOrEqual(page.right)
  },
}
