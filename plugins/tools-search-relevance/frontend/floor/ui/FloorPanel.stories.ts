/**
 * The search relevance section of the settings, over an invented relevance service. Also the test
 * corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { CALIBRATION_OUTCOME, FLOOR_SOURCE } from '../model/codes'
import type { FloorClient, FloorState } from '../model/types'
import { WORDS } from '../lib/words'
import FloorPanel from './FloorPanel.vue'

const STATE: FloorState = {
  floor: 0.41,
  source: FLOOR_SOURCE.CALIBRATED,
  model: 'garden-embed-small',
  automaticFloor: 0.41,
  isManual: false,
  manualFloor: 0,
  calibration: {
    median: 0.27,
    spread: 0.07,
    histogram: [0, 2, 6, 11, 7, 3, 1, 0, 0, 0],
    queryCount: 200,
    chunkCount: 500,
    calibratedAt: 1_776_000_000,
  },
}

const createClient = (state: FloorState): FloorClient => ({
  getFloor: async () => state,
  calibrateFloor: async () => ({ outcome: CALIBRATION_OUTCOME.DONE, state }),
  setFloor: async (request) => ({
    state: {
      ...state,
      isManual: request.isManual,
      manualFloor: request.floor,
      floor: request.isManual ? request.floor : state.automaticFloor,
    },
  }),
  probeFloor: async () => ({ bestSimilarity: 0.46, floor: state.floor, doesPass: true }),
})

const meta = {
  title: 'Search relevance/Panel',
  component: FloorPanel,
  parameters: { layout: 'fullscreen' },
  args: { pluginId: 'md.numen.tools.search.relevance', values: {}, client: createClient(STATE) },
  render: (args) => ({
    components: { FloorPanel },
    setup: () => ({ args }),
    template: `
      <div class="numen bg-surface" style="min-block-size:100vh;padding:1.5rem">
        <div style="max-inline-size:36rem"><FloorPanel v-bind="args" /></div>
      </div>
    `,
  }),
} satisfies Meta<typeof FloorPanel>

export default meta
type Story = StoryObj<typeof meta>

/** The threshold calibrated on the vault, and a query checked against it. */
export const ChecksAQuery: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'relevance', width: 680, height: 340, drawn: 5_000 }],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText(WORDS.sourceCalibrated)).toBeVisible()
    await userEvent.type(canvas.getByRole('textbox', { name: WORDS.probeLabel }), 'raised beds')
    await userEvent.click(canvas.getByRole('button', { name: WORDS.probeButton }))
    await expect(await canvas.findByRole('status')).toHaveTextContent('Best match 0.46')
  },
}

/** Manual shows the slider for a threshold of one's own. */
export const SetByHand: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('radio', { name: WORDS.modeManual }))
    await expect(await canvas.findByText(WORDS.sliderHint)).toBeVisible()
  },
}
