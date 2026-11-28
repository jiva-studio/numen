/** The stages of a task in one fixed list: done, active and coming stages share their rows. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import StageList from './StageList.vue'
import type { Stage } from './types'

const labels = ['First sheet', 'Second sheet', 'Restore']
const make = (states: Stage['state'][]): Stage[] =>
  labels.map((label, at) => ({ id: label, label, state: states[at] ?? 'pending' }))

const meta = {
  title: 'StageList',
  component: StageList,
  parameters: { layout: 'centered' },
  args: { stages: make(['pending', 'pending', 'pending']) },
} satisfies Meta<typeof StageList>

export default meta
type Story = StoryObj<typeof meta>

export const AllPending: Story = {}
export const FirstActive: Story = { args: { stages: make(['active', 'pending', 'pending']) } }
export const SecondActive: Story = {
  args: { stages: make(['done', 'active', 'pending']) },
}
export const LastActive: Story = { args: { stages: make(['done', 'done', 'active']) } }
export const AllDone: Story = { args: { stages: make(['done', 'done', 'done']) } }
