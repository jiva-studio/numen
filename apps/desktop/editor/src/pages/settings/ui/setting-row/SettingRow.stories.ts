/**
 * Settings rows with every control type and long titles and descriptions, at a
 * wide and a narrow width. The checks need real layout, so they run in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import { SecretField } from '../secret-field'
import SettingRow from './SettingRow.vue'

const CHOICES = [
  { id: 'quick', text: 'Quick' },
  { id: 'careful', text: 'Careful' },
]

const DECLARATION = {
  pluginId: 'md.numen.sample',
  pluginName: 'Sample',
  name: 'apiKey',
  group: 'openrouter',
  title: 'API key',
  description: 'The key of the service.',
}

const SECTIONS = [
  {
    group: 'openrouter',
    isAsked: true,
    secrets: [
      {
        id: 's00000001',
        name: 'Work',
        group: 'openrouter',
        allowed: [{ pluginId: 'md.numen.sample', name: 'apiKey', group: 'openrouter' }],
        hasValue: true,
      },
    ],
  },
]

const MENUS = [
  {
    pluginId: 'md.numen.sample',
    name: 'apiKey',
    group: 'openrouter',
    secrets: SECTIONS[0]?.secrets ?? [],
  },
]

const LONG_NAME = 'Maximum number of recognition sessions kept open at the same time'
const LONG_DETAIL =
  'Parallel ONNX recognition sessions used for every scanned page of every document in the vault queue'

const rows = (width: string) => () => ({
  components: { SettingRow, SecretField, Select, Switch, NumberField, Slider },
  setup: () => ({
    choices: CHOICES,
    longName: LONG_NAME,
    longDetail: LONG_DETAIL,
    declaration: DECLARATION,
    sections: SECTIONS,
    menus: MENUS,
  }),
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
      <SecretField at="secret" name="Key: apiKey" detail="The key of the service." :declaration="declaration" :sections="sections" :menus="menus" />
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
