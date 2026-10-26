/**
 * Every state the status bar has to survive. Also the test corpus: each story
 * its row, or a popover that stays open after a press elsewhere, can fail.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import StatusBar from './StatusBar.vue'
import type { StatusBarTask } from '../model/types'
import { LONG, RUSSIAN, UNBREAKABLE } from '@/shared/fixtures/prose'

interface Knobs {
  tasks: readonly StatusBarTask[]
  idleLabel: string
  workingLabel: string
  activeLabel: string
  failedLabel: string
  regionLabel: string
}

const INDEXING: StatusBarTask = {
  id: 'indexing',
  label: 'Indexing book',
  group: 'Indexing',
  about: 'library/mahabharata.epub',
  progress: 50,
  done: 2,
  total: 4,
  state: 'running',
  canCancel: true,
}

const TRANSCRIBING: StatusBarTask = {
  id: 'transcribing',
  label: 'Transcribing audio',
  group: 'Audio & Transcription',
  progress: 70,
  etaSeconds: 90,
  state: 'running',
  canCancel: true,
}

const FAILED: StatusBarTask = {
  id: 'failed',
  label: 'Reading file',
  group: 'Text Recognition',
  about: 'scans/page-4.pdf',
  state: 'failed',
  error: 'The file is corrupted',
}

const meta: Meta<Knobs> = {
  title: 'Application/StatusBar',
  component: StatusBar,
  parameters: { layout: 'padded' },
  argTypes: {
    tasks: { control: 'object' },
    idleLabel: { control: 'text' },
    workingLabel: { control: 'text' },
    activeLabel: { control: 'text' },
    failedLabel: { control: 'text' },
    regionLabel: { control: 'text' },
  },
  args: {
    tasks: [INDEXING, TRANSCRIBING],
    idleLabel: 'Ready',
    workingLabel: 'Background tasks',
    activeLabel: 'active',
    failedLabel: 'failed',
    regionLabel: 'Status Bar',
  },
  render: (args) => ({
    components: { StatusBar },
    setup: () => ({ args }),
    template: `
      <div style="display: flex; align-items: flex-end; block-size: 24rem">
        <StatusBar v-bind="args" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const isPopoverOpen = (canvas: HTMLElement): boolean =>
  canvas.querySelector('.status-bar__popover-container') !== null

/** Two tasks under way, summed in the bar. */
export const Working: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.textContent).toContain('2 active'))
  },
}

/** Nothing running, nothing failed. */
export const Idle: Story = {
  args: { tasks: [] },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.textContent).toContain('Ready'))
  },
}

/** One task, and one that stopped. */
export const WithAFailure: Story = {
  args: { tasks: [INDEXING, FAILED] },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.textContent).toContain('1 failed'))
  },
}

/** Far more tasks than the popover has room for. */
export const FarTooMany: Story = {
  args: {
    tasks: Array.from({ length: 40 }, (_, index) => ({
      ...INDEXING,
      id: `task-${index}`,
      label: `Indexing book ${index}`,
    })),
  },
}

/** Words that are not Latin, in every label the bar owns. */
export const NotLatin: Story = {
  args: {
    tasks: [{ ...INDEXING, label: RUSSIAN, about: RUSSIAN }],
    idleLabel: RUSSIAN,
    activeLabel: 'в работе',
    failedLabel: 'с ошибкой',
    workingLabel: RUSSIAN,
  },
}

/** Text far too long, and a word with nothing to break at. */
export const TextTooLong: Story = {
  args: {
    tasks: [
      { ...INDEXING, label: LONG, about: LONG },
      { ...TRANSCRIBING, label: UNBREAKABLE, about: UNBREAKABLE },
      { ...FAILED, error: LONG },
    ],
  },
}

/** No text at all where there is text to draw. */
export const NoText: Story = {
  args: {
    tasks: [{ ...INDEXING, label: '', about: '' }],
    idleLabel: '',
    activeLabel: '',
    workingLabel: '',
  },
}

/** A press outside the bar puts the popover away. */
export const DismissedByAPressOutside: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    await waitFor(() => expect(isPopoverOpen(canvasElement)).toBe(true))
    await userEvent.click(document.body)
    await waitFor(() => expect(isPopoverOpen(canvasElement)).toBe(false))
  },
}

/** Escape puts the popover away. */
export const DismissedByEscape: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    await waitFor(() => expect(isPopoverOpen(canvasElement)).toBe(true))
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(isPopoverOpen(canvasElement)).toBe(false))
  },
}

/**
 * The trigger rings in no colour of the machine's, pressed or reached by Tab: the bar stands
 * outside the workspace, and what says where focus stands is its own hover ground.
 */
export const NoRingOnFocus: Story = {
  play: async ({ canvasElement }) => {
    const trigger = await within(canvasElement).findByRole('button', { name: /active/ })
    await userEvent.click(trigger)
    await expect(trigger).toHaveFocus()
    await expect(getComputedStyle(trigger).outlineStyle).toBe('none')
    await userEvent.keyboard('{Escape}')

    trigger.blur()
    await userEvent.tab()
    await expect(trigger).toHaveFocus()
    await expect(getComputedStyle(trigger).outlineStyle).toBe('none')
  },
}
