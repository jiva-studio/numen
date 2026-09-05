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
        at="hanging"
        name="Hang the parts of a note"
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
    await expect(switches[0]).toHaveAccessibleName('Hang the parts of a note')
    await expect(list).toHaveAccessibleName('Put a transcript right')

    // A switch and a list far wider than it end at the same edge.
    const narrow = switches[0]!.getBoundingClientRect()
    const wide = list.getBoundingClientRect()
    await expect(Math.round(wide.right)).toBe(Math.round(narrow.right))
    await expect(wide.width).toBeGreaterThan(narrow.width)

    // A name too long for its half of the row leaves the control where it was.
    const last = switches[1]!.getBoundingClientRect()
    await expect(Math.round(last.right)).toBe(Math.round(narrow.right))
  },
}
