/**
 * One row of the settings, drawn on its own.
 *
 * What is asked here is what only a browser can answer: that controls of every
 * width end at the one edge, and that a name too long for its half of the row
 * does not push the control off it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { Select, Switch } from '@numen/ui'
import SettingRow from './SettingRow.vue'

const CHOICES = [
  { id: 'quick', text: 'Quick' },
  { id: 'careful', text: 'Careful' },
]

const UNBROKEN = 'A setting nobody gave a short name to and whose name runs on past the row'

/** A switch and a list of choices, each on a row of its own. */
const rows = () => ({
  components: { SettingRow, Select, Switch },
  setup: () => ({ choices: CHOICES, unbroken: UNBROKEN }),
  template: `
    <div class="numen" style="inline-size:40rem;padding:1rem;background:var(--numen-surface)">
      <SettingRow
        v-slot="{ labelledBy }"
        at="nesting"
        name="Nest headings under notes"
        detail="The headings of a note, under the box that draws it"
      >
        <Switch :aria-labelledby="labelledBy" />
      </SettingRow>
      <SettingRow
        v-slot="{ labelledBy }"
        at="proofread"
        name="Put a transcript right"
        detail="The profile a recording is read back against"
      >
        <Select
          :choices="choices"
          :aria-labelledby="labelledBy"
          model-value="quick"
          style="inline-size:18rem"
        />
      </SettingRow>
      <SettingRow
        v-slot="{ labelledBy }"
        at="unbroken"
        :name="unbroken"
        :detail="unbroken"
      >
        <Switch :aria-labelledby="labelledBy" />
      </SettingRow>
    </div>
  `,
})

const meta: Meta<typeof SettingRow> = {
  title: 'Window/Setting row',
  component: SettingRow,
  parameters: { layout: 'fullscreen' },
  render: rows,
}

export default meta
type Story = StoryObj<typeof meta>

/** The rows drawn, whatever control each carries. */
export const EveryControlAtTheOneEdge: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const switches = canvas.getAllByRole('switch')
    const list = canvas.getByRole('button')

    // Each control is announced by the name on its own row.
    await expect(switches[0]).toHaveAccessibleName('Nest headings under notes')
    await expect(list).toHaveAccessibleName('Put a transcript right')

    // A switch and a list far wider than it end at the same edge.
    const switchRect = switches[0]!.getBoundingClientRect()
    const listRect = list.getBoundingClientRect()
    await expect(Math.round(listRect.right)).toBe(Math.round(switchRect.right))
    await expect(listRect.width).toBeGreaterThan(switchRect.width)

    // A name too long for its half of the row leaves the control where it was.
    const last = switches[1]!.getBoundingClientRect()
    await expect(Math.round(last.right)).toBe(Math.round(switchRect.right))
  },
}

/** What a row is called, and under it what it means, read as two things. */
export const TheNameStandsOutFromWhatItMeans: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const name = canvas.getByText('Nest headings under notes')
    const detail = canvas.getByText('The headings of a note, under the box that draws it')

    await expect(getComputedStyle(name).fontWeight).toBe('500')
    await expect(getComputedStyle(name).color).not.toBe(getComputedStyle(detail).color)
  },
}

/** A row whose control is wide, drawn at the width of a narrow window and of a wide one. */
const wideControl = (width: string) => () => ({
  components: { SettingRow },
  template: `
    <div class="numen" style="inline-size:${width};padding:1rem;background:var(--numen-surface)">
      <SettingRow
        v-slot="{ labelledBy }"
        at="key"
        name="Sample Agent: apiKey"
        detail="Set. Stored in the system keychain and never shown again"
      >
        <input :aria-labelledby="labelledBy" style="inline-size:24rem;max-inline-size:100%" />
      </SettingRow>
    </div>
  `,
})

/** Beside the text when there is room, and the text keeps its width. */
export const WideControlBesideTheText: Story = {
  render: wideControl('56rem'),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const name = canvas.getByText('Sample Agent: apiKey').getBoundingClientRect()
    const control = canvas.getByRole('textbox').getBoundingClientRect()

    await expect(control.left).toBeGreaterThanOrEqual(name.right)
    await expect(control.width).toBeLessThanOrEqual(24 * 16 + 1)
    await expect(name.width).toBeGreaterThanOrEqual(18 * 16 - 1)
  },
}

/** Under the text when there is not, and the text keeps its width. */
export const WideControlUnderTheText: Story = {
  render: wideControl('26rem'),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const name = canvas.getByText('Sample Agent: apiKey').getBoundingClientRect()
    const control = canvas.getByRole('textbox').getBoundingClientRect()

    await expect(control.top).toBeGreaterThanOrEqual(name.bottom)
    await expect(name.width).toBeGreaterThanOrEqual(18 * 16 - 1)
  },
}
