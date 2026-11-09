/**
 * The status bar with nothing placed in it, as the window drags a bar entry over it. The room an
 * empty slot keeps needs a browser to measure.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, waitFor } from 'storybook/test'
import StatusBar from './StatusBar.vue'

const meta: Meta<typeof StatusBar> = {
  title: 'Application/StatusBar',
  component: StatusBar,
  args: {
    startEntries: [],
    endEntries: [],
    listItems: () => Promise.resolve([]),
    onRun: fn(),
  },
  render: (args) => ({
    components: { StatusBar },
    setup: () => ({ args }),
    template: `<div style="inline-size: 40rem"><StatusBar v-bind="args" /></div>`,
  }),
}

export default meta
type Story = StoryObj<typeof meta>

/** While the window drags a bar entry, each empty slot keeps the room of one chip to take the drop. */
export const EmptySlotsWhileDragging: Story = {
  play: async ({ canvasElement }) => {
    const root = canvasElement.ownerDocument.documentElement
    root.setAttribute('data-numen-slot-dragging', '')
    try {
      for (const name of ['statusBar.left', 'statusBar.right']) {
        await waitFor(() => {
          const slot = canvasElement.querySelector(`[data-numen-slot="${name}"]`)
          if (!slot) throw new Error(`the slot ${name} is not drawn`)
          return expect(slot.getBoundingClientRect().width).toBeGreaterThanOrEqual(20)
        })
      }
    } finally {
      root.removeAttribute('data-numen-slot-dragging')
    }
  },
}
