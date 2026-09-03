/**
 * The panel, with something on it and with nothing on it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Panel from './Panel.vue'
import { framed } from '@/fixtures/frame'
import { LONG } from '@/fixtures/prose'

const meta = {
  title: 'Generic/Panel',
  component: Panel,
  decorators: [framed],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A surface to put things on: rounded on every side, with what it ' +
          'covers showing through it. It stacks what it is given in a column ' +
          'and keeps them clear of its edges. Anything that scrolls scrolls ' +
          'itself.',
      },
    },
  },
} satisfies Meta<typeof Panel>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {
  render: () => ({
    components: { Panel },
    setup: () => ({ LONG }),
    template: `<Panel class="max-h-full"><p class="min-h-0 overflow-y-auto">{{ LONG }}</p></Panel>`,
  }),
  play: async ({ canvasElement }) => {
    // The theme the page chose reaches the components on it.
    const doc = canvasElement.ownerDocument
    const was = doc.documentElement.style.colorScheme
    for (const scheme of ['dark', 'light'] as const) {
      doc.documentElement.style.colorScheme = scheme
      const panel = canvasElement.querySelector('.numen')!
      await expect(getComputedStyle(panel).colorScheme).toBe(scheme)
    }
    doc.documentElement.style.colorScheme = was
  },
}

/**
 * More on it than it has room for. The panel keeps the things on it in a
 * column and clips what is past its edge; nothing here scrolls itself.
 */
export const Crowded: Story = {
  render: () => ({
    components: { Panel },
    setup: () => ({ LONG }),
    template: `
      <Panel data-panel class="h-40">
        <p>{{ LONG }}</p>
        <p>{{ LONG }}</p>
      </Panel>
    `,
  }),
  play: async ({ canvasElement }) => {
    const panel = canvasElement.querySelector<HTMLElement>('[data-panel]')!
    const [first, second] = [...panel.children] as HTMLElement[]

    // One under the other, clear of each other and of the panel's own edge.
    const above = first!.getBoundingClientRect()
    const below = second!.getBoundingClientRect()
    await expect(below.top).toBeGreaterThan(above.bottom)
    await expect(above.left).toBe(below.left)
    await expect(above.top).toBeGreaterThan(panel.getBoundingClientRect().top)

    // Past the edge, and no bar offered to reach it.
    await expect(panel.scrollHeight).toBeGreaterThan(panel.clientHeight)
    await expect(getComputedStyle(panel).overflowY).toBe('hidden')
  },
}

/** Nothing on it. */
export const Empty: Story = {
  render: () => ({
    components: { Panel },
    template: `<Panel class="h-40" />`,
  }),
}
