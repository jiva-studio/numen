/**
 * A select with several choices in force. Also the test corpus: each story is run in a browser
 * by `@storybook/addon-vitest`.
 *
 * The choices are drawn at the end of the document, so they are read off the document and not
 * off the canvas.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import MultiSelect from './MultiSelect.vue'

interface Knobs {
  chosen: string
}

const meta: Meta<Knobs> = {
  title: 'Window/Multi select',
  component: MultiSelect,
  parameters: { layout: 'centered' },
  argTypes: { chosen: { control: 'text' } },
  args: { chosen: '' },
  render: (args) => ({
    components: { MultiSelect },
    setup: () => {
      const choices = [
        { id: 'sea', text: 'Sea', detail: 'Group one' },
        { id: 'river', text: 'River', detail: 'Group two' },
        { id: 'lake', text: 'Lake', detail: 'Group one' },
      ]
      const selected = ref<readonly string[]>(args.chosen.split(',').filter((one) => one !== ''))
      return { choices, selected }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: '14rem' }">
        <MultiSelect v-model:selected="selected" placeholder="Nothing chosen" aria-label="Water" :choices="choices" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const control = (canvas: HTMLElement): HTMLButtonElement =>
  canvas.querySelector<HTMLButtonElement>('[data-slot="select"]')!

const options = (): NodeListOf<HTMLElement> =>
  document.body.querySelectorAll<HTMLElement>('[role="option"]')

const marks = (): readonly Element[] =>
  Array.from(document.body.querySelectorAll('.searchable-list__option [aria-hidden="true"]'))

/** Nothing is chosen: the line carries the stand-in. */
export const NoneChosen: Story = {
  play: async ({ canvasElement }) => {
    await expect(control(canvasElement).textContent).toContain('Nothing chosen')
    await userEvent.click(control(canvasElement))
    await waitFor(() => expect(options()).toHaveLength(3))
    await expect(marks()).toHaveLength(0)
  },
}

/** One choice is made: the line carries its text and its row one mark. */
export const OneChosen: Story = {
  args: { chosen: 'river' },
  play: async ({ canvasElement }) => {
    await expect(control(canvasElement).textContent).toContain('River')
    await userEvent.click(control(canvasElement))
    await waitFor(() => expect(marks()).toHaveLength(1))
  },
}

/** Several are made: the line carries the first and the count of the rest, and the list stays open. */
export const SeveralChosen: Story = {
  args: { chosen: 'sea,lake' },
  play: async ({ canvasElement }) => {
    await expect(control(canvasElement).textContent).toContain('Sea +1')
    await userEvent.click(control(canvasElement))
    await waitFor(() => expect(marks()).toHaveLength(2))
    options()[1]!.click()
    await waitFor(() => expect(control(canvasElement).textContent).toContain('Sea +2'))
    await expect(options()).toHaveLength(3)
  },
}

/** Escape puts the list away and gives the keyboard back to the line. */
export const EscapePutsItAway: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(control(canvasElement))
    await waitFor(() => expect(options()).toHaveLength(3))
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(options()).toHaveLength(0))
    await expect(document.activeElement).toBe(control(canvasElement))
  },
}
