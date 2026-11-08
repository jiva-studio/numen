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
  progress: 70,
  etaSeconds: 90,
  state: 'running',
  canCancel: true,
}

const FAILED: StatusBarTask = {
  id: 'failed',
  label: 'Reading file',
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

const readRowGeometry = (row: Element): string =>
  [row, ...row.querySelectorAll('.task-item__slot, .task-item__indicator, .task-item__btn')]
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

function findHoverLayoutRules(rules: CSSRuleList, parts: readonly Element[]): string[] {
  const found: string[] = []
  for (const rule of rules) {
    if (rule instanceof CSSStyleRule) {
      const isLayout = Array.from(rule.style).some((name) =>
        LAYOUT_PROPERTIES.some((one) => name.startsWith(one)),
      )
      if (isLayout && isMatchedWithoutHover(rule.selectorText, parts)) found.push(rule.cssText)
    }
    if ('cssRules' in rule && rule.cssRules instanceof CSSRuleList) {
      found.push(...findHoverLayoutRules(rule.cssRules, parts))
    }
  }
  return found
}

const getBox = (element: Element) => {
  const box = element.getBoundingClientRect()
  return { x: box.x, y: box.y, width: box.width, height: box.height }
}

/**
 * The progress ring and the cancel button are stacked in one slot, in the same
 * box. The row decides which one shows, and only their opacity changes: at rest
 * the ring shows, and with focus inside the row (the synthetic hover does not
 * start CSS `:hover`) the button shows in its place. No `:hover` rule changes
 * the layout of the row.
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

    await expect(getBox(cancel)).toEqual(getBox(indicator))
    const before = readRowGeometry(row)
    await expect(getComputedStyle(indicator).opacity).toBe('1')
    await expect(getComputedStyle(cancel).opacity).toBe('0')

    cancel.focus()
    await expect(cancel).toHaveFocus()
    await waitFor(() => expect(getComputedStyle(cancel).opacity).toBe('1'))
    await waitFor(() => expect(getComputedStyle(indicator).opacity).toBe('0'))
    await expect(readRowGeometry(row)).toBe(before)
    await expect(getBox(cancel)).toEqual(getBox(indicator))
    cancel.blur()

    const parts = [row, ...row.querySelectorAll('*')]
    const hoverRules = Array.from(document.styleSheets).flatMap((sheet) =>
      findHoverLayoutRules(sheet.cssRules, parts),
    )
    await expect(hoverRules).toEqual([])
  },
}

/** A task that cannot be cancelled has only the ring in its slot. */
export const NotCancellableHasOnlyTheRing: Story = {
  args: { tasks: [{ ...INDEXING, canCancel: false }] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    await waitFor(() => expect(canvasElement.querySelector('.task-item__slot')).not.toBeNull())
    const slot = canvasElement.querySelector('.task-item__slot')
    await expect(slot?.querySelectorAll('button')).toHaveLength(0)
    await expect(slot?.querySelectorAll('.task-item__indicator')).toHaveLength(1)
  },
}

/**
 * Tasks are listed in one flat list, newest first, without type headings.
 */
export const FlatNewestFirst: Story = {
  args: {
    tasks: [
      { ...INDEXING, id: 'first', label: 'Indexing book' },
      { ...TRANSCRIBING, id: 'second', label: 'Transcribing audio' },
      { id: 'third', label: 'Walking the notes', state: 'running' },
    ],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    await waitFor(() => expect(canvasElement.querySelectorAll('.task-item')).toHaveLength(3))
    const rows = Array.from(canvasElement.querySelectorAll('.task-item')).map((row) =>
      row.getAttribute('data-task-id'),
    )
    await expect(rows).toEqual(['third', 'second', 'first'])
    await expect(canvasElement.querySelectorAll('.task-group, .task-group__header')).toHaveLength(0)
  },
}

const SETTINGS_NOTICE: StatusBarTask = {
  id: 'settings-notice',
  label: 'Read settings',
  about:
    'numen.json: "app.appearance.theme": "plugin:md.numen.themes.solarized/dark" is not one of the options listed in themes; the default preset:numen applies',
  state: 'notice',
}

/** A notice is counted in neither number of the bar, and its row shows all its text and a cross. */
export const NoticeRow: Story = {
  args: { tasks: [INDEXING, SETTINGS_NOTICE] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /active/ }))
    await waitFor(() =>
      expect(canvasElement.querySelector('[data-task-id="settings-notice"]')).not.toBeNull(),
    )
    await expect(canvasElement.textContent).toContain('1 active')
    await expect(canvasElement.textContent).not.toContain('failed')
    const row = canvasElement.querySelector('[data-task-id="settings-notice"]') as HTMLElement
    await expect(row.querySelector('.task-item__notice-icon')).not.toBeNull()
    await expect(row.querySelector('.circular-progress')).toBeNull()
    const detail = row.querySelector('.task-item__detail') as HTMLElement
    await expect(detail.textContent).toBe(SETTINGS_NOTICE.about)
    await expect(detail.scrollWidth).toBeLessThanOrEqual(detail.clientWidth)
    await expect(getComputedStyle(detail).whiteSpace).toBe('normal')
    const dismiss = row.querySelector('button[aria-label="Dismiss"]') as HTMLElement
    await expect(getComputedStyle(dismiss).opacity).toBe('1')
    await expect(dismiss.querySelector('svg')).not.toBeNull()
  },
}
