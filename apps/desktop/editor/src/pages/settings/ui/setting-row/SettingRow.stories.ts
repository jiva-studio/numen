/**
 * Settings rows with every control type and long titles and descriptions, at a
 * wide and a narrow width. The checks need real layout, so they run in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { Button, NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from './SettingRow.vue'

const CHOICES = [
  { id: 'quick', text: 'Quick' },
  { id: 'careful', text: 'Careful' },
]

const LONG_NAME = 'Maximum number of recognition sessions kept open at the same time'
const LONG_DETAIL =
  'Parallel ONNX recognition sessions used for every scanned page of every document in the vault queue'

/** One row per control type, inside a column of the given width. */
const rows = (width: string) => () => ({
  components: { SettingRow, Select, Switch, NumberField, Slider, Button },
  setup: () => ({ choices: CHOICES, longName: LONG_NAME, longDetail: LONG_DETAIL }),
  template: `
    <div class="numen" style="container-type:inline-size;inline-size:${width};padding:1rem;background:var(--numen-surface);--settings-choice:16rem">
      <SettingRow v-slot="{ labelledBy }" at="switch" name="Nest headings under notes" detail="Show the headings of a note under its node.">
        <Switch :aria-labelledby="labelledBy" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="select" :name="longName" :detail="longDetail">
        <Select :choices="choices" :aria-labelledby="labelledBy" model-value="quick" class="settings__choice" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="text" name="Day starts at" detail="Time a review day starts. Answers before it count toward the day before.">
        <input type="text" value="04:00" :aria-labelledby="labelledBy" class="settings__text-input" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="path" name="Path to the yt-dlp program on this machine" :detail="longDetail">
        <input type="text" value="/usr/local/bin/a-very-long-folder-name/yt-dlp" :aria-labelledby="labelledBy" class="settings__text-input" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="number" :name="longName" detail="Text chunks processed in one forward pass.">
        <NumberField :model-value="8" :min="1" :max="64" :aria-labelledby="labelledBy" style="inline-size:6rem" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="range" name="Temperature" :detail="longDetail">
        <span style="display:flex;inline-size:var(--settings-choice, 16rem);max-inline-size:100%">
          <Slider :model-value="0.7" :min="0" :max="1" :step="0.1" :aria-labelledby="labelledBy" style="flex:1 1 auto" />
        </span>
      </SettingRow>
      <SettingRow v-slot="{ labelledBy }" at="secret" name="Key: apiKey" detail="Not set. Stored in the system keychain, not in the settings file.">
        <span style="display:flex;gap:0.375rem;min-inline-size:0;max-inline-size:100%">
          <input type="password" placeholder="Paste a key" :aria-labelledby="labelledBy" class="settings__text-input" style="flex:1 1 auto;min-inline-size:0" />
          <Button variant="outline" size="small">Save</Button>
        </span>
      </SettingRow>
    </div>
  `,
})

const meta: Meta<typeof SettingRow> = {
  title: 'Window/Setting row',
  component: SettingRow,
  parameters: { layout: 'fullscreen' },
  render: rows('56rem'),
}

export default meta
type Story = StoryObj<typeof meta>

/** Whether two boxes share any area. */
const isOverlapping = (a: DOMRect, b: DOMRect): boolean =>
  a.left < b.right - 0.5 &&
  b.left < a.right - 0.5 &&
  a.top < b.bottom - 0.5 &&
  b.top < a.bottom - 0.5

/** The label and control boxes of every row. */
const getRowBoxes = (root: HTMLElement) =>
  [...root.querySelectorAll<HTMLElement>('.settings__row')].map((row) => ({
    row: row.getBoundingClientRect(),
    label: row.querySelector('.settings__label')!.getBoundingClientRect(),
    value: row.querySelector('.settings__value')!.getBoundingClientRect(),
    control: row.querySelector('.settings__value')!.firstElementChild!.getBoundingClientRect(),
  }))

/** Wide: controls sit right of the text, never over it, and end at one edge. */
export const WideRowsDoNotOverlap: Story = {
  play: async ({ canvasElement }) => {
    const boxes = getRowBoxes(canvasElement)
    await expect(boxes).toHaveLength(7)
    const edge = Math.round(boxes[0]!.control.right)
    for (const one of boxes) {
      await expect(isOverlapping(one.label, one.control)).toBe(false)
      await expect(one.control.left).toBeGreaterThanOrEqual(one.label.right)
      await expect(Math.round(one.control.right)).toBe(edge)
      await expect(one.control.right).toBeLessThanOrEqual(one.row.right + 0.5)
    }
  },
}

/** Wide: text, select and path controls share one width. */
export const WideTextControlsShareOneWidth: Story = {
  play: async ({ canvasElement }) => {
    const widths = [...canvasElement.querySelectorAll('.settings__choice, .settings__text-input')]
      .slice(0, 3)
      .map((one) => Math.round(one.getBoundingClientRect().width))
    const rem = parseFloat(getComputedStyle(document.documentElement).fontSize)
    await expect(widths).toStrictEqual([16 * rem, 16 * rem, 16 * rem])
  },
}

/** Narrow: every control moves under its description at full width. */
export const NarrowRowsStackTheControl: Story = {
  render: rows('24rem'),
  play: async ({ canvasElement }) => {
    for (const one of getRowBoxes(canvasElement)) {
      await expect(isOverlapping(one.label, one.control)).toBe(false)
      await expect(one.value.top).toBeGreaterThanOrEqual(one.label.bottom)
      await expect(one.control.right).toBeLessThanOrEqual(one.row.right + 0.5)
    }
    const text = canvasElement.querySelector('.settings__text-input')!.getBoundingClientRect()
    const row = canvasElement.querySelectorAll('.settings__row')[2]!.getBoundingClientRect()
    await expect(Math.round(text.width)).toBe(Math.round(row.width))
  },
}

/** The row description renders inside the info button tooltip. */
export const LongDescriptionGoesToATooltip: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const button = canvas.getByRole('button', { name: 'About Day starts at' })
    const tooltip = canvas.getByText(
      'Time a review day starts. Answers before it count toward the day before.',
    )
    await expect(tooltip).not.toBeVisible()
    await expect(button).toHaveAccessibleDescription(
      'Time a review day starts. Answers before it count toward the day before.',
    )
    button.focus()
    await expect(tooltip).toBeVisible()
  },
}

/** Each control is labelled by the title of its row. */
export const ControlsAreLabelledByTheTitle: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('switch')[0]).toHaveAccessibleName('Nest headings under notes')
    await expect(canvas.getByDisplayValue('04:00')).toHaveAccessibleName('Day starts at')
  },
}
