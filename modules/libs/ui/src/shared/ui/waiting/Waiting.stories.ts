/**
 * What a pane says while what fills it is still coming.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import Waiting from './Waiting.vue'

const meta = {
  title: 'Waiting',
  component: Waiting,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A tab drawing nothing yet and a tab holding nothing look the ' +
          'same, and only this tells them apart. Every pane that waits says ' +
          'so the same way: one ring, one word, one size, in the middle of ' +
          'whatever room it was given.',
      },
    },
  },
} satisfies Meta<typeof Waiting>

export default meta
type Story = StoryObj<typeof meta>

/** In the room a pane gives it. */
export const Playground: Story = {
  render: () => ({
    components: { Waiting },
    template: `
      <div class="numen" style="block-size:320px;background:var(--numen-surface)">
        <Waiting />
      </div>
    `,
  }),
}

/**
 * Two panes of different heights, waiting side by side. The ring and the word
 * are the same size in both: neither takes its size from the room it stands in.
 */
export const OneSizeInEveryRoom: Story = {
  render: () => ({
    components: { Waiting },
    template: `
      <div class="numen" style="display:flex;gap:2px;background:var(--numen-surface)">
        <div style="flex:1;block-size:320px;font-size:2rem"><Waiting /></div>
        <div style="flex:1;block-size:120px;font-size:0.75rem"><Waiting /></div>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const [wide, narrow] = within(canvasElement).getAllByRole('status')

    const ringOf = (element: HTMLElement) =>
      element.querySelector<HTMLElement>('.spinner')!.getBoundingClientRect().height
    const wordOf = (element: HTMLElement) =>
      getComputedStyle(element.querySelector<HTMLElement>('span')!).fontSize

    expect(ringOf(wide!)).toBeCloseTo(ringOf(narrow!), 1)
    expect(wordOf(wide!)).toBe(wordOf(narrow!))
  },
}
