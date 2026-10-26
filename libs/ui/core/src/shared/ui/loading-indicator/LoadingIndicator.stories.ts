/**
 * What a pane says while what fills it is still coming.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import LoadingIndicator from './LoadingIndicator.vue'

const meta = {
  title: 'LoadingIndicator',
  component: LoadingIndicator,
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
} satisfies Meta<typeof LoadingIndicator>

export default meta
type Story = StoryObj<typeof meta>

/** In the room a pane gives it. */
export const Playground: Story = {
  render: () => ({
    components: { LoadingIndicator },
    template: `
      <div class="numen" style="block-size:320px;background:var(--numen-surface)">
        <LoadingIndicator :delay="0" />
      </div>
    `,
  }),
}

/**
 * Two panes of different heights, waiting side by side. The ring and the word
 */
export const OneSizeInEveryRoom: Story = {
  render: () => ({
    components: { LoadingIndicator },
    template: `
      <div class="numen" style="display:flex;gap:2px;background:var(--numen-surface)">
        <div style="flex:1;block-size:320px;font-size:2rem"><LoadingIndicator :delay="0" /></div>
        <div style="flex:1;block-size:120px;font-size:0.75rem"><LoadingIndicator :delay="0" /></div>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const [wide, narrow] = within(canvasElement).getAllByRole('status')

    const getRing = (element: HTMLElement) =>
      element.querySelector<HTMLElement>('.spinner')!.getBoundingClientRect().height
    const getWord = (element: HTMLElement) =>
      getComputedStyle(element.querySelector<HTMLElement>('span')!).fontSize

    expect(getRing(wide!)).toBeCloseTo(getRing(narrow!), 1)
    expect(getWord(wide!)).toBe(getWord(narrow!))
  },
}

/**
 * A pane whose content comes within the delay never shows the ring: the room stays empty until the
 * delay has passed.
 */
export const EmptyWithinItsDelay: Story = {
  render: () => ({
    components: { LoadingIndicator },
    template: `
      <div class="numen" style="block-size:120px;background:var(--numen-surface)">
        <LoadingIndicator :delay="300" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const pane = within(canvasElement).getByRole('status')

    expect(pane.querySelector('.spinner')).toBeNull()
    expect(pane).toHaveTextContent('')
    await new Promise((resolve) => setTimeout(resolve, 400))
    expect(pane.querySelector('.spinner')).not.toBeNull()
    expect(pane).toHaveTextContent('Loading…')
  },
}
