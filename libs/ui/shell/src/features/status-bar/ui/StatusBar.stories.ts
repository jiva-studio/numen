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

/** Many more tasks than fit in the popover. */
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
 * The trigger shows no system focus ring, whether pressed or reached by Tab:
 * the bar is outside the workspace, and its own hover background indicates focus.
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

/** The geometry and display of each part of a row. */
const readRowGeometry = (row: Element): string =>
  [row, ...row.querySelectorAll('.task-item__indicator, .task-item__action button')]
    .map((part) => {
      const box = part.getBoundingClientRect()
      const style = getComputedStyle(part)
      return `${box.x},${box.y},${box.width},${box.height},${style.display},${style.visibility}`
    })
    .join(';')

/** Properties that move a box, or take it out of the layout or the tab order. */
const LAYOUT_PROPERTIES = [
  'display',
  'visibility',
  'position',
  'inset',
  'width',
  'height',
  'margin',
  'padding',
]

/** Reports whether the selector, read without its `:hover`, matches one of the parts. */
function isMatchedWithoutHover(selector: string, parts: readonly Element[]): boolean {
  if (!selector.includes(':hover')) return false
  const selectorWithoutHover = selector.replaceAll(':hover', '')
  try {
    return parts.some((part) => part.matches(selectorWithoutHover))
  } catch {
    // A selector that is only `:hover` is invalid once stripped, and matches no part.
    return false
  }
}

/**
 * The `:hover` rules of the page that change the layout of a part of the row,
 * or the opacity of a part other than the action button.
 */
function findHoverLayoutRules(
  rules: CSSRuleList,
  parts: readonly Element[],
  fadedParts: readonly Element[],
): string[] {
  const found: string[] = []
  for (const rule of rules) {
    if (rule instanceof CSSStyleRule) {
      const names = Array.from(rule.style)
      const isLayout = names.some((name) => LAYOUT_PROPERTIES.some((one) => name.startsWith(one)))
      const isFade = names.includes('opacity')
      if (
        (isLayout && isMatchedWithoutHover(rule.selectorText, parts)) ||
        (isFade && isMatchedWithoutHover(rule.selectorText, fadedParts))
      ) {
        found.push(rule.cssText)
      }
    }
    if ('cssRules' in rule && rule.cssRules instanceof CSSRuleList) {
      found.push(...findHoverLayoutRules(rule.cssRules, parts, fadedParts))
    }
  }
  return found
}

/**
 * The cancel button keeps its slot whether or not it is shown: the pointer on
 * the row, or keyboard focus inside it, changes only the button's opacity, and
 * the progress stays drawn.
 */
export const HoverKeepsTheRow: Story = {
  args: { tasks: [INDEXING] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    const cancel = await canvas.findByRole('button', { name: 'Cancel task' })
    const row = cancel.closest('.task-item')
    const indicator = row?.querySelector('.task-item__indicator')
    if (!row || !indicator) throw new Error('the cancel button is outside a row with progress')
    const before = readRowGeometry(row)
    await expect(getComputedStyle(cancel).opacity).toBe('0')
    await expect(getComputedStyle(indicator).opacity).toBe('1')

    cancel.focus()
    await expect(cancel).toHaveFocus()
    await waitFor(() => expect(getComputedStyle(cancel).opacity).toBe('1'))
    await expect(readRowGeometry(row)).toBe(before)
    await expect(getComputedStyle(indicator).opacity).toBe('1')
    cancel.blur()

    const parts = [row, ...row.querySelectorAll('*')]
    const buttonParts = new Set<Element>([cancel, ...cancel.querySelectorAll('*')])
    const fadedParts = parts.filter((part) => !buttonParts.has(part))
    const hoverRules = Array.from(document.styleSheets).flatMap((sheet) =>
      findHoverLayoutRules(sheet.cssRules, parts, fadedParts),
    )
    await expect(hoverRules).toEqual([])

    const showRules = Array.from(document.styleSheets).flatMap((sheet) =>
      findHoverLayoutRules(sheet.cssRules, [], [cancel]),
    )
    await expect(showRules.length).toBeGreaterThan(0)
  },
}
