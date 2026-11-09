/**
 * The chip of the title bar and the status bar, with its popover. Also the test corpus: each story
 * is run in Chromium and WebKit, which place the popover against the chip. The stories check how
 * the popover opens and closes, not how the chip looks.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import BarChip from './BarChip.vue'
import type { BarChipTone } from '.'
import { FACE, LABEL, POPOVER_ACTION, POPOVER_LABEL, POPOVER_TEXT, TOOLTIP } from './fixtures/faces'
import { pressKey, pressOn } from './fixtures/press'

/** Where the chip is drawn in the viewport. */
type Place = 'top' | 'middle' | 'bottom'

interface Knobs {
  label: string
  tooltip: string
  tone: BarChipTone
  popoverLabel: string
  face: string
  place: Place
  hasPopover: boolean
}

const PLACES: Record<Place, string> = {
  top: 'position: fixed; top: 6px; right: 320px',
  middle: 'position: fixed; top: 300px; left: 500px',
  bottom: 'position: fixed; bottom: 0; right: 320px',
}

const meta: Meta<Knobs> = {
  title: 'Controls/Bar chip',
  component: BarChip,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    tone: { control: 'inline-radio', options: ['neutral', 'accent', 'alarm'] },
    place: { control: 'inline-radio', options: ['top', 'middle', 'bottom'] },
    hasPopover: { control: 'boolean' },
  },
  args: {
    label: LABEL,
    tooltip: TOOLTIP,
    tone: 'neutral',
    popoverLabel: POPOVER_LABEL,
    face: FACE,
    place: 'top',
    hasPopover: true,
  },
  render: (args) => ({
    components: { BarChip },
    setup: () => ({ args, isOpen: ref(false), style: PLACES[args.place] }),
    template: `
      <div class="numen" style="block-size: 100vh; background: var(--numen-surface); color: var(--numen-ink); font-family: var(--numen-font-sans)">
        <button type="button" data-outside style="position: fixed; top: 600px; left: 40px">Elsewhere</button>
        <div data-place :style="style">
          <BarChip v-model:open="isOpen" :label="args.label" :tooltip="args.tooltip" :tone="args.tone"
            :popover-label="args.popoverLabel">
            <span data-face>{{ args.face }}</span>
            <template v-if="args.hasPopover" #popover>
              <div data-content style="inline-size: 16rem; padding: 0.75rem">
                <p style="margin: 0 0 0.5rem">${POPOVER_TEXT}</p>
                <button type="button" data-action>${POPOVER_ACTION}</button>
              </div>
            </template>
          </BarChip>
        </div>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getChip = (canvas: HTMLElement): HTMLElement =>
  within(canvas).getByRole('button', { name: LABEL })

const findDialog = (): Promise<HTMLElement> =>
  within(document.body).findByRole('dialog', { name: POPOVER_LABEL })

const queryDialog = (): HTMLElement | null =>
  within(document.body).queryByRole('dialog', { name: POPOVER_LABEL })

const openPopover = async (canvas: HTMLElement): Promise<HTMLElement> => {
  await pressOn(getChip(canvas))
  return findDialog()
}

const expectClosed = async (canvas: HTMLElement): Promise<void> => {
  await waitFor(() => expect(queryDialog()).toBeNull())
  await expect(getChip(canvas).getAttribute('aria-expanded')).toBe('false')
}

const settle = (): Promise<void> =>
  new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))

/** A chip with a face and a popover, at rest. */
export const AChip: Story = {}

/** A press on the chip opens the popover below it and marks the chip expanded. */
export const OpensThePopoverBelowTheChip: Story = {
  play: async ({ canvasElement }) => {
    const chip = getChip(canvasElement)
    await expect(chip.getAttribute('aria-expanded')).toBe('false')
    const dialog = await openPopover(canvasElement)
    await expect(chip.getAttribute('aria-expanded')).toBe('true')
    await expect(within(dialog).getByText(POPOVER_TEXT)).toBeVisible()
    await waitFor(() =>
      expect(dialog.getBoundingClientRect().top).toBeGreaterThanOrEqual(
        chip.getBoundingClientRect().bottom,
      ),
    )
  },
}

/** At the bottom edge of the window the popover opens above the chip and stays inside the window. */
export const OpensThePopoverAboveTheChipAtTheBottomEdge: Story = {
  args: { place: 'bottom' },
  play: async ({ canvasElement }) => {
    const chip = getChip(canvasElement)
    const dialog = await openPopover(canvasElement)
    await waitFor(() =>
      expect(dialog.getBoundingClientRect().bottom).toBeLessThanOrEqual(
        chip.getBoundingClientRect().top,
      ),
    )
    const box = dialog.getBoundingClientRect()
    await expect(box.top).toBeGreaterThanOrEqual(0)
    await expect(box.left).toBeGreaterThanOrEqual(0)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
  },
}

/** Escape closes the popover. */
export const ClosesThePopoverOnEscape: Story = {
  play: async ({ canvasElement }) => {
    await openPopover(canvasElement)
    await pressKey('{Escape}')
    await expectClosed(canvasElement)
  },
}

/** A press outside the chip and the popover closes the popover. */
export const ClosesThePopoverOnAPressOutside: Story = {
  play: async ({ canvasElement }) => {
    await openPopover(canvasElement)
    await settle()
    await pressOn(canvasElement.querySelector('[data-outside]')!)
    await expectClosed(canvasElement)
  },
}

/** A press on a button inside the popover keeps it open. */
export const KeepsThePopoverOpenOnAPressInside: Story = {
  play: async ({ canvasElement }) => {
    const dialog = await openPopover(canvasElement)
    await settle()
    await pressOn(within(dialog).getByRole('button', { name: POPOVER_ACTION }))
    await settle()
    await expect(queryDialog()).not.toBeNull()
    await expect(getChip(canvasElement).getAttribute('aria-expanded')).toBe('true')
  },
}

/** A second press on the chip closes the popover. */
export const ClosesThePopoverOnASecondPressOfTheChip: Story = {
  play: async ({ canvasElement }) => {
    await openPopover(canvasElement)
    await settle()
    await pressOn(getChip(canvasElement))
    await expectClosed(canvasElement)
  },
}

/** A resize of the window keeps the popover open next to its chip. */
export const KeepsThePopoverOpenOnAResize: Story = {
  play: async ({ canvasElement }) => {
    const chip = getChip(canvasElement)
    const dialog = await openPopover(canvasElement)
    await settle()
    window.dispatchEvent(new Event('resize'))
    await settle()
    await expect(queryDialog()).not.toBeNull()
    await expect(dialog.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      chip.getBoundingClientRect().bottom,
    )
  },
}

/** The popover opens even when the face under the pointer is replaced between press and release. */
export const OpensThePopoverWhenTheFaceIsReplacedDuringThePress: Story = {
  render: () => ({
    components: { BarChip },
    setup: () => ({ LABEL, POPOVER_LABEL, POPOVER_TEXT, presses: ref(0) }),
    template: `
      <div class="numen" style="padding: 8px; background: var(--numen-surface); font-family: var(--numen-font-sans)"
        @pointerdown.capture="presses++">
        <BarChip :label="LABEL" :popover-label="POPOVER_LABEL">
          <span :key="presses" data-face>{{ presses }} of 9 done</span>
          <template #popover><p style="inline-size: 12rem">{{ POPOVER_TEXT }}</p></template>
        </BarChip>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const chip = getChip(canvasElement)
    await expect(chip.textContent?.trim()).toBe('0 of 9 done')
    await pressOn(chip)
    await expect(chip.textContent?.trim()).toBe('1 of 9 done')
    await findDialog()
    await expect(chip.getAttribute('aria-expanded')).toBe('true')
  },
}
