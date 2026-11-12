/**
 * Settings rows with every control type and long titles and descriptions, at a
 * wide and a narrow width. The checks need real layout, so they run in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from './SettingRow.vue'

const CHOICES = [
  { id: 'quick', text: 'Quick' },
  { id: 'careful', text: 'Careful' },
]

const LONG_NAME = 'Maximum number of recognition sessions kept open at the same time'
const LONG_DETAIL =
  'Parallel ONNX recognition sessions used for every scanned page of every document in the vault queue'

const rows = (width: string) => () => ({
  components: { SettingRow, Select, Switch, NumberField, Slider },
  setup: () => ({
    choices: CHOICES,
    longName: LONG_NAME,
    longDetail: LONG_DETAIL,
  }),
  template: `
    <div class="numen" style="container:settings / inline-size;inline-size:${width};padding:1rem;background:var(--numen-surface)">
      <SettingRow v-slot="{ labelledBy, describedBy }" at="switch" name="Nest headings under notes" detail="Show the headings of a note under its node.">
        <Switch :aria-labelledby="labelledBy" :aria-describedby="describedBy" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy, describedBy }" at="select" :name="longName" :detail="longDetail">
        <Select :choices="choices" :aria-labelledby="labelledBy" :aria-describedby="describedBy" model-value="quick" class="settings__choice" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy, describedBy }" at="text" name="Day starts at" detail="Time a review day starts. Answers before it count toward the day before.">
        <input type="text" value="04:00" :aria-labelledby="labelledBy" :aria-describedby="describedBy" class="settings__text-input" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy, describedBy }" at="path" name="Path to the yt-dlp program on this machine" :detail="longDetail">
        <input type="text" value="/usr/local/bin/a-very-long-folder-name/yt-dlp" :aria-labelledby="labelledBy" :aria-describedby="describedBy" class="settings__text-input" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy, describedBy }" at="number" :name="longName" detail="Text chunks processed in one forward pass.">
        <NumberField :model-value="8" :min="1" :max="64" :aria-labelledby="labelledBy" :aria-describedby="describedBy" style="inline-size:6rem" />
      </SettingRow>
      <SettingRow v-slot="{ labelledBy, describedBy }" at="range" name="Temperature" :detail="longDetail">
        <span style="display:flex;inline-size:var(--settings-choice);max-inline-size:100%">
          <Slider :model-value="0.7" :min="0" :max="1" :step="0.1" :aria-labelledby="labelledBy" :aria-describedby="describedBy" style="flex:1 1 auto" />
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

/** The description is cut to one line in the row, and the control is described by all of it. */
export const ControlsAreDescribedByTheDescription: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByDisplayValue('04:00')).toHaveAccessibleDescription(
      'Time a review day starts. Answers before it count toward the day before.',
    )
    await expect(canvas.getAllByRole('switch')[0]).toHaveAccessibleDescription(
      'Show the headings of a note under its node.',
    )
  },
}

export const ControlsAreLabelledByTheTitle: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('switch')[0]).toHaveAccessibleName('Nest headings under notes')
    await expect(canvas.getByDisplayValue('04:00')).toHaveAccessibleName('Day starts at')
  },
}
